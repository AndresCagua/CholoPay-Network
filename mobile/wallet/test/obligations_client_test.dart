import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:cholopay_wallet/obligations_client.dart';

class FakeTransport implements WalletHttpTransport {
  FakeTransport(this.response);

  final WalletHttpResponse response;
  String? method;
  Uri? uri;
  Map<String, String>? headers;
  String? body;

  @override
  Future<WalletHttpResponse> send(
    String requestMethod,
    Uri requestUri, {
    Map<String, String> headers = const {},
    String? body,
  }) async {
    method = requestMethod;
    uri = requestUri;
    this.headers = headers;
    this.body = body;
    return response;
  }
}

void main() {
  test('consulta participant-scoped usa cliente HTTP configurable', () async {
    final transport = FakeTransport(WalletHttpResponse(
      200,
      jsonEncode({
        'obligations': [
          {
            'id': 'bill-light-001',
            'participantId': 'COSTA',
            'kind': 'electricity',
            'description': 'Luz sintética',
            'amountMinor': 1800,
            'currency': 'USD',
            'status': 'OPEN',
          },
        ],
      }),
    ));
    final client = HttpObligationsClient(
      baseUri: Uri.parse('https://gateway.synthetic.test/'),
      transport: transport,
    );

    final obligations = await client.fetchObligations(participantId: 'COSTA');

    expect(transport.method, 'GET');
    expect(transport.uri!.path, '/v1/participants/COSTA/obligations');
    expect(obligations.single.participantId, 'COSTA');
    expect(obligations.single.amountMinor, 1800);
  });

  test('inicia pago participant-scoped sin acceder a una base', () async {
    final transport = FakeTransport(WalletHttpResponse(
      201,
      jsonEncode({
        'paymentIntent': {
          'id': 'intent-001',
          'obligationId': 'bill-light-001',
          'state': 'PENDING',
        },
      }),
    ));
    final client = HttpObligationsClient(
      baseUri: Uri.parse('https://gateway.synthetic.test'),
      bankCoreBaseUri: Uri.parse('https://bank-core.synthetic.test'),
      transport: transport,
    );

    final intent = await client.startPayment(
      participantId: 'COSTA',
      obligationId: 'bill-light-001',
      idempotencyKey: 'wallet-obligation-001',
      service: 'electricity',
      invoiceId: 'bill-light-001',
    );

    expect(transport.method, 'POST');
    expect(transport.uri!.path,
        '/v1/participants/COSTA/obligations/bill-light-001/payment-intents');
    expect(transport.headers!['Idempotency-Key'], 'wallet-obligation-001');
    final payload = jsonDecode(transport.body!) as Map<String, dynamic>;
    expect(payload['participantId'], 'COSTA');
    expect(payload['sourceAccountId'], '00000000-0000-4000-8000-000000000001');
    expect(payload['service'], 'electricity');
    expect(payload['invoiceId'], 'bill-light-001');
    expect(payload['amount'], '0.00');
    expect(payload['currency'], 'USD');
    expect(transport.uri!.host, 'bank-core.synthetic.test');
    expect(intent.state, 'PENDING');
  });

  test('envía tránsito al banco con moneda y servicio sintéticos', () async {
    final transport = FakeTransport(WalletHttpResponse(
      202,
      jsonEncode({'status': 'PENDING_BILLER', 'obligation_id': 'registration-001'}),
    ));
    final client = HttpObligationsClient(
      baseUri: Uri.parse('https://gateway.synthetic.test'),
      bankCoreBaseUri: Uri.parse('https://bank-core.synthetic.test'),
      transport: transport,
    );

    await client.startPayment(
      participantId: 'COSTA',
      obligationId: 'registration-001',
      idempotencyKey: 'wallet-transit-001',
      amount: '32.00',
      service: 'registration',
      currency: 'SYN',
    );

    final payload = jsonDecode(transport.body!) as Map<String, dynamic>;
    expect(payload['biller'], 'transit');
    expect(payload['service'], 'registration');
    expect(payload['invoiceId'], isNull);
    expect(payload['amount'], '32.00');
    expect(payload['currency'], 'SYN');
  });

  test('rechaza un servicio ausente en lugar de inventarlo desde el ID', () async {
    final client = HttpObligationsClient(
      baseUri: Uri.parse('https://gateway.synthetic.test'),
      bankCoreBaseUri: Uri.parse('https://bank-core.synthetic.test'),
      transport: FakeTransport(const WalletHttpResponse(201, '{}')),
    );

    expect(
      () => client.startPayment(
        participantId: 'COSTA',
        obligationId: 'bill-light-001',
        idempotencyKey: 'wallet-missing-service',
      ),
      throwsArgumentError,
    );
  });

  test('rechaza respuestas HTTP no exitosas', () async {
    final client = HttpObligationsClient(
      baseUri: Uri.parse('https://gateway.synthetic.test'),
      transport: FakeTransport(const WalletHttpResponse(503, 'caído')),
    );

    expect(
      () => client.fetchObligations(participantId: 'COSTA'),
      throwsA(isA<WalletHttpException>()),
    );
  });

  test('fixture sintético mantiene estados de consulta y pago', () async {
    const client = SyntheticObligationsClient(obligations: syntheticObligations);

    expect((await client.fetchObligations(participantId: 'COSTA')).length, 2);
    expect(
      (await client.startPayment(
        participantId: 'COSTA',
        obligationId: 'bill-light-001',
        idempotencyKey: 'key-001',
      )).state,
      'PENDING',
    );
  });
}

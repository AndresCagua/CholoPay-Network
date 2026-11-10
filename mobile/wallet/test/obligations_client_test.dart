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
      transport: transport,
    );

    final intent = await client.startPayment(
      participantId: 'COSTA',
      obligationId: 'bill-light-001',
      idempotencyKey: 'wallet-obligation-001',
    );

    expect(transport.method, 'POST');
    expect(transport.uri!.path,
        '/v1/participants/COSTA/obligations/bill-light-001/payment-intents');
    expect(transport.headers!['Idempotency-Key'], 'wallet-obligation-001');
    expect(jsonDecode(transport.body!)['participantId'], 'COSTA');
    expect(intent.state, 'PENDING');
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

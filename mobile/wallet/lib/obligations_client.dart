import 'dart:convert';
import 'dart:io';

class Obligation {
  const Obligation({
    required this.id,
    required this.participantId,
    required this.kind,
    required this.description,
    required this.amountMinor,
    required this.currency,
    required this.status,
  });

  final String id;
  final String participantId;
  final String kind;
  final String description;
  final int amountMinor;
  final String currency;
  final String status;

  factory Obligation.fromJson(Map<String, dynamic> json) => Obligation(
        id: json['id'] as String,
        participantId: json['participantId'] as String,
        kind: json['kind'] as String,
        description: json['description'] as String,
        amountMinor: json['amountMinor'] as int,
        currency: json['currency'] as String,
        status: json['status'] as String,
      );
}

class PaymentIntent {
  const PaymentIntent({
    required this.id,
    required this.obligationId,
    required this.state,
  });

  final String id;
  final String obligationId;
  final String state;

  factory PaymentIntent.fromJson(Map<String, dynamic> json) => PaymentIntent(
        id: json['id'] as String,
        obligationId: json['obligationId'] as String,
        state: json['state'] as String,
      );
}

abstract interface class ObligationsClient {
  Future<List<Obligation>> fetchObligations({required String participantId});

  Future<PaymentIntent> startPayment({
    required String participantId,
    required String obligationId,
    required String idempotencyKey,
    String? amount,
  });
}

class WalletHttpResponse {
  const WalletHttpResponse(this.statusCode, this.body);

  final int statusCode;
  final String body;
}

abstract interface class WalletHttpTransport {
  Future<WalletHttpResponse> send(
    String method,
    Uri uri, {
    Map<String, String> headers,
    String? body,
  });
}

class DartIoWalletHttpTransport implements WalletHttpTransport {
  DartIoWalletHttpTransport({HttpClient? httpClient})
      : _httpClient = httpClient ?? HttpClient();

  final HttpClient _httpClient;

  @override
  Future<WalletHttpResponse> send(
    String method,
    Uri uri, {
    Map<String, String> headers = const {},
    String? body,
  }) async {
    final request = await _httpClient.openUrl(method, uri);
    headers.forEach(request.headers.set);
    if (body != null) {
      request.headers.contentType = ContentType.json;
      request.write(body);
    }
    final response = await request.close();
    return WalletHttpResponse(response.statusCode, await response.transform(utf8.decoder).join());
  }
}

class WalletHttpException implements Exception {
  const WalletHttpException(this.statusCode, this.body);

  final int statusCode;
  final String body;

  @override
  String toString() => 'WalletHttpException($statusCode): $body';
}

class HttpObligationsClient implements ObligationsClient {
  HttpObligationsClient({
    required Uri baseUri,
    this.authorizationToken,
    this.sourceAccountId = 'account-costa-001',
    WalletHttpTransport? transport,
  })  : _baseUri = baseUri,
        _transport = transport ?? DartIoWalletHttpTransport();

  final Uri _baseUri;
  final WalletHttpTransport _transport;
  final String? authorizationToken;
  final String sourceAccountId;

  Uri _uri(String path) => _baseUri.resolve(path);

  Future<Map<String, dynamic>> _request(
    String method,
    Uri uri, {
    Map<String, String> headers = const {},
    String? body,
  }) async {
    final response = await _transport.send(method, uri, headers: headers, body: body);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw WalletHttpException(response.statusCode, response.body);
    }
    return jsonDecode(response.body) as Map<String, dynamic>;
  }

  @override
  Future<List<Obligation>> fetchObligations({required String participantId}) async {
    final json = await _request(
      'GET',
      _uri('/v1/participants/$participantId/obligations'),
    );
    return (json['obligations'] as List<dynamic>)
        .map((item) => Obligation.fromJson(item as Map<String, dynamic>))
        .where((item) => item.participantId == participantId)
        .toList(growable: false);
  }

  @override
  Future<PaymentIntent> startPayment({
    required String participantId,
    required String obligationId,
    required String idempotencyKey,
    String? amount,
  }) async {
    final headers = <String, String>{'Idempotency-Key': idempotencyKey};
    if (authorizationToken != null && authorizationToken!.isNotEmpty) headers['Authorization'] = 'Bearer $authorizationToken';
    final json = await _request(
      'POST',
      _uri('/v1/participants/$participantId/obligations/$obligationId/payment-intents'),
      headers: headers,
      body: jsonEncode({'participantId': participantId, 'participant_id': participantId, 'source_account_id': sourceAccountId, 'biller': 'utilities', 'amount': amount ?? '0.00', 'currency': 'USD'}),
    );
    final payload = (json['paymentIntent'] as Map<String, dynamic>?) ?? json;
    return PaymentIntent.fromJson({
      'id': payload['id'],
      'obligationId': payload['obligationId'] ?? payload['obligation_id'] ?? obligationId,
      'state': payload['state'] ?? payload['status'] ?? 'PENDING',
    });
  }
}

const syntheticObligations = [
  Obligation(
    id: 'bill-light-001',
    participantId: 'COSTA',
    kind: 'electricity',
    description: 'Luz sintética',
    amountMinor: 1800,
    currency: 'USD',
    status: 'OPEN',
  ),
  Obligation(
    id: 'obligation-registration-001',
    participantId: 'COSTA',
    kind: 'registration',
    description: 'Matrícula sintética',
    amountMinor: 3200,
    currency: 'USD',
    status: 'OPEN',
  ),
];

class SyntheticObligationsClient implements ObligationsClient {
  const SyntheticObligationsClient({
    this.obligations = syntheticObligations,
    this.failure,
  });

  final List<Obligation> obligations;
  final Exception? failure;

  @override
  Future<List<Obligation>> fetchObligations({required String participantId}) async {
    if (failure != null) throw failure!;
    return obligations.where((item) => item.participantId == participantId).toList(growable: false);
  }

  @override
  Future<PaymentIntent> startPayment({
    required String participantId,
    required String obligationId,
    required String idempotencyKey,
    String? amount,
  }) async {
    if (failure != null) throw failure!;
    final exists = obligations.any((item) => item.id == obligationId && item.participantId == participantId);
    if (!exists) throw StateError('Obligación fuera del participante');
    return PaymentIntent(
      id: 'intent-$idempotencyKey',
      obligationId: obligationId,
      state: 'PENDING',
    );
  }
}

import { createHmac, timingSafeEqual } from 'node:crypto';
import { createServer as createHttpServer, type IncomingMessage, type Server, type ServerResponse } from 'node:http';

export type Scope = 'payment:read' | 'payment:write' | 'webhook:write';
type PaymentIntentStatus = 'requires_confirmation' | 'succeeded';
type BillPayment = {
  id: string;
  object: 'bill_payment';
  participant_id: string;
  biller: 'utilities' | 'transit';
  invoice_id?: string;
  obligation_id?: string;
  amount: string;
  currency: string;
  status: 'APPLIED' | 'PENDING_BILLER' | 'PENDING_APPLICATION';
  payment_intent_id?: string;
  ledger_boundary: string;
};
type PaymentIntent = {
  id: string;
  object: 'payment_intent';
  amount_minor: number;
  currency: string;
  merchant_order_id: string;
  description?: string;
  status: PaymentIntentStatus;
  ledger_boundary: string;
};
type Refund = {
  id: string;
  object: 'refund';
  payment_intent_id: string;
  amount_minor: number;
  currency: string;
  status: 'succeeded';
  ledger_boundary: string;
};
type IdempotentResponse = { payload: string; status: number; body: Record<string, unknown> };
type WebhookEndpoint = { id: string; url: string; secret: string; replayedSignatures: Set<string> };
type WebhookDelivery = { endpointId: string; eventId: string; attempt: number; event: Record<string, unknown>; deadLettered?: boolean };
const MAX_WEBHOOK_ATTEMPTS = 3;
const MAX_DLQ_ENTRIES = 100;
const WEBHOOK_TIMESTAMP_TOLERANCE_SECONDS = 300;

const syntheticApiKeys = new Map<string, ReadonlySet<Scope>>([
  ['sk_sandbox_merchant_rw', new Set<Scope>(['payment:read', 'payment:write', 'webhook:write'])],
  ['sk_sandbox_merchant_ro', new Set<Scope>(['payment:read'])],
]);

const placeholderRoutes: Array<{ method: string; pattern: RegExp; scope: Scope; route: string; authRequired?: boolean }> = [
  { method: 'POST', pattern: /^\/v1\/bill-payments$/, scope: 'payment:write', route: 'POST /v1/bill-payments' },
  { method: 'GET', pattern: /^\/v1\/bill-payments\/[^/]+$/, scope: 'payment:read', route: 'GET /v1/bill-payments/{id}' },
  { method: 'POST', pattern: /^\/v1\/bill-payments\/[^/]+\/apply$/, scope: 'payment:write', route: 'POST /v1/bill-payments/{id}/apply' },
  { method: 'POST', pattern: /^\/v1\/payment-intents$/, scope: 'payment:write', route: 'POST /v1/payment-intents' },
  { method: 'GET', pattern: /^\/v1\/payment-intents\/[^/]+$/, scope: 'payment:read', route: 'GET /v1/payment-intents/{id}' },
  { method: 'POST', pattern: /^\/v1\/payment-intents\/[^/]+\/confirm$/, scope: 'payment:write', route: 'POST /v1/payment-intents/{id}/confirm' },
  { method: 'POST', pattern: /^\/v1\/refunds$/, scope: 'payment:write', route: 'POST /v1/refunds' },
  { method: 'POST', pattern: /^\/v1\/webhook-endpoints$/, scope: 'webhook:write', route: 'POST /v1/webhook-endpoints' },
  { method: 'POST', pattern: /^\/v1\/webhook-endpoints\/[^/]+\/events$/, scope: 'webhook:write', route: 'POST /v1/webhook-endpoints/{id}/events', authRequired: false },
  { method: 'POST', pattern: /^\/v1\/webhook-endpoints\/[^/]+\/deliveries$/, scope: 'webhook:write', route: 'POST /v1/webhook-endpoints/{id}/deliveries' },
];

function sendJson(response: ServerResponse, status: number, body: Record<string, unknown>): void {
  response.writeHead(status, { 'content-type': 'application/json; charset=utf-8' });
  response.end(JSON.stringify(body));
}

function errorBody(error: string, message: string): Record<string, unknown> {
  return { error, message };
}

function apiKeyFrom(request: IncomingMessage): string | undefined {
  const value = request.headers['x-api-key'];
  return Array.isArray(value) ? value[0] : value;
}

function idempotencyKeyFrom(request: IncomingMessage): string | undefined {
  const value = request.headers['idempotency-key'];
  return Array.isArray(value) ? value[0] : value;
}

function authorize(request: IncomingMessage, requiredScope: Scope): 'missing' | 'forbidden' | 'ok' {
  const key = apiKeyFrom(request);
  if (!key || !syntheticApiKeys.has(key)) return 'missing';
  return syntheticApiKeys.get(key)?.has(requiredScope) ? 'ok' : 'forbidden';
}

function canonicalize(value: unknown): string {
  if (Array.isArray(value)) return `[${value.map(canonicalize).join(',')}]`;
  if (value !== null && typeof value === 'object') {
    return `{${Object.entries(value as Record<string, unknown>).sort(([a], [b]) => a.localeCompare(b)).map(([key, item]) => `${JSON.stringify(key)}:${canonicalize(item)}`).join(',')}}`;
  }
  return JSON.stringify(value);
}

function readRawBody(request: IncomingMessage): Promise<string> {
  return new Promise((resolve, reject) => {
    let content = '';
    request.setEncoding('utf8');
    request.on('data', (chunk: string) => { content += chunk; });
    request.on('end', () => resolve(content));
    request.on('error', reject);
  });
}

async function readJsonBody(request: IncomingMessage): Promise<unknown> {
  const content = await readRawBody(request);
  if (!content.trim()) return {};
  try { return JSON.parse(content); } catch { throw new Error('invalid_json'); }
}

function webhookSignatureFrom(request: IncomingMessage): string | undefined {
  const value = request.headers['webhook-signature'];
  return Array.isArray(value) ? value[0] : value;
}

function verifyWebhookSignature(header: string | undefined, secret: string, rawBody: string): { timestamp: number; signature: string } | undefined {
  const match = header?.match(/^t=(\d+),v1=([a-f0-9]{64})$/);
  if (!match) return undefined;
  const timestamp = Number(match[1]);
  const signature = match[2];
  if (!signature || !Number.isSafeInteger(timestamp) || Math.abs(Math.floor(Date.now() / 1000) - timestamp) > WEBHOOK_TIMESTAMP_TOLERANCE_SECONDS) return undefined;
  const expected = createHmac('sha256', secret).update(`${timestamp}.${rawBody}`).digest('hex');
  const supplied = Buffer.from(signature, 'hex');
  const expectedBytes = Buffer.from(expected, 'hex');
  if (supplied.length !== expectedBytes.length || !timingSafeEqual(supplied, expectedBytes)) return undefined;
  return { timestamp, signature };
}

export type GatewayOptions = { billerBaseUrl?: string; utilitiesBaseUrl?: string; transitBaseUrl?: string; billerEventSecret?: string };

export function createServer(options: GatewayOptions = {}): Server {
  const intents = new Map<string, PaymentIntent>();
  const refunds = new Map<string, Refund>();
  const refundByIntent = new Map<string, Refund>();
  const billPayments = new Map<string, BillPayment>();
  const billPaymentRequests = new Map<string, { payload: string; payment: BillPayment; request: Record<string, unknown> }>();
  const idempotency = new Map<string, IdempotentResponse>();
  let nextIntentId = 1;
  let nextRefundId = 1;
  let nextBillPaymentId = 1;
  let nextWebhookEndpointId = 1;
  const utilitiesBaseUrl = options.utilitiesBaseUrl ?? options.billerBaseUrl ?? process.env.UTILITIES_URL ?? process.env.BILLER_URL ?? 'http://utilities:8080';
  const transitBaseUrl = options.transitBaseUrl ?? process.env.TRANSIT_URL ?? 'http://transit:8082';
  const billerEventSecret = options.billerEventSecret ?? process.env.INTERNAL_EVENT_SECRET ?? 'local-test-only';
  const webhookEndpoints = new Map<string, WebhookEndpoint>();
  const webhookDeliveries = new Map<string, WebhookDelivery>();
  const deadLetterQueue: WebhookDelivery[] = [];
  const ledgerBoundary = 'Esta pasarela nunca escribe saldos ni diarios.';

  return createHttpServer(async (request, response) => {
    const method = request.method ?? 'GET';
    const pathname = new URL(request.url ?? '/', 'http://gateway.local').pathname;

    if (method === 'GET' && pathname === '/health/live') {
      sendJson(response, 200, { status: 'ok', service: 'cholopay-gateway' });
      return;
    }

    const route = placeholderRoutes.find((candidate) => candidate.method === method && candidate.pattern.test(pathname));
    if (!route) {
      sendJson(response, 404, errorBody('not_found', 'Ruta no encontrada'));
      return;
    }

    if (route.authRequired !== false) {
      const authorization = authorize(request, route.scope);
      if (authorization === 'missing') {
        sendJson(response, 401, errorBody('unauthorized', 'Se requiere una clave API válida del entorno aislado'));
        return;
      }
      if (authorization === 'forbidden') {
        sendJson(response, 403, errorBody('forbidden', `La clave API no tiene el alcance ${route.scope}`));
        return;
      }
    }

    const segments = pathname.split('/');
    const endpointId = segments[3] ?? '';
    const endpoint = webhookEndpoints.get(endpointId);
    if (route.route === 'GET /v1/bill-payments/{id}') {
      const payment = billPayments.get(segments[3] ?? '');
      if (!payment) return sendJson(response, 404, errorBody('not_found', 'Pago de factura no encontrado'));
      sendJson(response, 200, payment);
      return;
    }

    if (route.route === 'POST /v1/bill-payments' || route.route === 'POST /v1/bill-payments/{id}/apply') {
      const isApply = route.route.endsWith('/apply');
      const key = idempotencyKeyFrom(request);
      if (!key) return sendJson(response, 400, errorBody('invalid_request', 'Se requiere la cabecera Idempotency-Key'));
      let body: unknown;
      try { body = await readJsonBody(request); } catch { return sendJson(response, 400, errorBody('invalid_request', 'El cuerpo de la solicitud debe ser JSON válido')); }
      let payment: BillPayment;
      let input: Record<string, unknown>;
      if (isApply) {
        const existing = billPayments.get(segments[3] ?? '');
        const saved = existing && [...billPaymentRequests.values()].find((entry) => entry.payment.id === existing.id);
        if (!existing || !saved) return sendJson(response, 404, errorBody('not_found', 'Pago de factura no encontrado'));
        payment = existing;
        input = saved.request;
      } else {
        if (!body || typeof body !== 'object' || Array.isArray(body)) return sendJson(response, 400, errorBody('invalid_request', 'El cuerpo de la solicitud debe ser un objeto'));
        input = body as Record<string, unknown>;
        const biller = input.biller === 'transit' ? 'transit' : 'utilities';
        const validUtilities = biller === 'utilities'
          && typeof input.invoice_id === 'string'
          && ['electricity', 'water', 'internet', 'mobile'].includes(String(input.service))
          && input.currency === 'USD';
        const validTransit = biller === 'transit'
          && typeof input.obligation_id === 'string'
          && input.currency === 'SYN';
        if (typeof input.participant_id !== 'string' || (!validUtilities && !validTransit) || typeof input.amount !== 'string') {
          return sendJson(response, 422, errorBody('invalid_request', 'La obligación sintética no es compatible con el facturador seleccionado'));
        }
        const payload = canonicalize(input);
        const previous = billPaymentRequests.get(key);
        if (previous) {
          if (previous.payload !== payload) return sendJson(response, 409, errorBody('idempotency_key_conflict', 'Idempotency-Key ya se usó con una carga diferente'));
          return sendJson(response, previous.payment.status === 'APPLIED' ? 201 : 202, previous.payment);
        }
        payment = { id: `bp_sandbox_${String(nextBillPaymentId++).padStart(8, '0')}`, object: 'bill_payment', participant_id: input.participant_id as string, biller, invoice_id: validUtilities ? input.invoice_id as string : undefined, obligation_id: validTransit ? input.obligation_id as string : undefined, amount: input.amount as string, currency: input.currency as string, status: 'PENDING_BILLER', ledger_boundary: ledgerBoundary };
        billPayments.set(payment.id, payment);
        billPaymentRequests.set(key, { payload, payment, request: input });
      }
      const billerKey = `bill-payment:${key}`;
      const billerBaseUrl = payment.biller === 'transit' ? transitBaseUrl : utilitiesBaseUrl;
      try {
        const intentPath = payment.biller === 'transit'
          ? `/v1/obligations/${encodeURIComponent(payment.obligation_id ?? '')}/payment-intents`
          : '/api/v1/payment-intents';
        const intentResponse = await fetch(`${billerBaseUrl}${intentPath}`, { method: 'POST', headers: { 'content-type': 'application/json', 'idempotency-key': billerKey }, body: JSON.stringify(payment.biller === 'transit' ? { amount_minor: Math.round(Number(payment.amount) * 100), currency: payment.currency } : { invoice_id: payment.invoice_id, amount: payment.amount, currency: payment.currency }), signal: AbortSignal.timeout(1000) });
        if (!intentResponse.ok) { payment.status = 'PENDING_BILLER'; return sendJson(response, 202, payment); }
        const intent = await intentResponse.json() as { id?: string };
        if (!intent.id) { payment.status = 'PENDING_APPLICATION'; return sendJson(response, 202, payment); }
        const event = payment.biller === 'transit'
          ? { event_type: 'payment.settled', obligation_id: payment.obligation_id, payment_intent_id: intent.id }
          : { event_type: 'payment.settled', invoice_id: payment.invoice_id, payment_intent_id: intent.id };
        const timestamp = String(Math.floor(Date.now() / 1000));
        const canonical = canonicalize(event);
        const signature = createHmac('sha256', billerEventSecret).update(`${timestamp}.${canonical}`).digest('hex');
        const eventPath = payment.biller === 'transit' ? '/v1/internal/payment-events' : '/api/v1/internal/payment-events';
        const eventResponse = await fetch(`${billerBaseUrl}${eventPath}`, { method: 'POST', headers: { 'content-type': 'application/json', 'x-cholopay-event-id': payment.id, 'x-cholopay-timestamp': timestamp, 'x-cholopay-signature': signature }, body: JSON.stringify(event), signal: AbortSignal.timeout(1000) });
        payment.payment_intent_id = intent.id;
        payment.status = eventResponse.ok ? 'APPLIED' : 'PENDING_APPLICATION';
      } catch { payment.status = 'PENDING_BILLER'; }
      sendJson(response, payment.status === 'APPLIED' ? 201 : 202, payment);
      return;
    }

    if (route.route === 'GET /v1/payment-intents/{id}') {
      const intent = intents.get(segments[3] ?? '');
      if (!intent) {
        sendJson(response, 404, errorBody('not_found', 'Intención de pago no encontrada'));
        return;
      }
      sendJson(response, 200, intent);
      return;
    }

    if (route.route === 'POST /v1/webhook-endpoints') {
      let body: unknown;
      try { body = await readJsonBody(request); } catch {
        sendJson(response, 400, errorBody('invalid_request', 'El cuerpo de la solicitud debe ser JSON válido'));
        return;
      }
      const input = body as Record<string, unknown>;
      if (!body || typeof body !== 'object' || Array.isArray(body) || typeof input.url !== 'string' || !/^https?:\/\/[^\s]+$/.test(input.url)) {
        sendJson(response, 400, errorBody('invalid_request', 'url debe ser un endpoint HTTP(S) sintético válido'));
        return;
      }
      const id = `we_sandbox_${String(nextWebhookEndpointId).padStart(8, '0')}`;
      const secret = `whsec_sandbox_${String(nextWebhookEndpointId++).padStart(8, '0')}`;
      webhookEndpoints.set(id, { id, url: input.url, secret, replayedSignatures: new Set() });
      sendJson(response, 201, { id, object: 'webhook_endpoint', url: input.url, status: 'enabled', secret, ledger_boundary: ledgerBoundary });
      return;
    }

    if (route.route === 'POST /v1/webhook-endpoints/{id}/events') {
      if (!endpoint) {
        sendJson(response, 404, errorBody('not_found', 'Endpoint de webhook no encontrado'));
        return;
      }
      const rawBody = await readRawBody(request);
      const verified = verifyWebhookSignature(webhookSignatureFrom(request), endpoint.secret, rawBody);
      if (!verified) {
        sendJson(response, 401, errorBody('invalid_signature', 'La firma del webhook no es válida o ha caducado'));
        return;
      }
      const replayKey = `${verified.timestamp}:${verified.signature}`;
      if (endpoint.replayedSignatures.has(replayKey)) {
        sendJson(response, 409, errorBody('replay_detected', 'La firma del webhook ya fue aceptada'));
        return;
      }
      let body: unknown;
      try { body = rawBody.trim() ? JSON.parse(rawBody) : {}; } catch {
        sendJson(response, 400, errorBody('invalid_request', 'El cuerpo de la solicitud debe ser JSON válido'));
        return;
      }
      const eventId = body && typeof body === 'object' && !Array.isArray(body) && typeof (body as Record<string, unknown>).id === 'string' ? (body as Record<string, unknown>).id as string : undefined;
      if (!eventId) {
        sendJson(response, 400, errorBody('invalid_request', 'Se requiere el identificador del evento del webhook'));
        return;
      }
      endpoint.replayedSignatures.add(replayKey);
      sendJson(response, 202, { accepted: true, event_id: eventId, attempt: 1 });
      return;
    }

    if (route.route === 'POST /v1/webhook-endpoints/{id}/deliveries') {
      if (!endpoint) {
        sendJson(response, 404, errorBody('not_found', 'Endpoint de webhook no encontrado'));
        return;
      }
      let body: unknown;
      try { body = await readJsonBody(request); } catch {
        sendJson(response, 400, errorBody('invalid_request', 'El cuerpo de la solicitud debe ser JSON válido'));
        return;
      }
      const input = body as Record<string, unknown>;
      const event = input.event as Record<string, unknown> | undefined;
      const eventId = event && typeof event.id === 'string' ? event.id : undefined;
      if (!event || Array.isArray(event) || !eventId || (input.outcome !== 'success' && input.outcome !== 'fail')) {
        sendJson(response, 400, errorBody('invalid_request', 'Se requieren event.id y outcome (success|fail)'));
        return;
      }
      const deliveryKey = `${endpoint.id}:${eventId}`;
      const delivery = webhookDeliveries.get(deliveryKey) ?? { endpointId: endpoint.id, eventId, attempt: 0, event };
      if (delivery.deadLettered) {
        sendJson(response, 202, { status: 'dead_lettered', attempt: delivery.attempt, max_attempts: MAX_WEBHOOK_ATTEMPTS, dlq_size: deadLetterQueue.length });
        return;
      }
      delivery.attempt += 1;
      webhookDeliveries.set(deliveryKey, delivery);
      if (input.outcome === 'success') {
        sendJson(response, 202, { status: 'delivered', attempt: delivery.attempt });
        return;
      }
      if (delivery.attempt < MAX_WEBHOOK_ATTEMPTS) {
        sendJson(response, 202, { status: 'retry_scheduled', attempt: delivery.attempt, max_attempts: MAX_WEBHOOK_ATTEMPTS });
        return;
      }
      if (deadLetterQueue.length >= MAX_DLQ_ENTRIES) {
        sendJson(response, 503, errorBody('dlq_full', 'La cola de mensajes muertos del webhook está llena'));
        return;
      }
      delivery.deadLettered = true;
      deadLetterQueue.push(delivery);
      sendJson(response, 202, { status: 'dead_lettered', attempt: delivery.attempt, max_attempts: MAX_WEBHOOK_ATTEMPTS, dlq_size: deadLetterQueue.length });
      return;
    }

    if (route.route === 'POST /v1/refunds') {
      const key = idempotencyKeyFrom(request);
      if (!key) return sendJson(response, 400, errorBody('invalid_request', 'Se requiere la cabecera Idempotency-Key'));
      let body: unknown;
      try { body = await readJsonBody(request); } catch {
        return sendJson(response, 400, errorBody('invalid_request', 'El cuerpo de la solicitud debe ser JSON válido'));
      }
      const payload = canonicalize(body);
      const idempotencyId = `refund:${key}`;
      const previous = idempotency.get(idempotencyId);
      if (previous) {
        if (previous.payload !== payload) return sendJson(response, 409, errorBody('idempotency_key_conflict', 'Idempotency-Key ya se usó con una carga diferente'));
        return sendJson(response, previous.status, previous.body);
      }
      const input = body as Record<string, unknown>;
      if (!body || typeof body !== 'object' || Array.isArray(body) || typeof input.payment_intent_id !== 'string' || !Number.isSafeInteger(input.amount_minor) || (input.amount_minor as number) <= 0 || typeof input.currency !== 'string') {
        return sendJson(response, 400, errorBody('invalid_request', 'Se requieren payment_intent_id, amount_minor y currency'));
      }
      const intent = intents.get(input.payment_intent_id);
      if (!intent) return sendJson(response, 404, errorBody('not_found', 'Intención de pago no encontrada'));
      if (intent.status !== 'succeeded') return sendJson(response, 409, errorBody('invalid_state', 'Solo se pueden reembolsar intenciones de pago con estado succeeded'));
      if (input.currency !== intent.currency || input.amount_minor !== intent.amount_minor) {
        return sendJson(response, 422, errorBody('invalid_request', 'Solo se admite un reembolso completo en la moneda de la intención de pago'));
      }
      if (refundByIntent.has(intent.id)) return sendJson(response, 409, errorBody('refund_conflict', 'La intención de pago ya tiene un reembolso'));
      const refund: Refund = {
        id: `re_sandbox_${String(nextRefundId++).padStart(8, '0')}`,
        object: 'refund', payment_intent_id: intent.id, amount_minor: intent.amount_minor,
        currency: intent.currency, status: 'succeeded', ledger_boundary: ledgerBoundary,
      };
      refunds.set(refund.id, refund);
      refundByIntent.set(intent.id, refund);
      const result = { status: 201, body: refund as unknown as Record<string, unknown>, payload };
      idempotency.set(idempotencyId, result);
      sendJson(response, result.status, result.body);
      return;
    }

    if (route.route === 'POST /v1/payment-intents') {
      const key = idempotencyKeyFrom(request);
      if (!key) {
        sendJson(response, 400, errorBody('invalid_request', 'Se requiere la cabecera Idempotency-Key'));
        return;
      }
      let body: unknown;
      try { body = await readJsonBody(request); } catch {
        sendJson(response, 400, errorBody('invalid_request', 'El cuerpo de la solicitud debe ser JSON válido'));
        return;
      }
      const payload = canonicalize(body);
      const idempotencyId = `create:${key}`;
      const previous = idempotency.get(idempotencyId);
      if (previous) {
        if (previous.payload !== payload) sendJson(response, 409, errorBody('idempotency_key_conflict', 'Idempotency-Key ya se usó con una carga diferente'));
        else sendJson(response, previous.status, previous.body);
        return;
      }
      const input = body as Record<string, unknown>;
      if (!body || typeof body !== 'object' || Array.isArray(body) || !Number.isSafeInteger(input.amount_minor) || (input.amount_minor as number) <= 0 || typeof input.currency !== 'string' || !/^[A-Z]{3}$/.test(input.currency) || typeof input.merchant_order_id !== 'string' || input.merchant_order_id.length === 0 || (input.description !== undefined && typeof input.description !== 'string')) {
        sendJson(response, 400, errorBody('invalid_request', 'Se requieren amount_minor, currency y merchant_order_id'));
        return;
      }
      const intent: PaymentIntent = {
        id: `pi_sandbox_${String(nextIntentId++).padStart(8, '0')}`,
        object: 'payment_intent',
        amount_minor: input.amount_minor as number,
        currency: input.currency,
        merchant_order_id: input.merchant_order_id,
        ...(input.description === undefined ? {} : { description: input.description as string }),
        status: 'requires_confirmation',
        ledger_boundary: ledgerBoundary,
      };
      intents.set(intent.id, intent);
      const result = { status: 201, body: intent as unknown as Record<string, unknown>, payload };
      idempotency.set(idempotencyId, result);
      sendJson(response, result.status, result.body);
      return;
    }

    if (route.route === 'POST /v1/payment-intents/{id}/confirm') {
      const key = idempotencyKeyFrom(request);
      if (!key) {
        sendJson(response, 400, errorBody('invalid_request', 'Se requiere la cabecera Idempotency-Key'));
        return;
      }
      const intent = intents.get(segments[3] ?? '');
      if (!intent) {
        sendJson(response, 404, errorBody('not_found', 'Intención de pago no encontrada'));
        return;
      }
      let body: unknown;
      try { body = await readJsonBody(request); } catch {
        sendJson(response, 400, errorBody('invalid_request', 'El cuerpo de la solicitud debe ser JSON válido'));
        return;
      }
      const payload = canonicalize(body);
      const idempotencyId = `confirm:${intent.id}:${key}`;
      const previous = idempotency.get(idempotencyId);
      if (previous) {
        if (previous.payload !== payload) sendJson(response, 409, errorBody('idempotency_key_conflict', 'Idempotency-Key ya se usó con una carga diferente'));
        else sendJson(response, previous.status, previous.body);
        return;
      }
      if (intent.status !== 'requires_confirmation') {
        sendJson(response, 409, errorBody('invalid_state', 'La intención de pago no se puede confirmar desde su estado actual'));
        return;
      }
      intent.status = 'succeeded';
      const result = { status: 200, body: intent as unknown as Record<string, unknown>, payload };
      idempotency.set(idempotencyId, result);
      sendJson(response, result.status, result.body);
      return;
    }

    sendJson(response, 501, {
      error: 'not_implemented',
      message: `${route.route} está reservada para una tarea posterior de la Fase 5`,
      route: route.route,
      ledger_boundary: ledgerBoundary,
    });
  });
}

if (process.argv[1] && new URL(import.meta.url).pathname === process.argv[1]) {
  const port = Number.parseInt(process.env.PORT ?? '8090', 10);
  createServer().listen(port, '0.0.0.0', () => {
    console.log(`cholopay-gateway escuchando en :${port}`);
  });
}

import { createHmac, timingSafeEqual } from 'node:crypto';
import { createServer as createHttpServer, type IncomingMessage, type Server, type ServerResponse } from 'node:http';

export type Scope = 'payment:read' | 'payment:write' | 'webhook:write';
type PaymentIntentStatus = 'requires_confirmation' | 'succeeded';
type BillPayment = {
  id: string;
  object: 'bill_payment';
  participant_id: 'BANCO_COSTA';
  invoice_id: string;
  amount: string;
  currency: 'USD';
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

export type GatewayOptions = { billerBaseUrl?: string; billerEventSecret?: string };

export function createServer(options: GatewayOptions = {}): Server {
  const intents = new Map<string, PaymentIntent>();
  const billPayments = new Map<string, BillPayment>();
  const billPaymentRequests = new Map<string, { payload: string; payment: BillPayment; request: Record<string, unknown> }>();
  const idempotency = new Map<string, IdempotentResponse>();
  let nextIntentId = 1;
  let nextBillPaymentId = 1;
  let nextWebhookEndpointId = 1;
  const billerBaseUrl = options.billerBaseUrl ?? process.env.BILLER_URL ?? 'http://utilities:8080';
  const billerEventSecret = options.billerEventSecret ?? process.env.INTERNAL_EVENT_SECRET ?? 'local-test-only';
  const webhookEndpoints = new Map<string, WebhookEndpoint>();
  const webhookDeliveries = new Map<string, WebhookDelivery>();
  const deadLetterQueue: WebhookDelivery[] = [];
  const ledgerBoundary = 'This gateway never writes balances or any ledger.';

  return createHttpServer(async (request, response) => {
    const method = request.method ?? 'GET';
    const pathname = new URL(request.url ?? '/', 'http://gateway.local').pathname;

    if (method === 'GET' && pathname === '/health/live') {
      sendJson(response, 200, { status: 'ok', service: 'cholopay-gateway' });
      return;
    }

    const route = placeholderRoutes.find((candidate) => candidate.method === method && candidate.pattern.test(pathname));
    if (!route) {
      sendJson(response, 404, errorBody('not_found', 'Route not found'));
      return;
    }

    if (route.authRequired !== false) {
      const authorization = authorize(request, route.scope);
      if (authorization === 'missing') {
        sendJson(response, 401, errorBody('unauthorized', 'A valid sandbox API key is required'));
        return;
      }
      if (authorization === 'forbidden') {
        sendJson(response, 403, errorBody('forbidden', `API key lacks scope ${route.scope}`));
        return;
      }
    }

    const segments = pathname.split('/');
    const endpointId = segments[3] ?? '';
    const endpoint = webhookEndpoints.get(endpointId);
    if (route.route === 'GET /v1/bill-payments/{id}') {
      const payment = billPayments.get(segments[3] ?? '');
      if (!payment) return sendJson(response, 404, errorBody('not_found', 'Bill payment not found'));
      sendJson(response, 200, payment);
      return;
    }

    if (route.route === 'POST /v1/bill-payments' || route.route === 'POST /v1/bill-payments/{id}/apply') {
      const isApply = route.route.endsWith('/apply');
      const key = idempotencyKeyFrom(request);
      if (!key) return sendJson(response, 400, errorBody('invalid_request', 'Idempotency-Key header is required'));
      let body: unknown;
      try { body = await readJsonBody(request); } catch { return sendJson(response, 400, errorBody('invalid_request', 'Request body must be valid JSON')); }
      let payment: BillPayment;
      let input: Record<string, unknown>;
      if (isApply) {
        const existing = billPayments.get(segments[3] ?? '');
        const saved = existing && [...billPaymentRequests.values()].find((entry) => entry.payment.id === existing.id);
        if (!existing || !saved) return sendJson(response, 404, errorBody('not_found', 'Bill payment not found'));
        payment = existing;
        input = saved.request;
      } else {
        if (!body || typeof body !== 'object' || Array.isArray(body)) return sendJson(response, 400, errorBody('invalid_request', 'Request body must be an object'));
        input = body as Record<string, unknown>;
        if (input.participant_id !== 'BANCO_COSTA' || input.invoice_id !== 'invoice-elec-2025-001' || input.service !== 'electricity' || typeof input.amount !== 'string' || input.currency !== 'USD') {
          return sendJson(response, 422, errorBody('invalid_request', 'Only Banco Costa electricity invoice payments are supported'));
        }
        const payload = canonicalize(input);
        const previous = billPaymentRequests.get(key);
        if (previous) {
          if (previous.payload !== payload) return sendJson(response, 409, errorBody('idempotency_key_conflict', 'Idempotency-Key was already used with a different payload'));
          return sendJson(response, previous.payment.status === 'APPLIED' ? 201 : 202, previous.payment);
        }
        payment = { id: `bp_sandbox_${String(nextBillPaymentId++).padStart(8, '0')}`, object: 'bill_payment', participant_id: 'BANCO_COSTA', invoice_id: input.invoice_id as string, amount: input.amount as string, currency: 'USD', status: 'PENDING_BILLER', ledger_boundary: ledgerBoundary };
        billPayments.set(payment.id, payment);
        billPaymentRequests.set(key, { payload, payment, request: input });
      }
      const utilityKey = `bill-payment:${key}`;
      try {
        const intentResponse = await fetch(`${billerBaseUrl}/api/v1/payment-intents`, { method: 'POST', headers: { 'content-type': 'application/json', 'idempotency-key': utilityKey }, body: JSON.stringify({ invoice_id: payment.invoice_id, amount: payment.amount, currency: payment.currency }), signal: AbortSignal.timeout(1000) });
        if (!intentResponse.ok) { payment.status = 'PENDING_BILLER'; return sendJson(response, 202, payment); }
        const intent = await intentResponse.json() as { id?: string };
        if (!intent.id) { payment.status = 'PENDING_APPLICATION'; return sendJson(response, 202, payment); }
        const event = { event_type: 'payment.settled', invoice_id: payment.invoice_id, payment_intent_id: intent.id };
        const timestamp = String(Math.floor(Date.now() / 1000));
        const canonical = canonicalize(event);
        const signature = createHmac('sha256', billerEventSecret).update(`${timestamp}.${canonical}`).digest('hex');
        const eventResponse = await fetch(`${billerBaseUrl}/api/v1/internal/payment-events`, { method: 'POST', headers: { 'content-type': 'application/json', 'x-cholopay-event-id': payment.id, 'x-cholopay-timestamp': timestamp, 'x-cholopay-signature': signature }, body: JSON.stringify(event), signal: AbortSignal.timeout(1000) });
        payment.payment_intent_id = intent.id;
        payment.status = eventResponse.ok ? 'APPLIED' : 'PENDING_APPLICATION';
      } catch { payment.status = 'PENDING_BILLER'; }
      sendJson(response, payment.status === 'APPLIED' ? 201 : 202, payment);
      return;
    }

    if (route.route === 'GET /v1/payment-intents/{id}') {
      const intent = intents.get(segments[3] ?? '');
      if (!intent) {
        sendJson(response, 404, errorBody('not_found', 'Payment intent not found'));
        return;
      }
      sendJson(response, 200, intent);
      return;
    }

    if (route.route === 'POST /v1/webhook-endpoints') {
      let body: unknown;
      try { body = await readJsonBody(request); } catch {
        sendJson(response, 400, errorBody('invalid_request', 'Request body must be valid JSON'));
        return;
      }
      const input = body as Record<string, unknown>;
      if (!body || typeof body !== 'object' || Array.isArray(body) || typeof input.url !== 'string' || !/^https?:\/\/[^\s]+$/.test(input.url)) {
        sendJson(response, 400, errorBody('invalid_request', 'url must be a valid synthetic HTTP(S) endpoint'));
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
        sendJson(response, 404, errorBody('not_found', 'Webhook endpoint not found'));
        return;
      }
      const rawBody = await readRawBody(request);
      const verified = verifyWebhookSignature(webhookSignatureFrom(request), endpoint.secret, rawBody);
      if (!verified) {
        sendJson(response, 401, errorBody('invalid_signature', 'Webhook signature is invalid or expired'));
        return;
      }
      const replayKey = `${verified.timestamp}:${verified.signature}`;
      if (endpoint.replayedSignatures.has(replayKey)) {
        sendJson(response, 409, errorBody('replay_detected', 'Webhook signature was already accepted'));
        return;
      }
      let body: unknown;
      try { body = rawBody.trim() ? JSON.parse(rawBody) : {}; } catch {
        sendJson(response, 400, errorBody('invalid_request', 'Request body must be valid JSON'));
        return;
      }
      const eventId = body && typeof body === 'object' && !Array.isArray(body) && typeof (body as Record<string, unknown>).id === 'string' ? (body as Record<string, unknown>).id as string : undefined;
      if (!eventId) {
        sendJson(response, 400, errorBody('invalid_request', 'Webhook event id is required'));
        return;
      }
      endpoint.replayedSignatures.add(replayKey);
      sendJson(response, 202, { accepted: true, event_id: eventId, attempt: 1 });
      return;
    }

    if (route.route === 'POST /v1/webhook-endpoints/{id}/deliveries') {
      if (!endpoint) {
        sendJson(response, 404, errorBody('not_found', 'Webhook endpoint not found'));
        return;
      }
      let body: unknown;
      try { body = await readJsonBody(request); } catch {
        sendJson(response, 400, errorBody('invalid_request', 'Request body must be valid JSON'));
        return;
      }
      const input = body as Record<string, unknown>;
      const event = input.event as Record<string, unknown> | undefined;
      const eventId = event && typeof event.id === 'string' ? event.id : undefined;
      if (!event || Array.isArray(event) || !eventId || (input.outcome !== 'success' && input.outcome !== 'fail')) {
        sendJson(response, 400, errorBody('invalid_request', 'event.id and outcome (success|fail) are required'));
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
        sendJson(response, 503, errorBody('dlq_full', 'Webhook dead-letter queue is full'));
        return;
      }
      delivery.deadLettered = true;
      deadLetterQueue.push(delivery);
      sendJson(response, 202, { status: 'dead_lettered', attempt: delivery.attempt, max_attempts: MAX_WEBHOOK_ATTEMPTS, dlq_size: deadLetterQueue.length });
      return;
    }

    if (route.route === 'POST /v1/payment-intents') {
      const key = idempotencyKeyFrom(request);
      if (!key) {
        sendJson(response, 400, errorBody('invalid_request', 'Idempotency-Key header is required'));
        return;
      }
      let body: unknown;
      try { body = await readJsonBody(request); } catch {
        sendJson(response, 400, errorBody('invalid_request', 'Request body must be valid JSON'));
        return;
      }
      const payload = canonicalize(body);
      const idempotencyId = `create:${key}`;
      const previous = idempotency.get(idempotencyId);
      if (previous) {
        if (previous.payload !== payload) sendJson(response, 409, errorBody('idempotency_key_conflict', 'Idempotency-Key was already used with a different payload'));
        else sendJson(response, previous.status, previous.body);
        return;
      }
      const input = body as Record<string, unknown>;
      if (!body || typeof body !== 'object' || Array.isArray(body) || !Number.isSafeInteger(input.amount_minor) || (input.amount_minor as number) <= 0 || typeof input.currency !== 'string' || !/^[A-Z]{3}$/.test(input.currency) || typeof input.merchant_order_id !== 'string' || input.merchant_order_id.length === 0 || (input.description !== undefined && typeof input.description !== 'string')) {
        sendJson(response, 400, errorBody('invalid_request', 'amount_minor, currency, and merchant_order_id are required'));
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
        sendJson(response, 400, errorBody('invalid_request', 'Idempotency-Key header is required'));
        return;
      }
      const intent = intents.get(segments[3] ?? '');
      if (!intent) {
        sendJson(response, 404, errorBody('not_found', 'Payment intent not found'));
        return;
      }
      let body: unknown;
      try { body = await readJsonBody(request); } catch {
        sendJson(response, 400, errorBody('invalid_request', 'Request body must be valid JSON'));
        return;
      }
      const payload = canonicalize(body);
      const idempotencyId = `confirm:${intent.id}:${key}`;
      const previous = idempotency.get(idempotencyId);
      if (previous) {
        if (previous.payload !== payload) sendJson(response, 409, errorBody('idempotency_key_conflict', 'Idempotency-Key was already used with a different payload'));
        else sendJson(response, previous.status, previous.body);
        return;
      }
      if (intent.status !== 'requires_confirmation') {
        sendJson(response, 409, errorBody('invalid_state', 'Payment intent cannot be confirmed from its current state'));
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
      message: `${route.route} is reserved for a later Phase 5 task`,
      route: route.route,
      ledger_boundary: ledgerBoundary,
    });
  });
}

if (process.argv[1] && new URL(import.meta.url).pathname === process.argv[1]) {
  const port = Number.parseInt(process.env.PORT ?? '8090', 10);
  createServer().listen(port, '0.0.0.0', () => {
    console.log(`cholopay-gateway listening on :${port}`);
  });
}

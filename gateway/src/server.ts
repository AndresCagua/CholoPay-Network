import { createServer as createHttpServer, type IncomingMessage, type Server, type ServerResponse } from 'node:http';

export type Scope = 'payment:read' | 'payment:write' | 'webhook:write';
type PaymentIntentStatus = 'requires_confirmation' | 'succeeded';
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

const syntheticApiKeys = new Map<string, ReadonlySet<Scope>>([
  ['sk_sandbox_merchant_rw', new Set<Scope>(['payment:read', 'payment:write', 'webhook:write'])],
  ['sk_sandbox_merchant_ro', new Set<Scope>(['payment:read'])],
]);

const placeholderRoutes: Array<{ method: string; pattern: RegExp; scope: Scope; route: string }> = [
  { method: 'POST', pattern: /^\/v1\/payment-intents$/, scope: 'payment:write', route: 'POST /v1/payment-intents' },
  { method: 'GET', pattern: /^\/v1\/payment-intents\/[^/]+$/, scope: 'payment:read', route: 'GET /v1/payment-intents/{id}' },
  { method: 'POST', pattern: /^\/v1\/payment-intents\/[^/]+\/confirm$/, scope: 'payment:write', route: 'POST /v1/payment-intents/{id}/confirm' },
  { method: 'POST', pattern: /^\/v1\/refunds$/, scope: 'payment:write', route: 'POST /v1/refunds' },
  { method: 'POST', pattern: /^\/v1\/webhook-endpoints$/, scope: 'webhook:write', route: 'POST /v1/webhook-endpoints' },
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

function readJsonBody(request: IncomingMessage): Promise<unknown> {
  return new Promise((resolve, reject) => {
    let content = '';
    request.setEncoding('utf8');
    request.on('data', (chunk: string) => { content += chunk; });
    request.on('end', () => {
      if (!content.trim()) return resolve({});
      try { resolve(JSON.parse(content)); } catch { reject(new Error('invalid_json')); }
    });
    request.on('error', reject);
  });
}

export function createServer(): Server {
  const intents = new Map<string, PaymentIntent>();
  const idempotency = new Map<string, IdempotentResponse>();
  let nextIntentId = 1;
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

    const authorization = authorize(request, route.scope);
    if (authorization === 'missing') {
      sendJson(response, 401, errorBody('unauthorized', 'A valid sandbox API key is required'));
      return;
    }
    if (authorization === 'forbidden') {
      sendJson(response, 403, errorBody('forbidden', `API key lacks scope ${route.scope}`));
      return;
    }

    const segments = pathname.split('/');
    if (route.route === 'GET /v1/payment-intents/{id}') {
      const intent = intents.get(segments[3] ?? '');
      if (!intent) {
        sendJson(response, 404, errorBody('not_found', 'Payment intent not found'));
        return;
      }
      sendJson(response, 200, intent);
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

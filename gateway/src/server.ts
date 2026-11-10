import { createServer as createHttpServer, type IncomingMessage, type Server, type ServerResponse } from 'node:http';

export type Scope = 'payment:read' | 'payment:write' | 'webhook:write';

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

function apiKeyFrom(request: IncomingMessage): string | undefined {
  const value = request.headers['x-api-key'];
  return Array.isArray(value) ? value[0] : value;
}

function authorize(request: IncomingMessage, requiredScope: Scope): 'missing' | 'forbidden' | 'ok' {
  const key = apiKeyFrom(request);
  if (!key || !syntheticApiKeys.has(key)) return 'missing';
  return syntheticApiKeys.get(key)?.has(requiredScope) ? 'ok' : 'forbidden';
}

export function createServer(): Server {
  return createHttpServer((request, response) => {
    const method = request.method ?? 'GET';
    const pathname = new URL(request.url ?? '/', 'http://gateway.local').pathname;

    if (method === 'GET' && pathname === '/health/live') {
      sendJson(response, 200, { status: 'ok', service: 'cholopay-gateway' });
      return;
    }

    const route = placeholderRoutes.find((candidate) => candidate.method === method && candidate.pattern.test(pathname));
    if (!route) {
      sendJson(response, 404, { error: 'not_found', message: 'Route not found' });
      return;
    }

    const authorization = authorize(request, route.scope);
    if (authorization === 'missing') {
      sendJson(response, 401, { error: 'unauthorized', message: 'A valid sandbox API key is required' });
      return;
    }
    if (authorization === 'forbidden') {
      sendJson(response, 403, { error: 'forbidden', message: `API key lacks scope ${route.scope}` });
      return;
    }

    sendJson(response, 501, {
      error: 'not_implemented',
      message: `${route.route} is reserved for a later Phase 5 task`,
      route: route.route,
      ledger_boundary: 'This gateway scaffold never writes balances or any ledger.',
    });
  });
}

if (process.argv[1] && new URL(import.meta.url).pathname === process.argv[1]) {
  const port = Number.parseInt(process.env.PORT ?? '8090', 10);
  createServer().listen(port, '0.0.0.0', () => {
    console.log(`cholopay-gateway listening on :${port}`);
  });
}

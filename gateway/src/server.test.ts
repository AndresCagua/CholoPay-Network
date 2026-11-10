import test from 'node:test';
import assert from 'node:assert/strict';
import type { AddressInfo } from 'node:net';
import { createServer as createMockServer } from 'node:http';
import { createServer } from './server.js';

async function withServer(run: (baseUrl: string) => Promise<void>): Promise<void> {
  const server = createServer();
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
  const address = server.address() as AddressInfo;
  try {
    await run(`http://127.0.0.1:${address.port}`);
  } finally {
    await new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
  }
}

async function withBiller(run: (baseUrl: string, setAvailable: () => void, setUnavailable: () => void) => Promise<void>): Promise<void> {
  let available = true;
  const intents = new Map<string, string>();
  const biller = createMockServer(async (request, response) => {
    const chunks: Buffer[] = [];
    for await (const chunk of request) chunks.push(Buffer.from(chunk));
    const body = chunks.length ? JSON.parse(Buffer.concat(chunks).toString()) : {};
    response.setHeader('content-type', 'application/json');
    if (!available) { response.writeHead(503); response.end(JSON.stringify({ error: 'unavailable' })); return; }
    if (request.url === '/api/v1/payment-intents') {
      const key = String(request.headers['idempotency-key']);
      const id = intents.get(key) ?? `pi_${intents.size + 1}`;
      intents.set(key, id);
      response.writeHead(201); response.end(JSON.stringify({ id })); return;
    }
    if (request.url === '/api/v1/internal/payment-events') { response.writeHead(200); response.end(JSON.stringify({ status: 'PAID', synthetic: true })); return; }
    response.writeHead(404); response.end('{}');
  });
  await new Promise<void>((resolve) => biller.listen(0, '127.0.0.1', resolve));
  const address = biller.address() as AddressInfo;
  try { await run(`http://127.0.0.1:${address.port}`, () => { available = true; }, () => { available = false; }); }
  finally { await new Promise<void>((resolve, reject) => biller.close((error) => error ? reject(error) : resolve())); }
}

test('health is public and reports the gateway scaffold', async () => {
  await withServer(async (baseUrl) => {
    const response = await fetch(`${baseUrl}/health/live`);
    assert.equal(response.status, 200);
    assert.deepEqual(await response.json(), { status: 'ok', service: 'cholopay-gateway' });
  });
});

test('protected contract routes reject missing and insufficient credentials', async () => {
  await withServer(async (baseUrl) => {
    const missing = await fetch(`${baseUrl}/v1/payment-intents`, { method: 'POST' });
    assert.equal(missing.status, 401);

    const readOnly = await fetch(`${baseUrl}/v1/payment-intents`, {
      method: 'POST',
      headers: { 'X-Api-Key': 'sk_sandbox_merchant_ro' },
    });
    assert.equal(readOnly.status, 403);
  });
});

test('creates and queries a synthetic payment intent without ledger writes', async () => {
  await withServer(async (baseUrl) => {
    const headers = {
      'X-Api-Key': 'sk_sandbox_merchant_rw',
      'Idempotency-Key': 'order-1001-create',
      'Content-Type': 'application/json',
    };
    const response = await fetch(`${baseUrl}/v1/payment-intents`, {
      method: 'POST',
      headers,
      body: JSON.stringify({
        amount_minor: 2500,
        currency: 'USD',
        merchant_order_id: 'ORDER-1001',
        description: 'Compra simulada',
      }),
    });
    assert.equal(response.status, 201);
    const created = await response.json() as Record<string, unknown>;
    assert.equal(created.object, 'payment_intent');
    assert.equal(created.status, 'requires_confirmation');
    assert.equal(created.amount_minor, 2500);
    assert.equal(created.currency, 'USD');
    assert.match(String(created.id), /^pi_sandbox_/);
    assert.match(String(created.ledger_boundary), /never writes balances or any ledger/);

    const queried = await fetch(`${baseUrl}/v1/payment-intents/${created.id}`, {
      headers: { 'X-Api-Key': 'sk_sandbox_merchant_ro' },
    });
    assert.equal(queried.status, 200);
    assert.deepEqual(await queried.json(), created);
  });
});

test('replays an idempotent create and rejects a conflicting payload', async () => {
  await withServer(async (baseUrl) => {
    const headers = {
      'X-Api-Key': 'sk_sandbox_merchant_rw',
      'Idempotency-Key': 'stable-create',
      'Content-Type': 'application/json',
    };
    const payload = { amount_minor: 100, currency: 'USD', merchant_order_id: 'ORDER-1' };
    const first = await fetch(`${baseUrl}/v1/payment-intents`, {
      method: 'POST', headers, body: JSON.stringify(payload),
    });
    const firstBody = await first.json();
    const replay = await fetch(`${baseUrl}/v1/payment-intents`, {
      method: 'POST', headers, body: JSON.stringify(payload),
    });
    assert.equal(replay.status, 201);
    assert.deepEqual(await replay.json(), firstBody);

    const conflict = await fetch(`${baseUrl}/v1/payment-intents`, {
      method: 'POST', headers,
      body: JSON.stringify({ ...payload, amount_minor: 101 }),
    });
    assert.equal(conflict.status, 409);
    assert.deepEqual(await conflict.json(), {
      error: 'idempotency_key_conflict',
      message: 'Idempotency-Key was already used with a different payload',
    });
  });
});

test('confirms an intent through explicit synthetic state transitions', async () => {
  await withServer(async (baseUrl) => {
    const create = await fetch(`${baseUrl}/v1/payment-intents`, {
      method: 'POST',
      headers: {
        'X-Api-Key': 'sk_sandbox_merchant_rw',
        'Idempotency-Key': 'transition-create',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ amount_minor: 500, currency: 'USD', merchant_order_id: 'ORDER-2' }),
    });
    const intent = await create.json() as { id: string };
    const confirm = await fetch(`${baseUrl}/v1/payment-intents/${intent.id}/confirm`, {
      method: 'POST',
      headers: { 'X-Api-Key': 'sk_sandbox_merchant_rw', 'Idempotency-Key': 'transition-confirm' },
    });
    assert.equal(confirm.status, 200);
    const confirmed = await confirm.json() as Record<string, unknown>;
    assert.equal(confirmed.status, 'succeeded');

    const replay = await fetch(`${baseUrl}/v1/payment-intents/${intent.id}/confirm`, {
      method: 'POST',
      headers: { 'X-Api-Key': 'sk_sandbox_merchant_rw', 'Idempotency-Key': 'transition-confirm' },
    });
    assert.deepEqual(await replay.json(), confirmed);

    const queried = await fetch(`${baseUrl}/v1/payment-intents/${intent.id}`, {
      headers: { 'X-Api-Key': 'sk_sandbox_merchant_ro' },
    });
    assert.equal((await queried.json() as { status: string }).status, 'succeeded');
  });
});

test('completes Banco Costa electricity payment through the biller and preserves replay/conflict semantics', async () => {
  await withBiller(async (billerUrl) => {
    await withServer(async () => {
      const headers = { 'X-Api-Key': 'sk_sandbox_merchant_rw', 'Idempotency-Key': 'costa-elec-001', 'Content-Type': 'application/json' };
      const body = { participant_id: 'BANCO_COSTA', invoice_id: 'invoice-elec-2025-001', service: 'electricity', amount: '42.50', currency: 'USD' };
      const server = createServer({ billerBaseUrl: billerUrl });
      await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
      const address = server.address() as AddressInfo;
      try {
        const url = `http://127.0.0.1:${address.port}/v1/bill-payments`;
        const first = await fetch(url, { method: 'POST', headers, body: JSON.stringify(body) });
        assert.equal(first.status, 201);
        const payment = await first.json() as Record<string, unknown>;
        assert.equal(payment.status, 'APPLIED');
        assert.equal(payment.participant_id, 'BANCO_COSTA');
        const replay = await fetch(url, { method: 'POST', headers, body: JSON.stringify(body) });
        assert.equal(replay.status, 201);
        assert.deepEqual(await replay.json(), payment);
        const conflict = await fetch(url, { method: 'POST', headers, body: JSON.stringify({ ...body, amount: '1.00' }) });
        assert.equal(conflict.status, 409);
      } finally { await new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve())); }
    });
  });
});

test('keeps bill payment pending while biller is unavailable and applies it on retry', async () => {
  await withBiller(async (billerUrl, setAvailable, setUnavailable) => {
    await withServer(async () => {
      setUnavailable();
      const server = createServer({ billerBaseUrl: billerUrl });
      await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
      const address = server.address() as AddressInfo;
      try {
        const headers = { 'X-Api-Key': 'sk_sandbox_merchant_rw', 'Idempotency-Key': 'costa-elec-eventual', 'Content-Type': 'application/json' };
        const body = { participant_id: 'BANCO_COSTA', invoice_id: 'invoice-elec-2025-001', service: 'electricity', amount: '42.50', currency: 'USD' };
        const first = await fetch(`http://127.0.0.1:${address.port}/v1/bill-payments`, { method: 'POST', headers, body: JSON.stringify(body) });
        assert.equal(first.status, 202);
        const pending = await first.json() as { id: string; status: string };
        assert.equal(pending.status, 'PENDING_BILLER');
        setAvailable();
        const apply = await fetch(`http://127.0.0.1:${address.port}/v1/bill-payments/${pending.id}/apply`, { method: 'POST', headers, body: '{}' });
        assert.equal(apply.status, 201);
        assert.equal((await apply.json() as { status: string }).status, 'APPLIED');
      } finally { await new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve())); }
    });
  });
});

test('validates payment intent input and keeps read scope unchanged', async () => {
  await withServer(async (baseUrl) => {
    const invalid = await fetch(`${baseUrl}/v1/payment-intents`, {
      method: 'POST',
      headers: { 'X-Api-Key': 'sk_sandbox_merchant_rw', 'Idempotency-Key': 'invalid' },
      body: JSON.stringify({ amount_minor: 0, currency: 'usd' }),
    });
    assert.equal(invalid.status, 400);
    assert.equal((await invalid.json() as { error: string }).error, 'invalid_request');

    const forbidden = await fetch(`${baseUrl}/v1/payment-intents/nope/confirm`, {
      method: 'POST',
      headers: { 'X-Api-Key': 'sk_sandbox_merchant_ro', 'Idempotency-Key': 'confirm' },
    });
    assert.equal(forbidden.status, 403);
  });
});

test('registers a synthetic webhook and accepts a valid timestamped HMAC signature once', async () => {
  await withServer(async (baseUrl) => {
    const registration = await fetch(`${baseUrl}/v1/webhook-endpoints`, {
      method: 'POST',
      headers: { 'X-Api-Key': 'sk_sandbox_merchant_rw', 'Content-Type': 'application/json' },
      body: JSON.stringify({ url: 'http://sandbox.local/webhooks/orders' }),
    });
    assert.equal(registration.status, 201);
    const endpoint = await registration.json() as { id: string; secret: string };
    const payload = JSON.stringify({ id: 'evt_001', type: 'payment.succeeded' });
    const timestamp = Math.floor(Date.now() / 1000);
    const signature = await import('node:crypto').then(({ createHmac }) => createHmac('sha256', endpoint.secret).update(`${timestamp}.${payload}`).digest('hex'));
    const delivered = await fetch(`${baseUrl}/v1/webhook-endpoints/${endpoint.id}/events`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Webhook-Signature': `t=${timestamp},v1=${signature}` },
      body: payload,
    });
    assert.equal(delivered.status, 202);
    assert.deepEqual(await delivered.json(), { accepted: true, event_id: 'evt_001', attempt: 1 });
  });
});

test('rejects a replayed webhook signature', async () => {
  await withServer(async (baseUrl) => {
    const registration = await fetch(`${baseUrl}/v1/webhook-endpoints`, {
      method: 'POST', headers: { 'X-Api-Key': 'sk_sandbox_merchant_rw', 'Content-Type': 'application/json' },
      body: JSON.stringify({ url: 'http://sandbox.local/webhooks/replay' }),
    });
    const endpoint = await registration.json() as { id: string; secret: string };
    const payload = JSON.stringify({ id: 'evt_replay', type: 'test' });
    const timestamp = Math.floor(Date.now() / 1000);
    const { createHmac } = await import('node:crypto');
    const signature = createHmac('sha256', endpoint.secret).update(`${timestamp}.${payload}`).digest('hex');
    const headers = { 'Content-Type': 'application/json', 'Webhook-Signature': `t=${timestamp},v1=${signature}` };
    const first = await fetch(`${baseUrl}/v1/webhook-endpoints/${endpoint.id}/events`, { method: 'POST', headers, body: payload });
    assert.equal(first.status, 202);
    const replay = await fetch(`${baseUrl}/v1/webhook-endpoints/${endpoint.id}/events`, { method: 'POST', headers, body: payload });
    assert.equal(replay.status, 409);
    assert.deepEqual(await replay.json(), { error: 'replay_detected', message: 'Webhook signature was already accepted' });
  });
});

test('progresses bounded synthetic retries and admits a final failure to the in-memory DLQ', async () => {
  await withServer(async (baseUrl) => {
    const registration = await fetch(`${baseUrl}/v1/webhook-endpoints`, {
      method: 'POST', headers: { 'X-Api-Key': 'sk_sandbox_merchant_rw', 'Content-Type': 'application/json' },
      body: JSON.stringify({ url: 'http://sandbox.local/webhooks/retries' }),
    });
    const endpoint = await registration.json() as { id: string };
    const headers = { 'X-Api-Key': 'sk_sandbox_merchant_rw', 'Content-Type': 'application/json' };
    const request = () => fetch(`${baseUrl}/v1/webhook-endpoints/${endpoint.id}/deliveries`, {
      method: 'POST', headers, body: JSON.stringify({ event: { id: 'evt_retry', type: 'test' }, outcome: 'fail' }),
    });
    assert.deepEqual(await (await request()).json(), { status: 'retry_scheduled', attempt: 1, max_attempts: 3 });
    assert.deepEqual(await (await request()).json(), { status: 'retry_scheduled', attempt: 2, max_attempts: 3 });
    const dead = await request();
    assert.equal(dead.status, 202);
    assert.deepEqual(await dead.json(), { status: 'dead_lettered', attempt: 3, max_attempts: 3, dlq_size: 1 });
    const replayedDeadLetter = await request();
    assert.equal(replayedDeadLetter.status, 202);
    assert.deepEqual(await replayedDeadLetter.json(), { status: 'dead_lettered', attempt: 3, max_attempts: 3, dlq_size: 1 });
  });
});

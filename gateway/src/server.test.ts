import test from 'node:test';
import assert from 'node:assert/strict';
import type { AddressInfo } from 'node:net';
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

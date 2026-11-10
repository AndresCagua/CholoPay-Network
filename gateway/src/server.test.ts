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

test('authorized placeholders are explicit and do not claim ledger behavior', async () => {
  await withServer(async (baseUrl) => {
    const response = await fetch(`${baseUrl}/v1/payment-intents`, {
      method: 'POST',
      headers: { 'X-Api-Key': 'sk_sandbox_merchant_rw' },
    });
    assert.equal(response.status, 501);
    const body = await response.json() as { error: string; ledger_boundary: string };
    assert.equal(body.error, 'not_implemented');
    assert.match(body.ledger_boundary, /never writes balances or any ledger/);
  });
});

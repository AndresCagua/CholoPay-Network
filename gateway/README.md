# CholoPay gateway sandbox

Docker-only Node.js/TypeScript public gateway (`cholopay-gateway`). It provides synthetic scoped API keys and an in-memory payment-intent flow for sandbox behavior only.

## Run

Build and run the image from the repository root:

```sh
docker build -f gateway/Dockerfile -t cholopay-gateway ./gateway
docker run --rm -p 8090:8090 cholopay-gateway
```

The Docker build runs the focused test suite before producing the runtime stage. No host Node.js installation is required.

## Synthetic authentication

Send `X-Api-Key` on every protected route. These development-only keys are intentionally synthetic and are not credentials:

| Key | Scopes |
| --- | --- |
| `sk_sandbox_merchant_rw` | `payment:read`, `payment:write`, `webhook:write` |
| `sk_sandbox_merchant_ro` | `payment:read` |

Missing or unknown keys return `401`; a known key without the route scope returns `403`. Keys are kept in memory and must not be used outside the sandbox.

## Synthetic payment intents

`POST /v1/payment-intents` (`payment:write`) creates an in-memory intent from `amount_minor`, three-letter uppercase `currency`, `merchant_order_id`, and optional `description`. New intents start in `requires_confirmation` and receive a synthetic `pi_sandbox_...` identifier.

Every create and confirm request requires an `Idempotency-Key`. Repeating the same key with the same JSON payload replays the original response; reusing it with a different payload returns `409 idempotency_key_conflict`.

- `GET /v1/payment-intents/{id}` (`payment:read`) queries the current synthetic state.
- `POST /v1/payment-intents/{id}/confirm` (`payment:write`) transitions `requires_confirmation` to `succeeded`.
- `POST /v1/refunds` (`payment:write`) accepts `{ "payment_intent_id": "pi_sandbox_...", "amount_minor": 2500, "currency": "USD" }` only after the intent is `succeeded`. This bounded sandbox supports one full refund per intent, returns `201` with `status: succeeded`, and never writes a ledger. Reusing an idempotency key with another payload returns `409`; a second refund for the same intent returns `409 refund_conflict`.

## Banco Costa electricity bill flow

`POST /v1/bill-payments` is the bounded public vertical slice. It requires the merchant write key, an `Idempotency-Key`, and `{ "participant_id": "BANCO_COSTA", "invoice_id": "invoice-elec-2025-001", "service": "electricity", "amount": "42.50", "currency": "USD" }`. The gateway validates the explicit Banco Costa caller boundary, asks CholoServicios for a pending intent, and submits a signed `payment.settled` event for idempotent application. `201 APPLIED` means the synthetic biller application completed; `202 PENDING_BILLER` or `PENDING_APPLICATION` is not settlement proof. `POST /v1/bill-payments/{id}/apply` retries eventual application and `GET /v1/bill-payments/{id}` reads status.

This is Docker-only synthetic traffic: no real money, external infrastructure, gateway ledger writes, or utility ledger writes occur.

## Synthetic webhooks

`POST /v1/webhook-endpoints` (`webhook:write`) registers an in-memory endpoint from a synthetic HTTP(S) URL and returns a sandbox-only `whsec_...` signing secret. No network delivery is performed. The returned secret signs the exact request body with HMAC-SHA256 using the format `${timestamp}.${raw_body}`; send it as `Webhook-Signature: t=<unix_seconds>,v1=<hex_digest>` to `POST /v1/webhook-endpoints/{id}/events`. Timestamps older than five minutes, malformed signatures, and mismatches return `401`; an already accepted timestamp/signature pair returns `409 replay_detected`.

For local delivery simulation, `POST /v1/webhook-endpoints/{id}/deliveries` (`webhook:write`) accepts `{ "event": { "id": "..." }, "outcome": "success" | "fail" }`. Failures are bounded to three attempts: the first two return `retry_scheduled`, and the third is admitted to an in-memory dead-letter queue. The queue is capped at 100 entries and all state is process-local.

`GET /health/live` is unauthenticated and returns `200`.

## Explicit ledger boundary

Payment intents are stored only in the gateway process memory. This service has no ledger client, database, balance store, payment processor, commerce state, or webhook delivery, and never reads or modifies balances. Confirmation is synthetic and does not create ledger entries or move money.

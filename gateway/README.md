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
- `POST /v1/refunds` (`payment:write`) remains reserved and returns `501`.
- `POST /v1/webhook-endpoints` (`webhook:write`) remains reserved and returns `501`.

`GET /health/live` is unauthenticated and returns `200`.

## Explicit ledger boundary

Payment intents are stored only in the gateway process memory. This service has no ledger client, database, balance store, payment processor, commerce state, or webhook delivery, and never reads or modifies balances. Confirmation is synthetic and does not create ledger entries or move money.

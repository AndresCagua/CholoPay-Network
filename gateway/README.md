# CholoPay gateway scaffold

Docker-only Node.js/TypeScript scaffold for the Phase 5 public gateway (`cholopay-gateway`). It provides a minimal HTTP surface, synthetic scoped API keys, and documented route placeholders.

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

## Contract placeholders

The following Phase 5 routes are present. They return `501 Not Implemented` until later tasks add commerce/payment behavior:

- `POST /v1/payment-intents` (`payment:write`)
- `GET /v1/payment-intents/{id}` (`payment:read`)
- `POST /v1/payment-intents/{id}/confirm` (`payment:write`)
- `POST /v1/refunds` (`payment:write`)
- `POST /v1/webhook-endpoints` (`webhook:write`)

`GET /health/live` is unauthenticated and returns `200`.

## Explicit ledger boundary

This scaffold has no ledger client, database, balance store, payment processor, commerce state, webhook delivery, or payment behavior. It stores no request data and never reads or modifies balances. Any future implementation must integrate through an explicit non-ledger contract and must not write a bank or settlement ledger from this service.

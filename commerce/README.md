# Commerce sandbox

A minimal, synthetic commerce surface for the CholoPay sandbox. It is deliberately independent from the gateway: the service uses only process-local memory, seeded catalog data, and synthetic USD-denominated minor units. It does not process money, call external services, or write a ledger.

## Docker-only usage

From this directory:

```sh
docker compose -f compose.yaml up --build commerce
docker compose -f compose.yaml --profile test run --rm commerce-test
docker compose -f compose.yaml down
```

The API is available on `http://localhost:18081` by default. Data disappears when the container stops.

## Observable API

- `GET /health/live`
- `GET /v1/catalog`
- `POST /v1/carts` — creates an empty cart.
- `GET /v1/carts/{cart_id}`
- `POST /v1/carts/{cart_id}/items` — JSON `{ "sku": "demo-coffee", "quantity": 2 }`.
- `POST /v1/orders` — JSON `{ "cart_id": "..." }` and required `Idempotency-Key` header.
- `POST /v1/checkout` — JSON `{ "cart_id": "..." }` and required `Idempotency-Key`; creates an order and a gateway payment intent, returning `payment_status: "pending"` or `"succeeded"`.
- `POST /v1/checkouts/{checkout_id}/confirm` — server-side synthetic confirmation with required `Idempotency-Key`; a browser redirect is never treated as payment proof.
- `GET /v1/orders/{order_id}`
- `GET /v1/inventory`

Creating an order checks and decrements synthetic inventory atomically. Repeating an order with the same idempotency key returns the original order without a second decrement. Checkout uses `GATEWAY_URL` plus `GATEWAY_API_KEY` for the HTTP gateway, or the explicitly named `GATEWAY_ADAPTER=synthetic` adapter (the default when no URL is configured). It exposes only pending/succeeded payment states; it has no redirect-based success path. A process restart resets all state.

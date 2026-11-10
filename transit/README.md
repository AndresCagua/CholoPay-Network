# Synthetic transit authority API

`cholopay-transit` is a Docker-only, process-local API for educational CholoPay flows. It uses synthetic registrations, fines, obligations, statuses, and constancias. **This is not a real authority, does not issue legal certificates, and must not be used with real personal or vehicle data.** It does not connect to banks, accounts, SPI/DICT, or external infrastructure.

## Docker-only usage

```sh
docker compose -f compose.yaml --profile test run --rm transit-test
docker compose -f compose.yaml up --build transit
docker compose -f compose.yaml down
```

No host port is exposed by default. Data is process-local and disappears when the container stops. To connect from another container on the same Compose network, use `http://transit:8082`.

## HTTP surface

- `GET /health/live`
- `POST /v1/registrations` — `{ "plate": "SYN-001", "vehicle_type": "synthetic-car" }`
- `GET /v1/registrations/{id-or-plate}`
- `POST /v1/fines` — creates a synthetic fine and its unpaid obligation.
- `GET /v1/fines/{id}`
- `GET /v1/obligations/{id}` and `GET /v1/obligations?registration_id=...&status=unpaid`
- `GET /v1/status?registration_id=...`
- `POST /v1/constancias` — creates a synthetic status constancia; it is not an official certificate.
- `POST /v1/payment-intents` — requires `Idempotency-Key`; creates a pending intent only.
- `GET /v1/payment-intents/{id}`

Payment intents are an explicit integration boundary. Repeating the same key and payload returns the same intent; a payload mismatch returns `409`. The service never debits an account, changes an obligation to paid, or writes a ledger. All state is in memory.

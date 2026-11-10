# CholoPay synthetic utilities biller

Docker-only synthetic API for the fictional multi-service utility company (electricity, water, internet, and mobile telephony). It uses only Python's standard library and stores its small fixture set in process memory.

## Boundary

This service is a simulator, not a bank or a payment processor. It uses synthetic identifiers and amounts only. It never connects to SPI, DICT, a bank, a wallet, a customer account, or external infrastructure. It never debits an account, settles money, or writes a ledger. `POST /api/v1/payment-intents` creates an idempotent **pending intent** only, for a future integration boundary.

The Docker configuration exposes port 8080 only to the Docker network; it has no host `ports` mapping by default. To call it from a local host, use an explicitly authorized temporary port mapping rather than changing the default compose file.

## Run

```sh
docker compose up --build
```

The API is available to other containers at `http://utilities:8080`.

## Contract

- `GET /health` — synthetic health response.
- `GET /api/v1/contracts` — list synthetic service contracts.
- `GET /api/v1/contracts/{contract_id}` — retrieve a contract.
- `GET /api/v1/invoices?contract_id={contract_id}` — query invoices, optionally by contract.
- `GET /api/v1/invoices/{invoice_id}` — retrieve an invoice.
- `GET /api/v1/invoices/{invoice_id}/collection-status` — query synthetic collection state.
- `GET /api/v1/invoices/{invoice_id}/payment-status` — query application state for a synthetic settlement.
- `POST /api/v1/payment-intents` — create an intent with `Idempotency-Key` and JSON `{ "invoice_id": "...", "amount": "...", "currency": "USD" }`.
- `POST /api/v1/internal/payment-events` — apply a signed synthetic `payment.settled` event with event-id replay protection. This endpoint does not debit accounts or write a ledger.

A repeated idempotency key and identical request returns the original intent. Reusing the key with a different request returns `409`. Every new intent remains `pending` and explicitly reports `debit_performed: false` and `ledger_entry_created: false`.

## HTTP behavior tests

The tests use the public HTTP interface and the same standard-library handler used by the container:

```sh
PYTHONPATH=utilities python -m unittest discover -s utilities/tests -v
```

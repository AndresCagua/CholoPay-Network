# Phase 4 — contrato de wallet, CholoKey y QR

## Boundary

La wallet Flutter es un cliente de demostración. Consulta APIs del banco autenticado y CholoNet mediante contratos HTTP; nunca accede a PostgreSQL ni escribe directamente en ledgers.

## Wallet session

A synthetic Docker-only session contains:

```json
{
  "session_id": "sess-0001",
  "customer_id": "customer-costa-001",
  "participant_id": "COSTA",
  "account_ids": ["account-costa-001"],
  "roles": ["CUSTOMER"],
  "scopes": ["accounts:read", "transfers:create", "transfers:read", "keys:read", "qr:read"],
  "issuer": "cholopay-bank-costa-local",
  "audience": "cholopay-wallet",
  "issued_at": "2026-11-09T00:00:00Z",
  "expires_at": "2026-11-09T01:00:00Z"
}
```

A session is scoped to one participant and cannot authorize Sierra, Oriente, or central operator resources.

## CholoKey resolution

The wallet displays a minimized destination reference returned by the authorized directory API:

```json
{
  "schema_version": "1.0",
  "key": "ck_demo_receiver_001",
  "participant_id": "SIERRA",
  "account_ref": "acct-ref-sierra-001",
  "display_name": "Synthetic Receiver",
  "correlation_id": "00000000-0000-0000-0000-000000000001",
  "trace_id": "trace-wallet-001",
  "resolved_at": "2026-11-09T00:00:00Z"
}
```

The wallet must not expose private bank data beyond the minimized display fields.

## CholoPay QR payload

This is a CholoPay-owned format and is not Pix-compatible:

```json
{
  "format": "cholopay.qr",
  "version": 1,
  "mode": "static",
  "participant_id": "SIERRA",
  "destination_key": "ck_demo_receiver_001",
  "merchant_ref": "merchant-demo-001",
  "amount_minor": null,
  "currency": "USD",
  "expires_at": null,
  "correlation_id": "00000000-0000-0000-0000-000000000002",
  "payload_id": "qr-payload-001",
  "created_at": "2026-11-09T00:00:00Z"
}
```

Dynamic QR uses a positive `amount_minor` and an expiry. The wallet validates format, version, participant, amount, currency, expiry, and destination before showing confirmation.

## Transfer confirmation

Every confirmation creates or reuses one idempotency key. UI retries never generate a new transfer identifier:

```json
{
  "schema_version": "1.0",
  "transfer_id": "00000000-0000-0000-0000-000000000010",
  "idempotency_key": "wallet-confirm-000000000000001",
  "amount_minor": 2500,
  "currency": "USD",
  "source_account_id": "account-costa-001",
  "destination_key": "ck_demo_receiver_001",
  "correlation_id": "00000000-0000-0000-0000-000000000003",
  "causation_id": "00000000-0000-0000-0000-000000000004",
  "created_at": "2026-11-09T00:00:00Z",
  "trace_id": "trace-wallet-confirm-001"
}
```

The UI renders `CREATED`, `VALIDATING`, `FUNDS_RESERVED`, `READY_FOR_SETTLEMENT`, `SETTLED`, `CREDIT_PENDING`, `COMPLETED`, `REJECTED`, and `UNKNOWN_PENDING_RECONCILIATION` distinctly. `SETTLED` is not `COMPLETED`.

## Movement read model

Movements are read-only projections from the authenticated bank API and include `transaction_id`, `account_id`, `amount_minor`, `currency`, `direction`, `state`, `created_at`, `settled_at`, `correlation_id`, and `trace_id`. Reconnection revalidates the server projection before updating the UI.

## Required contract metadata

New requests and events include `schema_version`, `correlation_id`, `idempotency_key` where applicable, UTC timestamps, and traceability. QR payloads are synthetic and versioned; no external network compatibility is claimed.

# Banco Sierra web panel

Reads `GET {TRANSFER_API_BASE_URL}/v1/participants/bank-sierra/transfers` through the participant-scoped client in `src/app/transfer-read-model.ts`. The default synthetic adapter is read-only and enforces the participant in its contract; configure the HTTP base at runtime through the injected `globalThis.__env.TRANSFER_API_BASE_URL`. No browser database, ledger, or filesystem access is used. Loading, error, and empty states are rendered explicitly.

Docker check: `docker build -t cholopay-bank-sierra-web ./web/bank-sierra`.

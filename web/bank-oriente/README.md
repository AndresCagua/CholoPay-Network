# Banco Oriente web panel

Reads `GET {VITE_TRANSFER_API_BASE_URL}/v1/participants/bank-oriente/transfers` through `src/transfer.ts`. Without a base URL the read-only synthetic adapter is used to exercise the same public API boundary; it enforces participant scope and performs no database, ledger, or filesystem access. Loading, error, and empty states are explicit.

Docker check: `docker build -t cholopay-bank-oriente-web ./web/bank-oriente`.

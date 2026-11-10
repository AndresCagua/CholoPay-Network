# Banco Costa web panel

Reads `GET {NEXT_PUBLIC_TRANSFER_API_BASE_URL}/v1/participants/bank-costa/transfers` through `app/transfers.ts`. The synthetic adapter is the default when the base URL is absent; it is read-only and participant-scoped, and exists only to exercise the public API boundary until bank-core exposes this projection. The browser never accesses a database, ledger, or filesystem. Loading, error, and empty responses are explicit.

Docker check: `docker build -t cholopay-bank-costa-web ./web/bank-costa`.

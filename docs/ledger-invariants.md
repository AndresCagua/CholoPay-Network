# Ledger invariants

1. Every journal is balanced: debits plus credits equal zero per currency.
2. Amounts are integer USD cents (`BIGINT`), never floating point.
3. Restricted accounts never have negative available balance.
4. One business instruction cannot create two settlement operations.
5. Interbank settlement positions conserve value unless an explicit, audited funding operation creates balancing entries.
6. Published entries are immutable; correction is compensating movement.
7. Idempotency keys are scoped, bound to a request hash, and reject a different payload with HTTP 409.
8. Ledger state is authoritative in PostgreSQL; caches and event streams can be rebuilt.
9. Every financial operation carries transaction, correlation, causation, and audit identifiers.

Tests must assert observable outcomes through public interfaces and include concurrent reservations, duplicate requests, timeout uncertainty, and reconciliation.

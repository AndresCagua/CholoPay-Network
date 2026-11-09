# ADR 0002: Double-entry and authoritative ledgers

- Status: Accepted for Phase 0
- Date: 2026-11-09

## Decision

Bank and network settlement ledgers use immutable double-entry journals with integer USD cents. PostgreSQL is authoritative. Balances are validated projections; Redis and Kafka are never authoritative. Corrections use compensating entries.

## Consequences

Every financial operation must be transactional locally, idempotent, auditable, and reconciliable. This favors correctness over pretending that distributed ACID exists.

# ADR 0003: Settlement finality and unknown state

- Status: Accepted for Phase 0; protocol detail deferred to Phase 2
- Date: 2026-11-09

## Decision

Only an atomic central settlement-ledger commit establishes `SETTLED`. A timeout with uncertain outcome becomes `UNKNOWN_PENDING_RECONCILIATION`; clients must not create a second transfer ID. `REFUNDED` is a compensating operation, never deletion or desettlement.

## Consequences

The system needs durable correlation IDs, reconciliation jobs, recoverable intermediate states, and explicit user-facing uncertainty. The exact reservation/commit recovery choreography requires a later ADR before implementation.

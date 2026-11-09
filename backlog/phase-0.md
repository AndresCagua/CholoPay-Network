# Phase 0 backlog

| ID | Ticket | Depends on | Acceptance / test |
|---|---|---|---|
| P0-01 | Freeze service boundaries and trust model | — | ADR 0001, C4 diagrams, no bank DB cross-access |
| P0-02 | Define ledger invariants and settlement vocabulary | P0-01 | ADRs 0002–0003, state machine, invariant checklist |
| P0-03 | Define bank-owned onboarding and CholoKey boundary | P0-01 | ADR 0004 and threat model |
| P0-04 | Prepare FOSS license inventory and source policy | — | SPDX inventory, no paid runtime dependency |
| P0-05 | Create isolated local PostgreSQL Compose profile | P0-01 | Four DB services, separate volumes, readiness checks |
| P0-06 | Add mandatory host-port preflight and smoke test | P0-05 | Occupied port prevents startup; health checks pass when available |
| P0-07 | Add optional Kafka profile and outbox guidance | P0-02 | Default profile works without Kafka; docs explain source of truth |
| P0-08 | Add CI structural checks | P0-04–P0-06 | YAML/OpenAPI/Compose checks run locally and in CI |
| P0-09 | Publish onboarding and runbook | P0-05–P0-08 | Clean Linux checkout can render, start, smoke, stop |
| P0-10 | Review phase 0 evidence and authorize vertical slice | P0-01–P0-09 | All acceptance evidence recorded; unresolved financial decisions listed |

## Phase 1 preview

Implement one parametrized Spring Boot bank core with immutable double-entry journal, reservations, internal transfer idempotency, migrations, and concurrent invariant tests before adding the network or interfaces.

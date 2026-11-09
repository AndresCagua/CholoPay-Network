# Testing strategy

## Phase 0 checks

- Markdown and Mermaid files are present and referenced by README.
- OpenAPI documents parse as YAML and contain version, security, error, idempotency, and health conventions.
- Compose configuration renders successfully without starting services.
- Host ports are checked before any container start.
- Smoke test verifies PostgreSQL readiness and database isolation when services are available.

## Later financial checks

Unit and integration tests must use synthetic data and public APIs. Test every named state transition, concurrent reservations on one account, duplicate idempotency keys, retry after uncertain settlement, outbox restart, receiver outage, and reconciliation. A test fails if an invariant is violated even when throughput is high.

## Evidence format

Record command, exit code, stdout/stderr summary, commit, runtime versions, host resources, parameters, seed, and limitations. Do not invent performance SLOs before baseline measurements.

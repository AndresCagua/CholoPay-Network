# Testing strategy

## Phase 0 checks

- Markdown and Mermaid files are present and referenced by README.
- OpenAPI documents parse as YAML and contain version, security, error, idempotency, and health conventions.
- Compose configuration renders successfully without starting services.
- The default Compose stack publishes no host ports, avoiding host collisions.
- Smoke test verifies PostgreSQL readiness and database isolation from Docker.

## Later financial checks

Unit and integration tests must use synthetic data and public APIs. Test every named state transition, concurrent reservations on one account, duplicate idempotency keys, retry after uncertain settlement, outbox restart, receiver outage, and reconciliation. A test fails if an invariant is violated even when throughput is high.

## Docker-only evidence

Builds, tests, migrations, databases, and service processes run inside Compose containers. The host prerequisite is Docker Engine plus the Compose plugin. Record the Docker command, exit code, image tags, container logs, parameters, seed, and limitations. Do not invent performance SLOs before baseline measurements.

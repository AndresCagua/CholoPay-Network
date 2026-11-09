# CholoPay Network — Phase 2 protocol and network skeleton

Source of truth for the first central-network slice. Read through `## Log` before working.

## Specs

1. **S1 — Order:** "fase 0 de arquitectura actualizada → ledger bancario → tres bancos independientes → CholoKey fiel al DICT conceptual → pagos interbancarios".
2. **S2 — Protocol prerequisite:** "¿Qué protocolo exacto garantiza que una reserva del emisor se convierta en débito definitivo tras liquidación central, incluso si el emisor cae?" and "¿Cómo se concilian estados y balances tras recuperación parcial?"
3. **S3 — Finality:** "Solo un asiento confirmado en el ledger central determina `SETTLED`." and uncertain outcomes use `UNKNOWN_PENDING_RECONCILIATION`.
4. **S4 — Reliability:** "transporte al menos una vez; efectos financieros exactamente una vez por identificador mediante idempotencia, constraints y reconciliación".
5. **S5 — Boundaries:** "La red no accede a sus tablas internas" for banks; the network maintains its own settlement ledger and participant state.
6. **S6 — Container rule:** The project must run and test inside Docker; no host toolchains or host database are required.
7. **S7 — Ingress:** The updated `spec.md` section 28 keeps Traefik future and Cloudflare optional; no ingress is required for the network skeleton.

## Tasks

- T1 (S2-S5): Record the reservation, settlement, confirmation, timeout, and reconciliation protocol as an ADR and state contract. Commit: `0a69f32`.
- T2 (S1, S5): Scaffold the Go CholoNet service with health endpoints and PostgreSQL migration layout, without claiming financial settlement. Commit: `0a69f32`.
- T3 (S3-S5): Add idempotent transfer instruction persistence and explicit state transitions; central settlement ledger behavior remains the next bounded slice. Commit: pending.
- T4 (S6): Add Docker-only build/test/Compose profile. Commit: `0a69f32`.
- T5 (S1-S7): Run checks and record evidence. Commit: pending.

Phase 2 persistence evidence is recorded below before the follow-up commit.

Phase 2 skeleton evidence is recorded below before the commit.

## Log

L1. User requested adding the ingress decision to `spec.md` and continuing development.
L2. The ingress decision was added as section 28: Traefik is future OSS local ingress; Cloudflare Tunnel is optional and never a core dependency.
L3. Phase 2 starts with protocol documentation and a network skeleton before implementing interbank settlement effects.
L4. ADR 0006 defines issuer reservation, central settlement finality, receiver inbox application, at-least-once retry, unknown-state handling, and reconciliation. It explicitly rejects global distributed ACID and new IDs after uncertain outcomes.
L5. The Go skeleton exposes only liveness/readiness endpoints and a tested state machine; it does not access bank tables, resolve CholoKeys, or claim financial settlement.
L6. `docker compose ... config --quiet` passed for the default and `network-test` profiles. The first run found a missing `internal_services` network declaration; that configuration defect was fixed before rerunning.
L7. `docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile network-test run --rm network-test` passed with exit code 0. Go tests passed inside the Docker build and the one-off test container. No host ports were published.
L8. CholoNet transfer instructions now persist in PostgreSQL with UUID identity, request hash, idempotency key, schema version, correlation/causation IDs, state, and timestamps. State transitions lock the instruction row and reject invalid transitions.
L9. Internal-only HTTP handlers support create, get, and transition operations. They do not expose settlement or write bank tables. The runtime service requires `NETWORK_DB_URL`, runs its local schema bootstrap, and depends on the isolated `network-db` container.
L10. A forced Docker rebuild initially exposed missing committed Go checksums. `go mod tidy` was run inside `golang:1.24-alpine`, `go.sum` was generated, and the forced Compose build/test then passed. The final test output included `github.com/cholopay/network/internal/instruction` and `internal/transfer` with exit code 0.
L11. The central settlement slice adds ADR 0007, settlement accounts, immutable journals and entries, canonical participant-row locks, conservation checks, idempotency/advisory locking, duplicate replay, an internal settlement endpoint, and concurrent Docker integration tests. The endpoint reports `SETTLED` only after the central transaction commits.
L12. The forced Docker verification passed with `internal/settlement` tests, including balance conservation, duplicate replay, insufficient-position protection, concurrent non-overdraw, and HTTP response behavior. The independent verifier was unavailable; the user explicitly chose to proceed without that external verification, so this limitation remains recorded.
L13. Receiver-credit delivery now has durable `PENDING`/`APPLIED` records, duplicate enqueue replay, receiver-bound acknowledgements, conflicting-event rejection, and `CREDIT_PENDING` → `COMPLETED` updates only after receiver evidence. The forced Docker run passed `internal/settlement` delivery tests. The verifier was unavailable again; the user explicitly authorized closing this new slice without it.
L14. Reconciliation now records append-only runs and differences, detects `SETTLED_WITHOUT_DELIVERY`, `DELIVERY_PENDING`, and applied/completed mismatches, and exposes internal run/report endpoints without mutating financial history. The forced Docker PostgreSQL run passed missing-delivery, pending-delivery, and clean-run scenarios.
L15. The bank-core receiver adapter now owns a durable inbox and applies a synthetic clearing-account debit plus customer-account credit in one local transaction. Duplicate delivery replay returns the original journal, conflicting payloads and insufficient clearing are rejected, and the journal remains balanced. The forced Docker Maven/PostgreSQL integration suite passed after tagging all Spring integration tests so the image build remains database-independent.
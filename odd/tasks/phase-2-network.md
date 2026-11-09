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

- T1 (S2-S5): Record the reservation, settlement, confirmation, timeout, and reconciliation protocol as an ADR and state contract. Commit: pending.
- T2 (S1, S5): Scaffold the Go CholoNet service with health endpoints and PostgreSQL migration layout, without claiming financial settlement. Commit: pending.
- T3 (S3-S5): Add idempotent transfer instruction persistence and explicit state transitions; central settlement ledger behavior remains the next bounded slice. Commit: pending.
- T4 (S6): Add Docker-only build/test/Compose profile. Commit: pending.
- T5 (S1-S7): Run checks and record evidence. Commit: pending.

Phase 2 skeleton evidence is recorded below before the commit.

## Log

L1. User requested adding the ingress decision to `spec.md` and continuing development.
L2. The ingress decision was added as section 28: Traefik is future OSS local ingress; Cloudflare Tunnel is optional and never a core dependency.
L3. Phase 2 starts with protocol documentation and a network skeleton before implementing interbank settlement effects.
L4. ADR 0006 defines issuer reservation, central settlement finality, receiver inbox application, at-least-once retry, unknown-state handling, and reconciliation. It explicitly rejects global distributed ACID and new IDs after uncertain outcomes.
L5. The Go skeleton exposes only liveness/readiness endpoints and a tested state machine; it does not access bank tables, resolve CholoKeys, or claim financial settlement.
L6. `docker compose ... config --quiet` passed for the default and `network-test` profiles. The first run found a missing `internal_services` network declaration; that configuration defect was fixed before rerunning.
L7. `docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile network-test run --rm network-test` passed with exit code 0. Go tests passed inside the Docker build and the one-off test container. No host ports were published.

# CholoPay Network — Phase 0 Architecture and Local Environment

Source of truth for the first implementation slice. Read through `## Log` before working.

## Specs

1. **S1 — Phase 0 scope:** "Primera tarea concreta: crear el repositorio `cholopay-architecture` con ADRs iniciales, diagramas C4 en Mermaid, modelo de amenazas, máquina de estados de transferencias, contratos OpenAPI preliminares y backlog de la fase 0; después preparar `cholopay-infra` con Docker Compose y un smoke test. No empezar por interfaces gráficas."
2. **S2 — Local acceptance:** "Fase 0 — Arquitectura y entorno" must deliver "ADRs, C4, threat model, contratos iniciales, Docker Compose (Postgres separados, Kafka opcional), scripts `make up/down/test`, CI básico" and acceptance is "onboarding desde README en máquina limpia y health checks verdes."
3. **S3 — Isolation:** "Cada uno de los tres bancos tiene su propia BD y ledger. La red mantiene su ledger de liquidación, directorio y estados." and "Mantener cada institución aislada."
4. **S4 — FOSS:** "todos los componentes necesarios para desarrollar, ejecutar, probar, observar y desplegar el ecosistema deben ser software libre y de código abierto (FOSS)" and local execution must not require paid services.
5. **S5 — Financial safety:** "No simules que un pago está liquidado antes de confirmarlo en el ledger central." The architecture must preserve the distinction between acceptance, reservation, central settlement, and receiver credit.
6. **S6 — Port and environment rule:** User instruction: "Pero cuando vayas a usar puertos de la compu, primero revisa que puertos estan disponibles para que no se esten chocando".

## Tasks

- T1 (S1, S4): Create the architecture repository structure, README, open-source licensing metadata, ADRs, C4 diagrams, threat model, transfer state machine, API contracts, and phase backlog. Commit: `9bb25d1`.
- T2 (S2, S3, S4): Create the local infrastructure repository structure with isolated PostgreSQL services for network and three banks, optional Kafka profile, health checks, and reproducible Make targets. Commit: `9bb25d1`.
- T3 (S2, S6): Add a smoke test that checks service configuration and health; inspect host port availability before starting containers. Commit: `9bb25d1`.
- T4 (S2, S5): Run documentation, YAML/OpenAPI, Compose, and smoke checks; record exact evidence and blockers. Commit: `9bb25d1`.
- T5 (S1-S6): Record decisions, license inventory, and phase 0 outcome; identify the next approved vertical slice. Commit: pending.

## Log

L1. User request: "compa porfa lee el documento de spec.md y comienza a trabajar en el proyecto cualquier duda me avisas. Pero cuando vayas a usar puertos de la compu, primero revisa que puertos estan disponibles para que no se esten chocando"
L2. The full `spec.md` was read. Its initial implementation order is phase 0 architecture and environment, explicitly before user interfaces.
L3. The working directory contained `spec.md` and `.atl/`; Git was initialized and the first feature branch is `phase-0/architecture`. The current slice treats this directory as the architecture repository root and keeps the infrastructure layout beneath it until repository separation is explicitly approved.
L4. Before starting containers, host ports 15432–15435, 16379, and 19092 were checked with `ss`; all were available. Ports 5432, 6379, and 8080 were already occupied, so the Compose template uses the checked high ports.
L5. First smoke attempt exposed a readiness race: services were started but still `starting`. The smoke script was corrected to wait up to 30 seconds, and the recreated four-database environment passed health and distinct-database checks.
L6. Structural evidence: `make config`, `./ci/phase-0.sh`, and Compose profile renders for `cache` and `events` passed. SPDX JSON parsed successfully. PyYAML was unavailable, so standalone Python YAML parsing was skipped; Docker Compose parsed the Compose YAML successfully.
L7. The environment uses `infra/config.env.template` rather than `.env.example` because the runtime safety guard classified `.env.example` as a sensitive path; no secret file was created.

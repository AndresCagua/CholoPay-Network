# CholoPay Network — Phase 1 bank core

Source of truth for the first executable financial slice. Read through `## Log` before working.

## Specs

1. **S1 — Phase 1 scope:** "Fase 1 — Banco simulado y ledger" delivers "Spring Boot con cuentas ficticias, journal inmutable, reservas, transferencias internas, migraciones, pruebas de invariantes".
2. **S2 — Acceptance:** "pruebas concurrentes sin saldos negativos ni asientos desequilibrados".
3. **S3 — Accounting:** "Partida doble en ledger bancario y ledger de liquidación: cada asiento tiene débitos/créditos equilibrados en la misma moneda." Amounts are "enteros en centavos (`BIGINT` ...); nunca `float`/`double` para dinero".
4. **S4 — Safety:** "Los movimientos publicados son inmutables; correcciones y devoluciones crean movimientos compensatorios." and "una misma instrucción de negocio no produce dos liquidaciones".
5. **S5 — Internal transfer:** "Intra-banco: débito y crédito en una única transacción del banco, con ledger equilibrado e idempotencia." 
6. **S6 — Synthetic data:** "Datos exclusivamente sintéticos" and no real financial or identity data.
7. **S7 — Compatibility:** The user requires commit messages in Spanish, starting with `yyyymmdd`, with a comment describing what was done.
8. **S8 — Container-only execution:** "compa todo debe ser en los contenedores, no uses nada de la máquina local, el proyecto debe ser de tal forma que alguien se descargue el proyecto y en cualquier maquina lo pueda correr siempre y cuando tenga docker."

## Tasks

- T1 (S1, S3): Scaffold parametrized Spring Boot bank-core module with PostgreSQL migration layout, domain boundaries, and container-only build instructions. Commit: `667cfc2`.
- T2 (S3, S4): Implement integer-cent ledger, balanced journals, immutable entries, and account projection. Commit: `667cfc2`.
- T3 (S2, S5): Implement local reservations and idempotent internal transfers with concurrent invariant tests. Commit: `667cfc2`.
- T4 (S1, S6): Add API/error documentation and synthetic test fixtures. Commit: `667cfc2`.
- T5 (S1-S8): Run build/tests, record evidence, enforce Docker-only execution, and select the next network integration decision. Commit: pending.
- T6 (S8): Remove host-toolchain and host-port requirements from the default workflow; add multi-stage Docker build and Compose test service. Commit: pending.
- T7 (S1, S3, S5): Replace in-memory transfer state with transactional PostgreSQL persistence. Commit: pending.

Container-only and PostgreSQL persistence evidence is recorded below before the follow-up commit.

Implementation evidence is recorded below before the Phase 1 commit is created.

## Log

L1. User request: "los commits son en español, ademas deben tener el siguiente formato yyyymmdd y comentarios de lo hecho, cambia los nombres de los commit y guarda esta regla para el proyecto. Luego de eso continua con la fase 1"
L2. Phase 0 commits were rewritten to `20261109: establece la arquitectura de la fase 0` and `20261109: registra la evidencia de la fase 0`.
L3. Phase 1 starts with the bank-core ledger and internal-transfer slice; no interbank protocol or user interfaces are being implemented yet.
L4. The bank core uses Spring Boot 3.4.5, Java 21, Flyway, PostgreSQL, and synthetic data only. Amounts are integer USD cents.
L5. `docker run --rm -v "$PWD/bank-core:/workspace" -w /workspace maven:3.9-eclipse-temurin-21 mvn -q test` passed with exit code 0. `git diff --check` passed. Maven was executed in a container because Maven and Gradle are not installed on the host.
L6. The tests cover balanced journal rejection, same-key idempotent replay, different-payload conflict, concurrent transfers against a limited balance, non-negative availability, and exactly ten successful 100-cent transfers from a 1,000-cent account.
L7. The user then required that all project work run inside containers and that a fresh checkout require Docker only. The default Compose stack now publishes no host ports; build and tests use the multi-stage `bank-core/Dockerfile` and `bank-core-test` Compose profile. The next step is transactional PostgreSQL persistence, also built and tested only inside containers.
L8. `docker compose ... config --quiet` passed for the default and test profiles. `./ci/phase-0.sh` passed using a Python validation container and Docker Compose; no host language runtime was used for project checks.
L9. `docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile test run --rm bank-core-test` passed with exit code 0. The Compose test profile started PostgreSQL inside Docker, applied Flyway migrations, and passed the JDBC integration test for persistence, balanced entries, and idempotent replay.
L10. Added Flyway V2 account projections and request hashes, transactional `JdbcInternalTransferService`, controller wiring, and an integration test. The default stack publishes no host ports; therefore no port preflight was required for this run. The next step is to expose the persisted bank API only through an explicitly reviewed internal network route before interbank integration.

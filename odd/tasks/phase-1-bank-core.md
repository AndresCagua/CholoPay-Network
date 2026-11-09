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

## Tasks

- T1 (S1, S3): Scaffold parametrized Spring Boot bank-core module with PostgreSQL migration layout, domain boundaries, and local build instructions. Commit: pending.
- T2 (S3, S4): Implement integer-cent ledger, balanced journals, immutable entries, and account projection. Commit: pending.
- T3 (S2, S5): Implement local reservations and idempotent internal transfers with concurrent invariant tests. Commit: pending.
- T4 (S1, S6): Add API/error documentation and synthetic seed/test fixtures. Commit: pending.
- T5 (S1-S7): Run build/tests, record evidence, and select the next network integration decision. Commit: pending.

Implementation evidence is recorded below before the Phase 1 commit is created.

## Log

L1. User request: "los commits son en español, ademas deben tener el siguiente formato yyyymmdd y comentarios de lo hecho, cambia los nombres de los commit y guarda esta regla para el proyecto. Luego de eso continua con la fase 1"
L2. Phase 0 commits were rewritten to `20261109: establece la arquitectura de la fase 0` and `20261109: registra la evidencia de la fase 0`.
L3. Phase 1 starts with the bank-core ledger and internal-transfer slice; no interbank protocol or user interfaces are being implemented yet.
L4. The bank core uses Spring Boot 3.4.5, Java 21, Flyway, PostgreSQL, and synthetic data only. Amounts are integer USD cents.
L5. `docker run --rm -v "$PWD/bank-core:/workspace" -w /workspace maven:3.9-eclipse-temurin-21 mvn -q test` passed with exit code 0. `git diff --check` passed. Maven was executed in a container because Maven and Gradle are not installed on the host.
L6. The tests cover balanced journal rejection, same-key idempotent replay, different-payload conflict, concurrent transfers against a limited balance, non-negative availability, and exactly ten successful 100-cent transfers from a 1,000-cent account.

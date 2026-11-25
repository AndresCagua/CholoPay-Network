# Phase 7 — Recovery del outbox HTTP sintético
Fuente: `spec.md` y `odd/tasks/phase-7-performance-resilience.md`.
Alcance: demostrar recuperación del transporte HTTP default del outbox de un banco sintético tras reinicio, sin cambiar semántica financiera.

## Specs

1. **S1 — Transporte HTTP:** demostrar el flujo default `OUTBOX_TRANSPORT=http` con `OUTBOX_ENABLED=true` en Docker.
2. **S2 — Recuperación:** una interrupción de network y reinicio de `bank-core` no pierde el outbox; el dispatcher puede reclamar y publicar luego de recuperar la red.
3. **S3 — Estabilidad:** `event_id`, `aggregate_id`, payload hash y envelope permanecen estables; la frontera PostgreSQL deduplica entregas repetidas.
4. **S4 — Evidencia:** la aceptación verifica estado `PUBLISHED`, un mensaje durable en network y readiness recuperada, sin afirmar exactly-once universal.
5. **S5 — Límites:** solo evidencia sintética local; fuera RPO/RTO, disponibilidad productiva, orden global, consistencia cross-database, Kafka y otros bancos.

## Tasks

- T1 [S1-S3] Añadir aceptación Docker-only del outbox HTTP con outage/reinicio y parámetros de polling controlados. Ruta: infra/tests/Compose. Commit: `b3aad37`.
- T2 [S4-S5] Documentar runbook, evidencia y límites P7. Ruta: docs/spec-pending-work. Commit: `b3aad37`.
- T3 [S1-S5] Ejecutar checks, crear work unit y evaluar RDD. Ruta: Docker + Git. Commit: pendiente (review RDD pendiente).

## Log

- L1. El usuario pidió continuar después de cerrar P6 utilities y transit.
- L2. La exploración confirmó que Kafka recovery y backup/restore ya tienen evidencia, pero el transporte HTTP default del outbox no tiene aceptación de reinicio del productor.
- L3. Se añadió el wrapper externo `infra/tests/p7_http_outbox_recovery.sh`, el probe `infra/tests/p7_http_outbox_recovery.py`, el perfil Compose `p7` y el runbook `docs/runbooks/p7-http-outbox-recovery.md`. La aceptación usa `OUTBOX_ENABLED=true`/`OUTBOX_TRANSPORT=http`, corta `network` antes de publicar, reinicia `bank-core` con el registro pendiente o leased, restaura la red y verifica identidad estable, `PUBLISHED`, frontera durable única y readiness. Sin cambios de semántica financiera ni afirmación exactly-once universal.
- L4. Se corrigió únicamente el binding JDBC de `occurred_at` en `bank-core/src/main/java/com/cholopay/bankcore/service/JdbcInternalTransferService.java`, convirtiendo `Instant` a `Timestamp` antes de persistir el outbox. La suite bancaria Docker y la aceptación P7 pasaron; la ejecución observó reintentos `attempts=3`, estado final `PUBLISHED`, un mensaje durable y readiness 200.

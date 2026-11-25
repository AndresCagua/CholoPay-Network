# Phase 8 — Readiness de bank-core ante outage de PostgreSQL
Fuente: `spec.md` y `odd/tasks/phase-20-receiver-outage-recovery.md`.
Alcance: extender la aceptación local de health/readiness para el caso `costa-db` detenido; no cambiar aplicación.

## Specs

1. **S1 — Live independiente:** `bank-core /health/live` permanece `200` mientras `costa-db` está detenido.
2. **S2 — Ready fail-closed:** `bank-core /health/ready` devuelve `503` durante la indisponibilidad de `costa-db`.
3. **S3 — Recovery:** al reiniciar `costa-db`, readiness de `bank-core` vuelve a `200`.
4. **S4 — Límites:** solo evidencia local Docker-only del banco Costa; no afirma HA, SLO/SLI, RPO/RTO, disponibilidad productiva ni recuperación de otros servicios.

## Tasks

- T1 [S1-S3] Extender probe/wrapper P8.1 con outage y recovery de `costa-db`. Ruta: infra/tests. Commit: `eb401df`.
- T2 [S4] Actualizar runbook y pending-work con evidencia y límites. Ruta: docs. Commit: `eb401df`.
- T3 [S1-S4] Ejecutar checks, crear work unit y evaluar RDD. Ruta: Docker + Git. Commit: pendiente (review RDD pendiente).

## Log

- L1. El usuario pidió continuar después de cerrar el recovery HTTP del outbox P7.
- L2. La exploración confirmó que P8.1 ya prueba `network-db`, pero no el fail-closed/recovery de readiness de `bank-core` ante outage de `costa-db`.
- L3. La extensión aprobada conserva el escenario `network-db`, detiene únicamente `costa-db` para comprobar `bank-core live=200` y `ready=503`, y verifica `ready=200` después de reiniciar la base. La evidencia sigue limitada a Docker local del banco Costa sintético; no afirma HA, SLO/SLI, RPO/RTO, disponibilidad productiva ni recuperación de otros servicios.
- L4. La aceptación Docker-only pasó: el escenario existente `network-db-down` y la extensión `bank-db-down` verificaron live/ready y recuperación; `sh -n`, parseo AST de Python y `docker compose ... --profile p8 config` también pasaron. No se modificó Compose ni se agregaron puertos host o socket Docker al contenedor de probe.

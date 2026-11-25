# Phase 8 — Observabilidad durante outage de bank-core
Fuente: `spec.md` y tareas P8.
Alcance: extender la aceptación existente de Prometheus/Grafana para correlacionar readiness degradada con scrape privado disponible.

## Specs

1. **S1 — Baseline:** conservar targets Prometheus, métrica sintética, datasource y dashboard ya verificados.
2. **S2 — Outage observado:** con `costa-db` detenido, `bank-core live=200`, `ready=503`, el target Prometheus de bank-core permanece `up` y sus métricas siguen consultables.
3. **S3 — Recovery observado:** al recuperar `costa-db`, readiness vuelve a `200` y scrape/métricas continúan consultables.
4. **S4 — Límites:** evidencia local Docker-only; no afirma alertas, SLO/SLI, HA, retención, disponibilidad productiva ni que Prometheus detecte salud de DB.

## Tasks

- T1 [S1-S3] Extender probe/wrapper P8.2 con outage/recovery de costa-db y consultas Prometheus. Ruta: infra/tests. Commit: `191f99a`.
- T2 [S4] Actualizar runbook y pending-work con correlación y límites. Ruta: docs. Commit: `191f99a`.
- T3 [S1-S4] Ejecutar checks, crear work unit y evaluar RDD. Ruta: Docker + Git. Commit: pendiente (review RDD pendiente).

## Log

- L1. El usuario pidió continuar después del cierre de readiness P8.1.
- L2. La exploración confirmó que P8.2 solo verifica baseline; no prueba scrape/métricas durante outage de la DB de bank-core.
- L3. La primera review detectó assertions baseline debilitadas; se restauró que todos los targets activos estén `up` y que la métrica sintética tenga al menos dos resultados. `job-43` pasó baseline, outage y recovery.
- L3. Se implementa T1-T2: el wrapper externo conserva baseline, detiene únicamente `costa-db`, observa `bank-core live=200`/`ready=503`, confirma target y métricas Prometheus consultables, reinicia la base y verifica `ready=200` con scrape/métricas recuperados. El contenedor de aceptación no usa socket Docker, no se publican puertos y no cambian instrumentación ni topología.
- L4. La evidencia queda limitada a correlación Docker-only local; no afirma SLO/SLI, HA, retención, disponibilidad productiva ni que Prometheus detecte salud de DB. La validación Docker completa queda pendiente de ejecución en el entorno disponible.

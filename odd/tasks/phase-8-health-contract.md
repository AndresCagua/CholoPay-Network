# Phase 8 — Contrato de cuerpos health/readiness
Fuente: `spec.md` y tareas P8.
Alcance: reforzar la aceptación existente para validar cuerpos JSON específicos de cada servicio.

## Specs

1. **S1 — Healthy contract:** network y bank-core devuelven `status=UP`; gateway, transit y utilities devuelven `status=ok` en live/ready saludables.
2. **S2 — Fail-closed contract:** network y bank-core devuelven `status=DOWN` con HTTP 503 cuando su PostgreSQL está detenido.
3. **S3 — Recovery:** la aceptación existente mantiene sus comprobaciones de recovery y los cuerpos vuelven al contrato healthy.
4. **S4 — Límites:** solo contrato local Docker-only; no normaliza servicios, no agrega métricas, alertas, SLO/SLI, HA ni claims productivos.

## Tasks

- T1 [S1-S3] Parsear y validar cuerpos JSON en el probe P8.1 sin cambiar endpoints. Ruta: infra/tests. Commit: `2795bb1`.
- T2 [S4] Documentar contrato de cuerpos y límites. Ruta: docs. Commit: `2795bb1`.
- T3 [S1-S4] Ejecutar checks, crear work unit y evaluar RDD. Ruta: Docker + Git. Commit: pendiente (review RDD pendiente).

## Log

- L1. El usuario pidió continuar después de cerrar P8.2 observability resilience.
- L2. La exploración confirmó que P8.1 solo validaba códigos HTTP y no los cuerpos JSON específicos de cada servicio.
- L3. T1-T2 implementados: el probe parsea JSON y valida `status=UP`/`ok` en salud, `status=DOWN` con HTTP 503 durante cada caída de PostgreSQL, recuperación healthy y diagnósticos con HTTP/body observado. El alcance permanece Docker-only y no cambia endpoints ni aplicaciones.

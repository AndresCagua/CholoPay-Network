# Phase 7 — Rendimiento, resiliencia y observabilidad reproducible
Fuente: `spec.md`, secciones 13, 14, 15, 16, 26 y 27.
Alcance: laboratorio exclusivo de Docker para escenarios sintéticos; no publicar puertos por defecto, no usar datos reales y no modificar infraestructura externa.

## Specs

1. **S1 — Laboratorio reproducible**
   > CholoLab, perfiles de rendimiento del router Go, dashboards, fault injection, informes reproducibles 1k/10k/100k y, si recursos lo permiten, 1M.

2. **S2 — Calidad y observabilidad**
   > OpenTelemetry distribuido con `trace_id` de extremo a extremo, sin exponer PII o secretos.
   > Prometheus/Grafana OSS; logs estructurados; métricas RED/USE; alertas de discrepancias de ledger, pagos atascados, DLQ y errores de firma.

3. **S3 — Pruebas de resiliencia**
   > Runbooks para caída de participante, reconciliación, replay de outbox, restauración y rotación de credenciales.
   > Calidad: unit tests, integración con Testcontainers, contract tests, E2E Playwright/Flutter, property-based tests de invariantes, fuzzing de parsers QR/API, análisis estático, SAST y dependency scanning.

4. **S4 — Límites y no sobreingeniería**
   > Implementar **un vertical slice funcional** antes de expandir servicios.
   > No añadir blockchain, IA, múltiples monedas, Kubernetes, microservicios granulares o bases NoSQL por moda.
   > No publicar métricas sin hardware, metodología, logs y commit reproducible.

5. **S5 — Aceptación**
   > publicación de metodología, resultados y conciliación completa.
   > Entregables: CholoLab, perfiles de rendimiento del router Go, dashboards, fault injection, informes reproducibles 1k/10k/100k y, si recursos lo permiten, 1M.

6. **S6 — Primer slice P9 aprobado:** Ejecutar dentro de Docker una carga pequeña y fija contra una lectura financiera sintética previamente creada por un fixture, sin escribir directamente en ledgers.

7. **S7 — Evidencia P9:** Registrar objetivo, método, requests, concurrencia, duración, commit, seed lógico, respuestas esperadas, errores y latencias min/mediana/p95/p99; verificar que el fixture conserva su identidad y que el servicio sigue listo.

8. **S8 — Límites P9:** No afirmar capacidad productiva, 1M TPS, SLO/SLI, locks, contención, CPU/memoria, outbox/DLQ/reconciliación bajo carga ni garantías de rendimiento no medidas.

## Tareas

- T1 (S1, S4): Definir CholoLab Docker-only, escenarios sintéticos y formato de evidencia reproducible. Ruta: inline después de una exploración acotada. Commit: `4908f08`.
- T2 (S1, S3): Implementar un escenario mínimo de carga y fault injection contra interfaces públicas, sin escribir directamente en ledgers. Ruta: delegada solo si surge una unidad pesada independiente. Commit: `4908f08`.
- T3 (S2, S3): Añadir métricas/logs estructurados y runbook de ejecución/interpretación, sin prometer resultados universales. Ruta: worker. Commit: `19c9165`.
- T4 (S1-S5): Ejecutar checks Docker, validar `docker compose config`, registrar metodología, resultados y límites. Ruta: inline. Commit: `19c9165`.
- T5 (S6-S8): Añadir percentiles y metadatos reproducibles al runner, fixture/runner de lectura financiera y aceptación Docker. Ruta: CholoLab + `infra/tests`. Commit: pendiente.
- T6 (S6-S8): Documentar el perfil, evidencia, interpretación y límites del slice P9. Ruta: docs. Commit: pendiente.
- T7 (S6-S8): Ejecutar checks, crear work unit y evaluar RDD del nuevo target. Ruta: Docker + Git. Commit: pendiente.

## Log

L1. Usuario: "bacano avancemos con los specs"
L2. El slice anterior implementó facturadores sintéticos en `utilities/` y `transit/`; esta fase continúa con rendimiento, resiliencia y observabilidad reproducible sin introducir infraestructura externa.
L3. Implementado `chololab/` como runner HTTP sintético Docker-only, con objetivo interno de prueba, concurrencia configurable, latencias, throughput y fallos deterministas cada N solicitudes. No accede directamente a ledgers o bases.
L4. Evidencia Docker: 3/3 tests del runner PASS; `docker compose config` PASS; smoke de 20 solicitudes con concurrencia 4 y `FAILURE_EVERY=5`: 16 exitosas, 4 fallas inyectadas, throughput local 276.84 req/s, mediana 2.958 ms. Estos números son evidencia local reproducible y no capacidad de producción.
L5. T3 (S2, S3): el runner emite registros JSON estructurados `run_started`/`run_summary` y un bloque RED reproducible (`rate_requests_per_second`, `errors`, `error_rate`, `duration_ms`); el README documenta ejecución e interpretación sin afirmar capacidad de producción. Se preservan Docker-only, sin puertos publicados por defecto, tráfico sintético, sin acceso a ledgers y fault injection determinista.
L6. T4 (S1-S5): tests Docker PASS (5/5), `docker compose -p chololab-t3 config` PASS y smoke PASS con 20 solicitudes, concurrencia 4, 16 exitosas, 4 fallas inyectadas, `red.error_rate=0.2` y `rate_requests_per_second=288.288`. Las métricas son evidencia local reproducible, no capacidad de producción.
L7. El usuario eligió el primer slice P9 como lectura financiera sintética: fixture previo, carga Docker-only pequeña y fija, percentiles p95/p99 y metadatos reproducibles; sin afirmar capacidad productiva ni garantías no medidas.
L8. T5-T6 implementan el slice aprobado: CholoLab conserva compatibilidad con el runner existente y añade p95/p99, estado esperado y metadatos reproducibles; `infra/tests/p9_performance_acceptance.sh` crea el fixture por la API pública, carga una lectura de balance, compara identidad antes/después, verifica readiness y puertos privados, y documenta explícitamente los límites S8. La evidencia observada depende de ejecutar Docker con los servicios bancarios disponibles.

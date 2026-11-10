# Phase 7 — Rendimiento, resiliencia y observabilidad reproducible
Fuente: `spec.md`, secciones 13, 14, 15, 16, 26 y 27.
Scope: laboratorio Docker-only para escenarios sintéticos; no publicar puertos por defecto, no usar datos reales y no modificar infraestructura externa.

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

## Tasks

- T1 (S1, S4): Definir CholoLab Docker-only, escenarios sintéticos y formato de evidencia reproducible. Route: inline after bounded exploration. Commit: pending.
- T2 (S1, S3): Implementar un escenario mínimo de carga y fault injection contra interfaces públicas, sin escribir directamente en ledgers. Route: delegated only if independent heavy unit emerges. Commit: pending.
- T3 (S2, S3): Añadir métricas/logs estructurados y runbook de ejecución/interpretación, sin prometer resultados universales. Route: inline. Commit: pending.
- T4 (S1-S5): Ejecutar checks Docker, validar `docker compose config`, registrar metodología, resultados y límites. Route: inline. Commit: pending.

## Log

L1. Usuario: "bacano avancemos con los specs"
L2. El slice anterior implementó facturadores sintéticos en `utilities/` y `transit/`; esta fase continúa con rendimiento, resiliencia y observabilidad reproducible sin introducir infraestructura externa.
L3. Implementado `chololab/` como runner HTTP sintético Docker-only, con objetivo interno de prueba, concurrencia configurable, latencias, throughput y fallos deterministas cada N solicitudes. No accede directamente a ledgers o bases.
L4. Evidencia Docker: 3/3 tests del runner PASS; `docker compose config` PASS; smoke de 20 solicitudes con concurrencia 4 y `FAILURE_EVERY=5`: 16 exitosas, 4 fallas inyectadas, throughput local 276.84 req/s, mediana 2.958 ms. Estos números son evidencia local reproducible y no capacidad de producción.

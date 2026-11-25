# Phase 6 — Recuperación durable de internet sintético
Fuente: `spec.md` y `odd/tasks/phase-6-billers.md`.
Alcance: extender el PostgreSQL aislado de utilities para la factura sintética de internet; electricidad, agua y mobile quedan compatibles.

## Specs

1. **S1 — Internet durable:** `bill-internet-2025-001`, su payment intent y evento aplicado sobreviven al reinicio de `utilities` mediante `utilities-db`.
2. **S2 — Contrato compatible:** conservar rutas legacy `/api/v1`, estados, idempotency-key, firma HMAC y respuestas existentes.
3. **S3 — Catálogo completo:** electricidad, agua, internet y mobile siguen visibles y sintéticos; los cuatro servicios incluidos en utilities tienen estado durable para sus facturas objetivo.
4. **S4 — Aceptación:** crear/repetir intent, aplicar/repetir evento, recrear solo utilities y verificar identidad, estado `paid/APPLIED` y filas únicas para internet.
5. **S5 — Límites:** solo internet sintético; no incluye gateway, dinero real, registros oficiales ni claims productivos.

## Tasks

- T1 [S1-S3] Generalizar la persistencia existente de utilities para incluir la factura internet sin cambiar los otros facturadores. Ruta: utilities. Commit: `f113050`.
- T2 [S4-S5] Extender aceptación Docker, documentación y límites para internet. Ruta: infra/docs. Commit: `f113050`.
- T3 [S1-S5] Ejecutar checks, crear work unit y evaluar RDD. Ruta: Docker + Git. Commit: pendiente (review RDD pendiente).

## Log

- L1. El usuario pidió que internet también estuviera dentro del alcance durable: "internet también debe estar dentro".
- L2. El slice reutiliza `utilities-db`, el wrapper de recreación de `utilities` y los contratos legacy ya existentes.
- L3. T1-T2 implementó el seed y lookup durable de `bill-internet-2025-001`, manteniendo el orden del catálogo, las rutas legacy, estados, idempotencia, HMAC y comportamiento sintético de los otros tres servicios.
- L4. La aceptación Docker ahora crea/repite intent, aplica/repite evento firmado y comprueba `paid/APPLIED`, identidad post-recreación y unicidad durable para internet; no agrega gateway, ledger, settlement ni dinero real.

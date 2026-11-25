# Phase 6 — Recuperación durable de mobile sintético
Fuente: `spec.md` y `odd/tasks/phase-6-billers.md`.
Alcance: extender el PostgreSQL aislado de utilities para la factura sintética móvil; electricidad y agua quedan compatibles, internet permanece legacy.

## Specs

1. **S1 — Mobile durable:** `bill-mobile-2025-001`, su payment intent y evento aplicado sobreviven al reinicio de `utilities` mediante `utilities-db`.
2. **S2 — Contrato compatible:** conservar rutas legacy `/api/v1`, estados, idempotency-key, firma HMAC y respuestas existentes.
3. **S3 — Catálogo:** electricidad, agua, internet y mobile siguen visibles como servicios sintéticos; internet no se vuelve durable incidentalmente.
4. **S4 — Aceptación:** crear/repetir intent, aplicar/repetir evento, recrear solo utilities y verificar identidad, estado `paid/APPLIED` y filas únicas para mobile.
5. **S5 — Límites:** solo mobile sintético; no incluye internet durable, gateway, dinero real, registros oficiales ni claims productivos.

## Tasks

- T1 [S1-S3] Generalizar la persistencia existente de utilities para incluir la factura mobile sin cambiar electricidad, agua ni internet legacy. Ruta: utilities. Commit: `20187e2`.
- T2 [S4-S5] Extender aceptación Docker, documentación y límites para mobile. Ruta: infra/docs. Commit: `20187e2`.
- T3 [S1-S5] Ejecutar checks, crear work unit y evaluar RDD. Ruta: Docker + Git. Commit: pendiente (review RDD pendiente).

## Log

- L1. El usuario pidió continuar después del cierre del slice de agua P6.
- L2. La exploración recomendó mobile antes que internet porque su suite ya cubre intent, replay, evento HMAC y estado final.
- L3. T1-T2 extendió `DurableStore` para sembrar y consultar `bill-mobile-2025-001`, manteniendo internet legacy/en memoria y las rutas/estados/HMAC/idempotencia existentes.
- L4. La aceptación `p6` ahora verifica catálogo de cuatro servicios, intent mobile por ruta nueva, replay, evento firmado/replay, `paid/APPLIED`, identidad tras recreación de `utilities` y unicidad durable de intents/eventos para mobile y water; no afirma gateway, ledger, settlement ni dinero real.

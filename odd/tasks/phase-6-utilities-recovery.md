# Phase 6 — Recuperación durable de utilities sintético
Fuente: `spec.md`, secciones 6, 7, 13 y 27.
Alcance: persistir y recuperar únicamente el flujo sintético de factura eléctrica de `utilities`; no ampliar settlement ni facturadores productivos.

## Specs

1. **S1 — Autoridad durable:** Factura eléctrica, payment intent sintético y evento de aplicación deben sobrevivir al reinicio del contenedor `utilities` mediante un PostgreSQL propio y aislado.
2. **S2 — Idempotencia:** Crear/repetir el intent con la misma clave y reenviar el mismo evento firmado deben conservar identidad y producir un solo efecto durable.
3. **S3 — Recuperación Docker:** La aceptación debe detener/recrear solo `utilities`, conservar su volumen PostgreSQL y comprobar factura, intent, evento y estado aplicado antes/después.
4. **S4 — Aislamiento:** `utilities-db` no puede conectarse a ledgers bancarios ni reemplazar la autoridad PostgreSQL de `bank-core`; el flujo es sintético y Docker-only.
5. **S5 — Límites:** No incluir transit, agua, internet o móvil; no afirmar recuperación del gateway, de todas las bases, settlement global, dinero real ni garantías productivas.

## Tasks

- T1 [S1-S2] Persistir catálogo eléctrico, intents y eventos en PostgreSQL con compatibilidad de API. Ruta: utilities. Commit: `25eae15`.
- T2 [S3-S5] Añadir utilities-db aislado, aceptación Docker y runbook de reinicio/replay. Ruta: Compose/infra/docs. Commit: `25eae15`.
- T3 [S1-S5] Ejecutar checks, crear work unit y evaluar RDD. Ruta: Docker + Git. Commit: pendiente (review RDD pendiente).

## Log

- L1. El usuario eligió P6 como siguiente área después de cerrar P5.
- L2. La exploración confirmó que `utilities` mantiene facturas, intents y eventos en memoria; se eligió persistir únicamente electricidad como primer slice de recuperación.
- L3. T1-T2 implementa PostgreSQL aislado para electricidad, intent/idempotencia y eventos HMAC, con `utilities-db`, volumen, healthcheck, profile `p6`, aceptación Docker y runbook. T3 (checks, work unit y RDD) queda pendiente.

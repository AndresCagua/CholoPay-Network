# Phase 6 — Recuperación durable de utilities agua sintética
Fuente: `spec.md` y `odd/tasks/phase-6-billers.md`.
Alcance: extender el PostgreSQL aislado de utilities para la factura sintética de agua; electricidad queda compatible y fuera de cambios semánticos.

## Specs

1. **S1 — Agua durable:** `invoice-water-2025-001`, su payment intent y evento aplicado sobreviven al reinicio de `utilities` mediante `utilities-db`.
2. **S2 — Contrato compatible:** conservar rutas `/v1/bills/...`, estados, idempotency-key, firma HMAC y respuestas existentes.
3. **S3 — Aislamiento:** agua y electricidad permanecen en el PostgreSQL propio de utilities; no se conecta a bancos, ledgers ni settlement.
4. **S4 — Aceptación:** crear/repetir intent, aplicar/repetir evento, recrear solo utilities y verificar identidad, estado `paid/APPLIED` y filas únicas para agua.
5. **S5 — Límites:** solo agua sintética; no incluye internet, móvil, gateway, dinero real, registros oficiales ni claims productivos.

## Tasks

- T1 [S1-S3] Generalizar la persistencia existente de utilities para incluir la factura de agua sin cambiar electricidad. Ruta: utilities. Commit: `3b20ca5`.
- T2 [S4-S5] Extender aceptación Docker, documentación y límites para agua. Ruta: infra/docs. Commit: `3b20ca5`.
- T3 [S1-S5] Ejecutar checks, crear work unit y evaluar RDD. Ruta: Docker + Git. Commit: pendiente (review RDD pendiente).

## Log

- L1. El usuario pidió continuar después del cierre de transit P6.
- L2. La exploración identificó agua como el siguiente slice mínimo: comparte el esquema aislado de utilities, pero actualmente solo electricidad se persiste.
- L3. T1-T2 implementados para `Spec: odd/tasks/phase-6-water-recovery.md, T1-T2, S1-S5`: `DurableStore` siembra, lista y consulta agua junto con electricidad; intents y eventos mantienen las mismas rutas, idempotencia, HMAC y estados. El probe Docker verifica create/replay/apply/replay, `paid/APPLIED`, una fila durable por intent/evento, recreación exclusiva de `utilities` y presencia eléctrica. Se mantienen los límites sintético, aislado y sin conexiones bancarias/ledger/settlement; T3 (checks, work unit y RDD) queda pendiente.
- L4. La review `review-d89d712288347365` detectó que el listing/lookup durable ocultaba invoices legacy de internet y móvil. Se corrigió preservando esos invoices en memoria mientras electricidad y agua permanecen durables; `job-42` pasó suite y aceptación Docker.

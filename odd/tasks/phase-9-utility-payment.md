# Phase 9 — Vertical slice de pago de utility
Fuente: `spec.md`, secciones 23.3, 24.1, 24.3, 24.4, 26 y 27.
Alcance: flujo sintético de una factura de electricidad desde el intent de pago hasta la aplicación idempotente del facturador; sin débito bancario real, escritura directa de ledger ni infraestructura externa.

## Specs

1. **S1 — CholoServicios**
   > Una **sola empresa multiservicios simulada** provee electricidad, agua potable, internet fijo y telefonía móvil, con un único portal web y backend, pero módulos de negocio separados.

2. **S2 — APIs mínimas**
   > `POST /v1/bills/{billId}/payment-intents`
   > `GET /v1/bills/{billId}/payment-status`
   > `POST /v1/internal/payment-events` (solo gateway/red, firma + idempotencia)

3. **S3 — Estados y consistencia**
   > `ISSUED -> PAYMENT_PENDING -> PAID`
   > No descontar una deuda antes de la confirmación final; no permitir que dos pagos concurrentes liquiden dos veces la misma obligación.
   > Si el pago finaliza pero el facturador está caído, registrar `SETTLED_AWAITING_APPLICATION` y reintentar la aplicación sin perder ni duplicar fondos.

4. **S4 — Fronteras**
   > No debitar cuentas de clientes o liquidar sin la red.
   > Cada facturador conserva la autoridad sobre su obligación y cada banco sobre su ledger.

5. **S5 — Aceptación vertical**
   > Implementar **primero** un recorrido vertical completo (Banco Costa paga una factura de luz emitida por CholoServicios) antes de generalizar a los seis tipos de obligaciones.

## Tareas

- T1 (S1-S4): Extender utilities con estado de obligación, payment intent idempotente y evento interno autenticado sintético. Ruta: inline después de una exploración acotada. Commit: pendiente.
- T2 (S3-S5): Probar replay, concurrencia y aplicación eventual sin débito ni ledger local. Ruta: inline. Commit: pendiente.
- T3 (S1-S5): Actualizar contrato, README y evidencia Docker. Ruta: inline. Commit: pendiente.

## Log

L1. Usuario: "listo continua"
L2. Se prioriza el vertical slice de electricidad indicado explícitamente por `spec.md` antes de ampliar a agua, internet, móvil, tránsito o portales completos.
L3. Utilities ahora expone estado de pago y `POST /api/v1/internal/payment-events`; valida firma HMAC sintética, aplica eventos `payment.settled` una sola vez, rechaza conflictos de event ID y mantiene `debit_performed=false` y `ledger_entry_created=false`.
L4. Evidencia Docker: Compose config PASS; 6/6 tests PASS; replay del evento de liquidación devuelve el mismo recibo y la factura queda `paid` sin débito ni escritura de ledger. `py_compile` y `git diff --check` PASS.

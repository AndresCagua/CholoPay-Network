# Phase 13 — Expansión vertical de pagos y superficies públicas
Fuente: `spec.md`, secciones 8.1, 8.3, 8.4, 24.4 y 26.
Scope: completar progresivamente un pago sintético de factura, operaciones web participant-scoped y capacidades faltantes del gateway/comercio. Todo permanece Docker-only, sintético y sin ledger escrito por gateway/facturador.

## Specs

1. **S1 — Recorrido vertical prioritario**
   > "Completar el recorrido vertical de pagos de facturas"
   > "Banco → CholoNet/Gateway → CholoServicios → aplicación idempotente."

2. **S2 — Paneles web**
   > "Convertir los paneles web scaffold en operaciones reales"
   > "Transferencias, movimientos y lectura participant-scoped desde APIs."

3. **S3 — Gateway/comercio**
   > "Completar gateway/comercio"
   > "Reembolsos sintéticos, integración comercio-gateway y contrato de checkout."

4. **S4 — Orden autorizado**
   > "ya dale con 1 luego 2 y luego 3"

## Tasks

- T1 (S1): Implementar el recorrido sintético Banco Costa → CholoNet/Gateway → CholoServicios para una factura de electricidad, con idempotencia, aplicación eventual y evidencia pública. Route: delegated bounded writer. Commit: `43c805a`.
- T2 (S2): Convertir los paneles web scaffold en una lectura/operación participant-scoped contra APIs, sin acceso directo a bases. Route: worker. Commit: `3d33ab9`.
- T3 (S3): Completar reembolsos sintéticos e integración mínima comercio-gateway con contrato de checkout. Route: worker. Commit: `cbcacb2`.
- T4 (S1-S3): Ejecutar verificación Docker, actualizar contratos/runbooks y registrar límites. Route: inline. Commit: pending.

## Log

L1. Usuario: "ya dale con 1 luego 2 y luego 3"
L2. La prioridad es completar primero el recorrido vertical de pago de factura, después operaciones web participant-scoped y finalmente reembolsos/integración commerce-gateway.
L3. T1 implementó `POST /v1/bill-payments`, consulta y aplicación/reintento para `BANCO_COSTA` sobre la factura sintética de electricidad. Gateway crea el intent en CholoServicios y aplica un evento `payment.settled` firmado e idempotente; gateway y utilities no escriben ledgers. La frontera bancaria es explícita mediante `participant_id`; el débito del ledger bancario real queda fuera de este slice y no se inventa.
L4. Evidencia Docker: gateway test image PASS; utilities test image PASS (7 tests); `docker compose -f infra/compose.yaml config` PASS sin `ports`; smoke integrado PASS con respuesta `201`, estado `APPLIED`, factura `invoice-elec-2025-001` y participante `BANCO_COSTA`. El primer check falló porque el worker añadió un puerto host al Compose; se corrigió a `expose` antes de la verificación final.
L5. T2 (S2) implementado: los cuatro paneles consultan un cliente de lectura participant-scoped mediante `GET /v1/participants/{participant_id}/transfers`, con base URL configurable, estados explícitos de carga/error/vacío y adaptadores sintéticos read-only para ejercer la frontera pública mientras no exista la proyección backend. No se añadió acceso de navegador a bases de datos, ledger o filesystem; no se implementaron reembolsos ni comercio. Builds Docker de operator, Costa, Sierra y Oriente PASS.
L6. T3 (S3) implementado: gateway añade reembolso sintético único, full-amount, ligado a intent `succeeded`, con API-key scoped e idempotencia/conflictos; comercio añade checkout HTTP configurable o adaptador sintético explícito, estados `pending`/`succeeded` y confirmación server-side sin prueba por redirect. Compose de comercio queda sin puertos host por defecto. No se modificaron bank-core, network ni panel web. Gateway tests Docker PASS (12); commerce tests Docker PASS (5); Compose y `git diff --check` PASS.
L7. T4 (S1-S3) verificado: `docker compose -f infra/compose.yaml config`, `docker compose -f commerce/compose.yaml config`, `git diff --check`, build/test Docker del gateway y suite Docker de commerce PASS. El worktree quedó limpio; las métricas y contratos siguen siendo sintéticos y no representan integración financiera real.

# Phase 17 — Entrega interbancaria por participante

Fuente: `docs/spec-pending-work.md`, P2; `spec.md`, ADR 0008 y protocolo de transferencias.
Alcance: enrutar la entrega asíncrona de una liquidación confirmada hacia el banco receptor correcto, usando configuración Docker-only de participantes, URLs internas y secretos HMAC independientes. No implementa todavía las APIs de lectura completas de P3 ni una liquidación financiera real.

## Specs

1. **S1 — Solicitud del usuario:**
   > listo dale con el siguiente pendiente

2. **S2 — Entrega interbancaria:**
   > Banco Costa → CholoNet → Banco Sierra/Oriente → receiver inbox → ledger receptor → consulta de estado → replay, retry y reconciliación

3. **S3 — Settlement desacoplado:**
   > El settlement no realiza HTTP ni enqueue implícito dentro de su commit financiero.

4. **S4 — Transporte:**
   > Transporte at-least-once con efectos financieros idempotentes.

5. **S5 — Aislamiento:**
   > La red no accede a tablas internas bancarias.

6. **S6 — Límites:**
   > La liquidación productiva, fondeo real, redes financieras externas, mTLS productivo y compatibilidad normativa real permanecen fuera de alcance.

## Tasks

- T1 (S2-S5): añadir un resolver de destinos por `receiver_participant_id`, con URL y secreto HMAC independientes configurados por Compose. Ruta: inline. Commit: `c4ff0aa`.
- T2 (S2-S4): conservar retries, leases, `delivery_id` y `event_id`, y añadir pruebas de routing correcto, destino desconocido y replay idempotente. Ruta: inline. Commit: `c4ff0aa`.
- T3 (S2-S6): configurar destinos sintéticos Sierra/Oriente y documentar la frontera de aceptación; no añadir todavía APIs públicas de lectura de P3. Ruta: inline. Commit: `c4ff0aa`.
- T4 (S2-S6): ejecutar pruebas Docker de network/bank-core y registrar la evidencia y limitaciones. Ruta: inline. Commit: `c4ff0aa`.

## Log

- L1. Usuario: "listo dale con el siguiente pendiente"
- L2. P1 quedó completado en Phase 16; se selecciona P2 para eliminar el receptor único hardwired a Costa.
- L3. La exploración confirmó que CholoNet usa un único `BANK_RECEIVER_CREDIT_URL` y un único secreto, aunque Sierra y Oriente ya tienen secretos distintos. La red no posee aún un registro de URL/capacidad por participante.
- L4. La aceptación pública completa hasta balances y journals queda limitada por la ausencia de APIs bancarias de lectura, que pertenece a P3; este slice se concentra en routing y sus pruebas de entrega.
- L5. Se añadió `BANK_RECEIVER_ROUTES` con formato `participant_uuid=url|secret`, un `ReceiverTarget` por participante y selección del destino en `DeliveryDispatcher`. Los destinos desconocidos se registran como `RECEIVER_ROUTE_NOT_FOUND`; el fallback legacy conserva compatibilidad cuando no existe el mapa.
- L6. `job-4` ejecutó la suite completa de red dentro de Docker: `cmd/network`, `directory`, `instruction`, `message`, `orchestration`, `routing`, `settlement` y `transfer` terminaron en `ok`. `gofmt`, `git diff --check` y Compose config también pasaron.
- L7. Este slice completa el routing configurable de P2, pero no demuestra todavía una transferencia Costa→Sierra/Oriente de extremo a extremo ni lectura pública de balances/journals; esas evidencias permanecen pendientes y las APIs de lectura pertenecen a P3.
- L8. El work unit quedó en `c4ff0aa 20261118: enruta entregas por participante`. ASSESS sobre el rango `852565a..c4ff0aa` reportó riesgo medio, `reviewDue=false` y plan de auto-verificación; no se inició review nativa adicional porque no estaba debida y el usuario no la solicitó explícitamente.

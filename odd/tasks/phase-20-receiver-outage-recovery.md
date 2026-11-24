# Phase 20 — Recovery E2E ante caída y DLQ del receptor

Fuente: `docs/spec-pending-work.md`, P2/P7; Phase 18 y ADRs 0006, 0008 y 0009.
Alcance: demostrar dentro de Docker una caída real de un banco receptor durante una entrega de CholoNet, retry/backoff, agotamiento de intentos, DLQ observable, replay autenticado e idempotente, reconciliación y evidencia estable de settlement.

## Specs

1. **S1 — Solicitud del usuario:**
   > listo dale con el slice 1

2. **S2 — Alcance confirmado:**
   > Recuperación ante caída breve
   > Detener `bank-core-sierra` durante una entrega, observar retry/backoff real, reiniciarlo y verificar `APPLIED`, `COMPLETED`, `delivery_id/event_id` estables e inbox/journal únicos.

3. **S3 — Límites:** El slice no define replay de DLQ ni una nueva semántica de reconciliación para entregas `FAILED`; esas decisiones quedan fuera de alcance.

4. **S4 — Docker-only:** builds, servicios, migraciones, bases, parada/reinicio y aceptación se ejecutan dentro del entorno Docker/Compose.

5. **S5 — Solicitud del usuario:**
   > listo continua con el slice 2

6. **S6 — Contrato aprobado para el slice 2:** Una entrega que agota retries permanece como `state=FAILED` con `failure_code=DELIVERY_DLQ`; un replay autenticado por `delivery_id` la pasa a `PENDING`, conserva `delivery_id`, `event_id` y `attempt_count`, mantiene la transferencia en `CREDIT_PENDING` hasta la aplicación, y la reconciliación reporta `DELIVERY_DLQ` hasta que la entrega se aplique.

7. **S7 — Límites del slice 2:** Solo se recupera una entrega fallida individual de forma observable e idempotente. No se agregan un estado persistido `DLQ` separado, replay por lote, compensaciones financieras, cambios de ledger ni reconciliación de entregas `FAILED` con una semántica distinta de `DELIVERY_DLQ`.

8. **S8 — Solicitud del usuario:**
   > Slice 3: reordenamiento e idempotencia

9. **S9 — Alcance del slice 3:** Probar dos transferencias independientes entregadas en orden inverso, verificando que ambas se apliquen sin efectos duplicados, con una sola entrega/inbox/journal por transferencia y balances/journals finales correctos. No se afirma un orden global de eventos ni un secuenciamiento causal que el protocolo actual no persiste.

10. **S10 — Solicitud del slice 4:** Mantener el endpoint interno de reconciliación sin alcance y sus semánticas actuales, agregando un `transfer_id` opcional para que la ejecución acotada solo devuelva diferencias del transfer objetivo.

11. **S11 — Aceptación del slice 4:** Desde Docker, invocar la interfaz HTTP acotada y verificar para el target los totales autoritativos exactos del settlement CholoNet y los balances/journal del receiver Sierra. No se agregan fixtures de API pública, débitos de cuentas públicas de Costa, cleanup global, compensaciones ni semántica nueva de ledger.

## Tasks

- T1 [S1-S4] Crear runner/escenario de outage breve con stop/start externo de `bank-core-sierra` y observación de retry/backoff. Ruta: `infra/tests` + Compose. Commit: pendiente.
- T2 [S1-S4] Verificar estado final `COMPLETED`/`APPLIED`, identidad estable de delivery/event y unicidad de inbox/journal receptor. Ruta: `infra/tests` + consultas Docker. Commit: pendiente.
- T3 [S1-S4] Registrar evidencia, límites y resultado en documentación P2. Ruta: docs. Commit: pendiente.
- T4 [S1-S4] Ejecutar checks, crear work unit y evaluar RDD. Ruta: Docker + Git. Commit: pendiente.
- T5 [S5-S7] Agregar la transición autenticada e idempotente de replay individual `FAILED/DELIVERY_DLQ → PENDING`, conservando identidad y contador de intentos. Ruta: `network/internal/settlement` + migraciones nuevas si fueran estrictamente necesarias. Commit: pendiente.
- T6 [S5-S7] Extender reconciliación para observar `DELIVERY_DLQ` y aceptar su desaparición solo después de `APPLIED`/`COMPLETED`. Ruta: `network/internal/settlement` + pruebas. Commit: pendiente.
- T7 [S5-S7] Crear aceptación Docker-only de outage prolongada, DLQ, replay, recuperación, reconciliación limpia e inbox/journal únicos. Ruta: `infra/tests` + Compose. Commit: pendiente.
- T8 [S5-S7] Ejecutar checks, crear work unit y evaluar RDD del nuevo target. Ruta: Docker + Git. Commit: pendiente.
- T9 [S8-S9] Agregar pruebas de dos transferencias independientes con llegada inversa y duplicados/retries idempotentes, sin introducir una garantía de orden global. Ruta: `network/internal/settlement`, `bank-core` y pruebas. Commit: pendiente.
- T10 [S8-S9] Extender la aceptación Docker-only para entregar las dos transferencias en orden inverso y verificar balances, journals e inboxes únicos. Ruta: `infra/tests` + Compose. Commit: pendiente.
- T11 [S8-S9] Ejecutar checks, registrar evidencia, crear work unit y evaluar RDD del nuevo target. Ruta: Docker + Git. Commit: pendiente.
- T12 [S10] Agregar el filtro opcional `transfer_id` a la reconciliación interna, preservando la ejecución sin alcance. Ruta: `network/internal/settlement` + pruebas. Commit: pendiente.
- T13 [S10] Verificar mediante pruebas Go que la reconciliación acotada devuelve solo el target y la no acotada conserva todos los resultados. Ruta: `network/internal/settlement`. Commit: pendiente.
- T14 [S11] Extender la aceptación Docker-only para llamar la interfaz HTTP acotada y comprobar settlement, receiver inbox/journal y balances exactos de Sierra. Ruta: `infra/tests`. Commit: pendiente.
- T15 [S10-S11] Ejecutar checks, registrar evidencia, actualizar documentación de alcance y evaluar RDD del nuevo target. Ruta: Docker + Git. Commit: pendiente.

## Log

- L1. Usuario: "listo dale con el slice 1"
- L2. La exploración de Phase 18 confirmó que existen dispatcher, retry/backoff, leases, receiver inbox, aplicación idempotente y endpoints de settlement, pero nunca se detuvo/reinició un receptor real dentro de la aceptación E2E.
- L3. Se eligió el slice de recuperación ante caída breve. DLQ prolongada y reconciliación de entregas `FAILED` quedan fuera porque el repositorio no define todavía una transición de recuperación/replay para ese estado.
- L4. La exploración confirmó que `DELIVERY_DLQ` ya se persiste como `failure_code` junto con `state=FAILED`; no existe un estado `DLQ` separado ni una operación de replay.
- L5. El usuario autorizó continuar con el slice 2: "listo continua con el slice 2".
- L6. Se aprobó la semántica conservadora: replay autenticado por `delivery_id`, identidad `delivery_id/event_id` estable, `attempt_count` acumulado, transferencia `CREDIT_PENDING` hasta aplicar y diferencia de reconciliación `DELIVERY_DLQ` mientras la entrega siga fallida.
- L7. La aceptación Docker-only `job-4` pasó con outage prolongada: retries agotados en `attempt_count=5`, replay idempotente conservando `delivery_id=f9a2d69d-4bd8-4582-89c6-f7a6fc95b901`, `event_id=97959977-b68a-4034-a803-9b6a3ba5afe0`, recuperación `APPLIED`/`COMPLETED`, inbox=1, journal=1 y balance `125/125`; la reconciliación final del transfer `bcab076e-cadc-45c4-9b13-bb9f598dca1d` quedó limpia.
- L8. Para resolver el finding `R3-replay-default-token`, el replay ahora falla cerrado si falta `OPERATOR_CONSOLE_TOKEN`, exige exactamente `Authorization: Bearer <token>` y Compose exige configurar el token sin fallback conocido. Las pruebas HTTP de settlement y `docker compose config` pasaron.
- L9. El usuario eligió continuar con "Slice 3: reordenamiento e idempotencia". La exploración confirmó que el sistema ya prueba duplicados e idempotencia, pero no tiene secuencias ni causalidad persistidas; por eso el slice probará orden inverso entre transferencias independientes y no una garantía global.
- L10. La aceptación Docker-only `job-2` pasó: dos transferencias independientes se liquidaron en orden inverso, se recuperaron tras reiniciar Sierra, produjeron 2 deliveries, 2 inboxes y 2 journals únicos, con totales balanceados `325/325` y balances observados de clearing `998225` y destino `1775`.
- L11. Se aprueba el slice 4 acotado: reconciliación interna opcionalmente filtrada por `transfer_id`, con ejecución sin alcance intacta y evidencia Docker de settlement CholoNet y receiver Sierra exactos para el target. Quedan fuera APIs públicas, débitos públicos Costa, cleanup global, compensaciones y semántica nueva de ledger.

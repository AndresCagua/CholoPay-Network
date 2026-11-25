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

12. **S12 — Solicitud aprobada para P7:** Definir un adapter Kafka mínimo para un evento interbancario versionado, con PostgreSQL como fuente de verdad y un consumer de prueba que entregue al boundary PostgreSQL.

13. **S13 — Contrato mínimo de P7:** Publicar el envelope persistido del outbox en `cholonet.inter-network.v1`, usando `event_id` como key estable, consumer group `cholonet-network-boundary` y commit del offset solo después de aceptación durable por PostgreSQL.

14. **S14 — Límites de P7:** Demostrar solo recuperación at-least-once ante una caída/reinicio de Kafka y deduplicación del boundary. No afirmar exactly-once, orden global, DLQ Kafka, backup/restore, RPO/RTO ni disponibilidad productiva.

15. **S15 — Siguiente slice aprobado:** Demostrar backup/restore lógico de un solo PostgreSQL sintético, usando `network-db` como fuente durable y preservando sus invariantes observables.

16. **S16 — Contrato del slice de backup/restore:** Crear un registro sintético identificable, capturar su baseline, ejecutar `pg_dump`, mutar/eliminar solo el registro objetivo, restaurar con `psql` y verificar identidad, payload hash, idempotencia y disponibilidad posterior.

17. **S17 — Límites del slice de backup/restore:** No afirmar consistencia entre las cuatro bases, RPO/RTO, PITR/WAL, cifrado, retención, rotación, backup de Kafka ni garantías productivas.

18. **S18 — P8.1 aprobado:** Agregar health/readiness operacional local: endpoints `live`/`ready` uniformes, healthchecks HTTP Compose y aceptación Docker de estado sano, caída de una base sintética y recuperación. No incluye métricas, dashboards ni tracing.

19. **S19 — Contrato P8.1:** `live` demuestra proceso activo; `ready` demuestra dependencias PostgreSQL disponibles y devuelve `503` cuando la dependencia está caída. Los datos son sintéticos y el flujo permanece privado dentro de Docker.

20. **S20 — Límites P8.1:** No afirmar OpenTelemetry, Prometheus, Grafana, alertas, SLO/SLI, tracing distribuido, retención histórica, disponibilidad productiva ni métricas estables de negocio.

21. **S21 — P8.2 aprobado:** Agregar un stack opcional de Prometheus/Grafana para observabilidad local reproducible, activado únicamente mediante un profile explícito de Compose y sin puertos publicados por defecto.

22. **S22 — Métricas P8.2:** Exponer métricas Prometheus mínimas de proceso/HTTP y una métrica sintética de aceptación en los servicios instrumentados, sin convertirlas en contrato de negocio, settlement, ledger, outbox o DLQ.

23. **S23 — Evidencia P8.2:** La aceptación Docker debe comprobar que Prometheus descubre y scrapea los targets configurados, que Grafana carga el datasource/dashboard provisionado y que la configuración por defecto no publica puertos.

24. **S24 — Límites P8.2:** No afirmar SLO/SLI, alertas productivas, retención durable, capacidad, tracing distribuido, seguridad productiva, alta disponibilidad ni monitoreo remoto. La retención es efímera y el stack es educativo/sintético.

25. **S25 — P2 histórico aprobado:** Verificar que una reconciliación con diferencia `DELIVERY_DLQ` permanezca consultable e inmutable después de recuperar el mismo delivery y ejecutar una reconciliación posterior limpia.

26. **S26 — Evidencia histórica P2:** Conservar y comparar `transfer_id`, `delivery_id`, `event_id`, conteos de journal/inbox/delivery y resultados de runs antes y después de recovery; PostgreSQL sigue siendo la fuente de verdad.

27. **S27 — Límites históricos P2:** No implementar ni inferir orden global, causalidad entre transferencias independientes, sequence de stream, ordering por timestamp/Kafka offset ni evidencia histórica de ledger receptor no exportada por el banco.

28. **S28 — P2 public receiver evidence:** La aceptación debe consultar por HTTP público el balance y journals de la cuenta receptora Sierra usando una identidad sintética autorizada y correlacionar el journal con `transfer_id`.

29. **S29 — Alcance de la evidencia pública:** La consulta debe verificar un único `CREDIT` receptor por `125`, balance receptor y repetición estable; el `DEBIT` de la cuenta compensadora permanece fuera de esa vista por cuenta.

30. **S30 — Límites:** No se afirma UI completa, ordering global, exactly-once, causalidad, dinero real ni exportación del ledger receptor completo.

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
- T12 [S12-S14] Definir el contrato mínimo del adapter Kafka: evento versionado, topic, key, consumer y confirmación posterior a persistencia PostgreSQL. Ruta: ADR + `network`/`infra`. Commit: pendiente.
- T13 [S12-S14] Implementar producer/consumer de prueba sobre el outbox existente, con deduplicación en el boundary PostgreSQL y configuración solo bajo profile `events`. Ruta: `network`, `bank-core` y Compose. Commit: pendiente.
- T14 [S12-S14] Crear aceptación Docker de un evento durante caída/reinicio de Kafka y verificar identidad estable, persistencia durable y ausencia de doble efecto financiero. Ruta: `infra/tests` + Compose. Commit: pendiente.
- T15 [S12-S14] Ejecutar checks, documentar límites, crear work unit y evaluar RDD del nuevo target. Ruta: Docker + Git. Commit: pendiente.
- T16 [S10] Agregar el filtro opcional `transfer_id` a la reconciliación interna, preservando la ejecución sin alcance. Ruta: `network/internal/settlement` + pruebas. Commit: pendiente.
- T17 [S10] Verificar mediante pruebas Go que la reconciliación acotada devuelve solo el target y la no acotada conserva todos los resultados. Ruta: `network/internal/settlement`. Commit: pendiente.
- T18 [S11] Extender la aceptación Docker-only para llamar la interfaz HTTP acotada y comprobar settlement, receiver inbox/journal y balances exactos de Sierra. Ruta: `infra/tests`. Commit: pendiente.
- T19 [S10-S11] Ejecutar checks, registrar evidencia, actualizar documentación de alcance y evaluar RDD del nuevo target. Ruta: Docker + Git. Commit: pendiente.
- T20 [S15-S17] Crear runner Docker-only para generar baseline, backup lógico, mutación controlada y restore de un registro sintético en `network-db`. Ruta: `infra/tests` + `infra/scripts/backup-restore.sh`. Commit: pendiente.
- T21 [S15-S17] Verificar identidad, hash, idempotencia, constraints y readiness después del restore; sin afirmar consistencia cross-database o RPO/RTO. Ruta: `infra/tests` + SQL/HTTP. Commit: pendiente.
- T22 [S15-S17] Documentar evidencia y límites del slice en P7/runbook. Ruta: docs. Commit: pendiente.
- T23 [S15-S17] Ejecutar checks, crear work unit y evaluar RDD del nuevo target. Ruta: Docker + Git. Commit: pendiente.
- T24 [S18-S20] Añadir endpoints live/ready y healthchecks HTTP para los servicios incluidos, preservando respuestas compatibles. Ruta: `network`, `bank-core`, `gateway`, `transit`, `utilities`, Compose. Commit: pendiente.
- T25 [S18-S20] Crear aceptación Docker de live/ready sano, caída de una base sintética y recuperación, sin puertos publicados. Ruta: `infra/tests` + scripts. Commit: pendiente.
- T26 [S18-S20] Documentar comandos, interpretación y límites de P8.1. Ruta: docs. Commit: pendiente.
- T27 [S18-S20] Ejecutar checks, crear work unit y evaluar RDD del nuevo target. Ruta: Docker + Git. Commit: pendiente.
- T28 [S21-S24] Añadir métricas Prometheus mínimas y composición opcional Prometheus/Grafana con provisioning reproducible. Ruta: servicios, `infra/compose.yaml`, `infra/observability`. Commit: pendiente.
- T29 [S21-S24] Crear aceptación Docker de scrape, targets, datasource/dashboard y configuración privada por defecto. Ruta: `infra/tests`. Commit: pendiente.
- T30 [S21-S24] Documentar ejecución, consultas y límites educativos/efímeros del stack. Ruta: docs. Commit: pendiente.
- T31 [S21-S24] Ejecutar checks, crear work unit y evaluar RDD del nuevo target. Ruta: Docker + Git. Commit: pendiente.
- T32 [S25-S27] Ampliar aceptación de outage/replay para conservar y verificar runs históricos de reconciliación. Ruta: `infra/tests/phase20_receiver_outage.py`. Commit: pendiente.
- T33 [S25-S27] Añadir tests de persistencia histórica de reconciliación y documentación de límites. Ruta: `network/internal/settlement`, docs. Commit: pendiente.
- T34 [S25-S27] Ejecutar checks, crear work unit y evaluar RDD del nuevo target. Ruta: Docker + Git. Commit: pendiente.
- T35 [S28-S30] Crear fixture público sintético para asociar la cuenta receptora Sierra con un customer/token sin tocar migraciones aplicadas. Ruta: `bank-core/src/main/resources/db/migration`. Commit: pendiente.
- T36 [S28-S30] Extender la aceptación Phase 20 para consultar balance/journals públicos, correlacionar `transaction_id` y verificar crédito/repetición estable. Ruta: `infra/tests` + Compose. Commit: pendiente.
- T37 [S28-S30] Ejecutar checks, documentar evidencia y límites P2 y evaluar RDD del nuevo target. Ruta: Docker + docs + Git. Commit: pendiente.

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
- L11. El usuario indicó continuar con el siguiente slice. La exploración P7 confirmó que Kafka era solo un placeholder y que el outbox durable existente usa HTTP/HMAC y PostgreSQL.
- L12. Se aprobó definir un adapter Kafka mínimo para un evento interbancario, con PostgreSQL como fuente de verdad y un consumer de prueba que entregue al boundary PostgreSQL. Quedan fuera exactly-once, DLQ Kafka, orden global, backup/restore y RPO/RTO.
- L13. Se aprueba el slice 4 acotado: reconciliación interna opcionalmente filtrada por `transfer_id`, con ejecución sin alcance intacta y evidencia Docker de settlement CholoNet y receiver Sierra exactos para el target. Quedan fuera APIs públicas, débitos públicos Costa, cleanup global, compensaciones y semántica nueva de ledger.
- L14. La aceptación P7 Kafka pasó en Docker: identidad estable `event_id`/key, recuperación tras reinicio, replay at-least-once y una sola fila durable en el boundary PostgreSQL.
- L15. Se aprueba el siguiente slice: backup/restore lógico de un solo `network-db` sintético, con baseline y restore verificable. Quedan fuera consistencia entre bases, RPO/RTO, PITR/WAL, cifrado, retención, rotación, backup de Kafka y garantías productivas.
- L16. La aceptación Docker `phase20_backup_restore.sh` pasó: creó un registro sintético, obtuvo backup `network-db`, eliminó el objetivo, restauró con `psql` y verificó identidad, payload hash, idempotencia, constraint y readiness.
- L17. El usuario aprobó P8.1 health/readiness operacional local: endpoints live/ready uniformes, healthchecks HTTP Compose y aceptación Docker de caída/recuperación de una base sintética. Quedan fuera métricas, dashboards, tracing y SLO/SLI.
- L18. El usuario eligió P8.2 stack Prometheus/Grafana. El alcance aprobado es opcional bajo profile explícito, sin puertos publicados por defecto, con métricas y dashboard reproducibles y sin afirmaciones productivas.
- L19. P8.1 implementó T24-T27: live/ready en network, bank-core, gateway, transit y utilities; Compose verifica HTTP sin publicar puertos; `p8_health_acceptance.sh` detiene externamente `network-db`, observa `network ready=503` con live activo y confirma `ready=200` al recuperar. Ver `docs/runbooks/p8-observability.md`; no implica monitoreo ni disponibilidad productiva.
- L20. P8.2 implementó T28-T30: `/metrics` bounded en network y bank-core, profile Compose explícito `observability`, Prometheus/Grafana privados sin puertos publicados, almacenamiento efímero, scrape config y provisioning reproducibles. `p8_observability_acceptance.sh` verificó targets up, métrica scrapeada, datasource/dashboard y emitió PASS. No implica métricas de negocio, SLO/SLI, alertas, tracing, HA, retención durable, seguridad productiva ni monitoreo remoto.
- L21. El usuario eligió P2 como siguiente brecha; la exploración determinó que el primer slice verificable debe ser reconciliación histórica append-only, no causalidad de stream, porque todavía no existe un contrato de sequence/order para transferencias.
- L22. T32-T33 amplían la aceptación Docker para ejecutar una reconciliación `DELIVERY_DLQ`, conservar su `run_id`/`difference_id`, recuperar el mismo delivery, ejecutar una reconciliación posterior sin alcance limpia y consultar nuevamente el primer run; el test de settlement verifica que la diferencia histórica conserva identidad y detalle. `CHOLONET_HMAC_SECRET=local-test-hmac SIERRA_CHOLONET_HMAC_SECRET=sierra-test-hmac ORIENTE_CHOLONET_HMAC_SECRET=oriente-test-hmac sh infra/tests/phase20_receiver_outage.sh` pasó: `DELIVERY_DLQ`, replay idempotente, `APPLIED`/`COMPLETED`, delivery/inbox/journal `1/1/1`, `125/125`, run histórico consultable sin mutación y reverse-arrival `2/2/2` con `325/325`. El límite sigue siendo S25-S27: no se implementan garantías de ordering, sequence o exportación adicional de ledger receptor.
- L23. El usuario aprobó continuar con P2 agregando evidencia end-to-end por la frontera HTTP pública del banco receptor. Se creó `V18__public_receiver_fixture.sql` para asociar la cuenta receptora Sierra a un customer/token sintético sin tocar migraciones aplicadas.
- L24. El runner Phase 20 consulta `GET /v1/accounts/{account_id}/balance` y `GET /v1/accounts/{account_id}/journals` en Sierra, verifica `transaction_id=transfer_id`, un único `CREDIT` receptor por `125`, balance receptor y repetición estable. La aceptación `job-102` pasó y mantuvo las verificaciones internas de settlement `125/125`, replay, reconciliación y reverse-arrival.
- L25. El límite es deliberado: la API pública por cuenta expone el crédito de la cuenta receptora, no el débito de la cuenta compensadora; no se afirma UI completa, ordering global, exactly-once ni dinero real.

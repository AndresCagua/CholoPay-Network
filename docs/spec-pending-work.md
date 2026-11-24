# Trabajo pendiente de `spec.md`

Este documento registra los diez bloques que todavía no están completamente implementados o demostrados frente a `spec.md`. Se resolverán uno por uno mediante slices revisables, pruebas Docker-only y una review RDD independiente por target.

## Estado y límites

- El proyecto sigue siendo educativo, sintético y exclusivo de Docker.
- No se procesará dinero real ni se conectarán redes financieras externas.
- Una fase o componente parcialmente implementado no se presenta como cumplimiento total del requisito.
- La evidencia de cada punto debe incluir código, pruebas, documentación y las limitaciones que permanezcan.
- La review RDD de Phase 15 fue aprobada para su target; no certifica estos diez puntos.

## Puntos pendientes

### P1. Tres bancos realmente operativos

**Estado:** Completado en el alcance P1 — Phase 16 (`odd/tasks/phase-16-three-independent-banks.md`). La aceptación específica de tres instancias pasó en `job-2`, la suite completa de `bank-core` pasó en `job-3` y el commit es `1b42e9d`. ASSESS clasificó el candidato como riesgo medio pero `reviewDue=false`; no se inició review nativa adicional según la política configurada.

Existen bases PostgreSQL, frontends y configuración para Costa, Sierra y Oriente, pero falta demostrar tres instancias autónomas de `bank-core` ejecutándose simultáneamente con sus propias bases, identidades criptográficas, cuentas, secretos, configuración y ledgers.

**Evidencia pendiente:** Compose aislado, fixtures por banco, aislamiento entre bases y pruebas públicas contra las tres instancias.

### P2. Transferencia interbancaria completa

**Estado:** Parcial completado en Phase 18 y Phase 20 slice 1 (`odd/tasks/phase-18-interbank-e2e-acceptance.md`, `odd/tasks/phase-20-receiver-outage-recovery.md`). La aceptación Costa→Sierra/Oriente, entrega, replay e idempotencia pasaron; Phase 20 slice 1 demostró caída breve real del receptor, retry/backoff y recuperación. El slice 2 queda definido para cerrar la recuperación tras agotamiento de retries.

Falta cerrar y demostrar el recorrido completo `Banco Costa → CholoNet → Banco Sierra/Oriente → receiver inbox → ledger receptor → consulta desde UI` bajo una outage prolongada, con DLQ observable, replay individual autenticado y reconciliación posterior.

**Evidencia:** Phase 20 slice 2 (`job-4`) demostró outage prolongada, agotamiento en 5 intentos, `FAILED/DELIVERY_DLQ`, replay autenticado e idempotente conservando `delivery_id/event_id`, recuperación `APPLIED`/`COMPLETED`, inbox/journal únicos, balance `125/125` y reconciliación limpia para la transferencia objetivo.

**Evidencia:** Phase 20 slice 3 (`job-2`) demostró dos transferencias independientes liquidadas en orden inverso, recuperación tras reinicio, 2 deliveries, 2 receiver inboxes, 2 journals únicos y totales balanceados `325/325`; los balances observados fueron clearing `998225` y destino `1775`.

**Evidencia pendiente:** reordenamiento causal dentro de un mismo stream, si el protocolo futuro lo define, saldos y journals completos de emisor y receptor, además de una reconciliación global sin diferencias históricas preexistentes. Phase 20 slice 4 agrega una ejecución interna opcionalmente acotada por `transfer_id`; no cambia la semántica del endpoint sin alcance y la aceptación Docker verifica settlement CholoNet y receiver Sierra exactos para el target.

### P3. APIs bancarias completas

**Estado:** Parcial — superficies T1–T4c demostradas en alcance sintético.

El contrato bancario versionado cubre customers/onboarding, cuentas, balances, movimientos, journals, payment intents públicos, refunds, participantes, CholoKey, QR, paginación cursor y Problem Details para las rutas nuevas. P3 sigue parcial porque no representa autenticación productiva ni dinero real.

**Evidencia:** OpenAPI bank/network actualizado y validador PyYAML Docker-only en el perfil `test`; `job-34` confirma OpenAPI 3.1.0, rutas T3/T4b, lifecycle interno, seguridad, respuestas y referencias de schemas. `job-33` confirma la suite de red después de exigir `DIRECTORY_PORT_TOKEN` en resolución interna.

### P4. CholoKey y QR

**Estado:** Parcial — T4a–T4c completan el lifecycle sintético y el contrato de fachada/QR.

Existe creación pública participant/customer-scoped, bloqueo, suspensión, baja, límites por cuenta, historial append-only, alias sintéticos y QR estático/dinámico; no se ofrece una ruta pública de enumeración. La autenticación sigue siendo bearer sintético y el alcance no es productivo.

**Evidencia:** suites T4b, `network-test` (`job-33`) y validador OpenAPI Docker-only (`job-34`). La resolución interna exige bearer sintético y no existe una ruta pública de enumeración.

### P5. Seguridad de usuario y servicio

**Estado:** Parcial.

Existen mecanismos sintéticos como HMAC, bearer y API key, pero falta cubrir OIDC/OAuth 2.1, PKCE, MFA/step-up, JWT completo, mTLS operativo, TLS, rotación y hash de secretos, rate limiting, CORS/CSRF y autorización exhaustiva por cuenta y participante.

**Límite:** Las implementaciones seguirán siendo educativas y no se presentarán como certificación de cumplimiento normativo.

### P6. Facturadores persistentemente respaldados

**Estado:** Parcial.

Utilities y tránsito son servicios sintéticos; electricidad cuenta con aplicación persistente en `bank-core`, mientras agua, internet, móvil, matrícula y multas conservan el flujo legacy sintético.

**Evidencia pendiente:** persistencia durable de estados, recuperación tras reinicio, replay, aplicación idempotente y PostgreSQL como fuente de verdad de cada facturador cuando el requisito lo exija.

### P7. Outbox, Kafka y recuperación completa

**Estado:** Parcial; Phase 20 slice 2 acota una evidencia de recuperación de delivery, no el cumplimiento total de P7.

Existe outbox bancario, retries y DLQ, pero falta demostrar recuperación completa con outbox durable, integración del perfil de eventos, caída y reinicio de Kafka, reordenamiento, recuperación de bancos, backup/restore, RPO/RTO y reconciliación posterior a interrupciones. Phase 20 slice 2 (`job-4`) aporta evidencia acotada para una entrega individual `FAILED`/`DELIVERY_DLQ`, su replay autenticado e idempotente y la reconciliación del transfer objetivo; no certifica el cumplimiento total de P7.

**Límite:** Kafka no sustituirá a los ledgers PostgreSQL como fuente de verdad.

### P8. Observabilidad

**Estado:** Parcial.

Faltan OpenTelemetry, Prometheus, Grafana, Jaeger, dashboards, alertas, métricas por banco, settlement, outbox y DLQ, además de correlación completa de traces y runbooks operativos basados en métricas.

**Evidencia pendiente:** stack Docker reproducible, consultas, umbrales y escenarios de incidente.

### P9. Pruebas de carga

**Estado:** Parcial.

CholoLab existe como laboratorio, pero todavía no existe evidencia reproducible del sistema financiero completo para los perfiles de carga definidos por `spec.md`.

**Evidencia pendiente:** TPS, latencia, locks, contención, outbox, DLQ, reconciliación, recuperación, CPU/memoria y metodología con hardware, seed y commit identificados.

**Límite:** No se afirmará capacidad productiva de 1M TPS sin evidencia específica.

### P10. Trazabilidad documental completa

**Estado:** Parcial.

Falta consolidar la matriz completa:

```text
requisito de spec.md
→ tarea ODD
→ commit
→ código
→ prueba
→ evidencia Docker
→ limitación conocida
```

**Evidencia pendiente:** matriz actualizada y reconciliada después de cada slice.

## Orden de ejecución

Se resolverán en este orden, salvo que una dependencia técnica obligue a reordenar un slice:

1. P1 — Tres bancos realmente operativos.
2. P2 — Transferencia interbancaria completa.
3. P3 — APIs bancarias completas.
4. P4 — CholoKey y QR.
5. P5 — Seguridad de usuario y servicio.
6. P6 — Facturadores persistentemente respaldados.
7. P7 — Outbox, Kafka y recuperación completa.
8. P8 — Observabilidad.
9. P9 — Pruebas de carga.
10. P10 — Trazabilidad documental completa y cierre de brechas.

Cada punto se marcará como `Pendiente`, `En progreso`, `Parcial`, `Bloqueado` o `Completado`, con referencias a sus tareas, commits y pruebas. `Completado` solo significa que existe evidencia suficiente para el alcance sintético definido; no implica liquidación real, cumplimiento normativo ni despliegue productivo.

## Registro de progreso

- 2026-11-17: se registran los diez bloques pendientes después de cerrar Phase 15. La repetición T7 posterior a `cdc1fce` queda como evidencia no disponible debido al agotamiento del pool de redes Docker; la T7 previa y la suite `bank-core` permanecen documentadas por separado.
- 2026-11-17: Usuario: "Los 10 puntos que me marcaste me parecen importantes, anótalos en la parte donde documentas que esos son los puntos pendientes y vamos resolviendo uno por uno". Se crea este registro como fuente de seguimiento y se selecciona P1 como siguiente slice.
- 2026-11-17: La exploración de P1 confirma que existen `costa-db`, `sierra-db` y `oriente-db`, pero `infra/compose.yaml` solo ejecuta una instancia `bank-core` conectada a Costa. El siguiente slice debe añadir las tres instancias parametrizadas, identidades y secretos independientes, routing de receiver credits por participante, fixtures diferenciados y una prueba Docker de aislamiento. Las webs quedan fuera de P1 porque sus proyecciones reales pertenecen a la brecha de APIs/web.
- 2026-11-17: Phase 16 añadió las instancias Sierra y Oriente, fixtures V9 y una identidad interna protegida por token. La aceptación `job-2` pasó con tres identidades correctas y rechazo de todos los tokens cruzados. La suite completa `bank-core-test` pasó en Docker como `job-3`. ASSESS clasificó el candidato como riesgo medio con `reviewDue=false`, por lo que no se inició review nativa adicional. El routing completo por participante y la transferencia interbancaria permanecen en P2.
- 2026-11-18: Usuario: "listo dale con el siguiente pendiente". Se inicia Phase 17 para enrutar entregas de settlement por participante hacia Costa, Sierra u Oriente, conservando at-least-once, retries, leases, `delivery_id`, `event_id` e idempotencia. Las APIs bancarias de lectura permanecen en P3.
- 2026-11-18: Phase 17 completó el routing configurable por `receiver_participant_id`, con URLs y secretos sintéticos independientes para Costa, Sierra y Oriente. `job-4` pasó toda la suite de red Docker. P2 permanece parcial porque aún no existe una aceptación E2E que cree una liquidación y observe inbox, journal, retry y replay en Sierra/Oriente; las APIs públicas de lectura se mantienen en P3.
- 2026-11-18: Usuario: "continua porfa". Se inicia Phase 18 para construir esa aceptación E2E dentro de Compose, sin convertir todavía las consultas de balances/journals en APIs públicas de P3.
- 2026-11-18: Phase 18 pasó `job-11`: settlement y entrega idempotente completados para Sierra y Oriente, con delivery persistente, replay y un único inbox/journal por entrega. El dispatcher ahora usa Base64, compatible con bank-core. La caída real de un receptor no se inyectó en este slice y las APIs públicas de lectura permanecen en P3.
- 2026-11-18: El work unit de Phase 18 es `499377c 20261118: acepta entrega interbancaria E2E`; `job-12` pasó nuevamente la suite Docker de red. ASSESS sobre `f25851d..499377c` reportó riesgo medio y `reviewDue=false`, por lo que no se inició review nativa adicional.
- 2026-11-18: La review `review-5232591bb22a1a4c` detectó `R3-cross-bank-fixtures`: V9/V10 sembraban cuentas cruzadas en cada base. La autoridad terminó `corrected_candidate_unavailable`; la corrección nueva condiciona las migraciones con `PARTICIPANT_ID` y `job-13` confirmó aislamiento. Requiere una review nueva.
- 2026-11-18: Se inicia Phase 19 / P3 en `odd/tasks/phase-19-banking-public-apis.md`. T1 añade onboarding sintético, cuentas participant/customer-scoped y balances públicos fail-closed en bank-core, con V12, OpenAPI y pruebas de autorización/aislamiento. `job-2` pasó la suite bancaria estándar y `job-4` pasó la suite Docker de integración (`mvn -Pintegration test`). Refunds, movimientos/journals, participantes, CholoKey y Problem Details uniforme permanecen pendientes.
- 2026-11-18: T2 añade movimientos/journals públicos con cursor Base64URL opaco, orden estable por timestamps persistidos, aislamiento por cliente/cuenta y Problem Details `application/problem+json` para las rutas nuevas. `job-12` pasó la suite estándar y `job-13` pasó `mvn -Pintegration test` tras normalizar mensajes al español. Payment intents completos, refunds, participantes y CholoKey siguen pendientes.
- 2026-11-18: T3 añade consulta pública de payment intents, refunds sintéticos full-only idempotentes con `ledger_boundary: not_claimed` y consulta administrativa del participante configurado. La cancelación se excluyó porque `PENDING_APPLICATION` ya tiene débito/journal local y no se puede cambiar de estado sin compensación. `job-17` y `job-18` pasaron las suites bancaria estándar e integración; `job-20` validó `api/bank.openapi.yaml` en Docker. CholoKey permanece pendiente.
- 2026-11-18: T4a completa el lifecycle interno de CholoKey en `network`: creación, suspensión, bloqueo, baja/tombstone, autorización por participante, límites 5/20, unicidad activa e historial append-only mediante V9. `job-26` pasó la suite completa `network-test` en Docker; T4b (facade bancaria/QR) y T4c (contrato OpenAPI/protección contra enumeración) siguen pendientes.
- 2026-11-18: T4b añade facade bancaria participant/customer-scoped, delegación autenticada al directorio, lifecycle público y QR sintético estático/dinámico con expiración y replay durable mediante V15. `job-28` y `job-29` pasaron las suites bancaria estándar e integración; `job-30` pasó `network-test`. T4c debe actualizar los contratos OpenAPI y validar las rutas dentro de Docker.

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

**Estado:** Parcial.

Falta cerrar y demostrar el recorrido completo `Banco Costa → CholoNet → Banco Sierra/Oriente → receiver inbox → ledger receptor → consulta desde UI`.

**Evidencia pendiente:** replay, reordenamiento, caída del receptor, retry, DLQ, reconciliación, saldos y journals de emisor y receptor.

### P3. APIs bancarias completas

**Estado:** Parcial.

El contrato bancario versionado existe, pero todavía faltan o están incompletas superficies para customers/onboarding, cuentas, balances, movimientos, journals, payment intents públicos completos, refunds, participantes, CholoKey, paginación cursor y Problem Details uniforme.

**Evidencia pendiente:** implementación pública, autorización, OpenAPI actualizado y validación automática del contrato en CI Docker.

### P4. CholoKey y QR

**Estado:** Parcial.

Existe resolución y parte del directorio, pero falta completar creación pública, bloqueo, suspensión, baja, ownership claim, límites por cuenta, historial de portabilidad, alias sintéticos, QR estático y QR dinámico.

**Evidencia pendiente:** pruebas contra los casos de `spec.md`, protección contra enumeración y autorización participant-scoped.

### P5. Seguridad de usuario y servicio

**Estado:** Parcial.

Existen mecanismos sintéticos como HMAC, bearer y API key, pero falta cubrir OIDC/OAuth 2.1, PKCE, MFA/step-up, JWT completo, mTLS operativo, TLS, rotación y hash de secretos, rate limiting, CORS/CSRF y autorización exhaustiva por cuenta y participante.

**Límite:** Las implementaciones seguirán siendo educativas y no se presentarán como certificación de cumplimiento normativo.

### P6. Facturadores persistentemente respaldados

**Estado:** Parcial.

Utilities y tránsito son servicios sintéticos; electricidad cuenta con aplicación persistente en `bank-core`, mientras agua, internet, móvil, matrícula y multas conservan el flujo legacy sintético.

**Evidencia pendiente:** persistencia durable de estados, recuperación tras reinicio, replay, aplicación idempotente y PostgreSQL como fuente de verdad de cada facturador cuando el requisito lo exija.

### P7. Outbox, Kafka y recuperación completa

**Estado:** Parcial.

Existe outbox bancario, retries y DLQ, pero falta demostrar recuperación completa con outbox durable, integración del perfil de eventos, caída y reinicio de Kafka, reordenamiento, recuperación de bancos, backup/restore, RPO/RTO y reconciliación posterior a interrupciones.

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

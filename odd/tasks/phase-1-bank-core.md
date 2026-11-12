# CholoPay Network — Núcleo bancario de la fase 1

Fuente de verdad del primer slice financiero ejecutable. Lee hasta `## Log` antes de trabajar.

## Specs

1. **S1 — Phase 1 scope:** "Fase 1 — Banco simulado y ledger" delivers "Spring Boot con cuentas ficticias, journal inmutable, reservas, transferencias internas, migraciones, pruebas de invariantes".
2. **S2 — Acceptance:** "pruebas concurrentes sin saldos negativos ni asientos desequilibrados".
3. **S3 — Accounting:** "Partida doble en ledger bancario y ledger de liquidación: cada asiento tiene débitos/créditos equilibrados en la misma moneda." Amounts are "enteros en centavos (`BIGINT` ...); nunca `float`/`double` para dinero".
4. **S4 — Safety:** "Los movimientos publicados son inmutables; correcciones y devoluciones crean movimientos compensatorios." and "una misma instrucción de negocio no produce dos liquidaciones".
5. **S5 — Internal transfer:** "Intra-banco: débito y crédito en una única transacción del banco, con ledger equilibrado e idempotencia." 
6. **S6 — Synthetic data:** "Datos exclusivamente sintéticos" y no se usan datos financieros ni de identidad reales.
7. **S7 — Compatibility:** La persona usuaria requiere mensajes de commit en español, comenzando con `yyyymmdd`, con un comentario que describa lo realizado.
8. **S8 — Container-only execution:** "compa todo debe ser en los contenedores, no uses nada de la máquina local, el proyecto debe ser de tal forma que alguien se descargue el proyecto y en cualquier maquina lo pueda correr siempre y cuando tenga docker."

## Tasks

- T1 (S1, S3): Scaffold parametrized Spring Boot bank-core module with PostgreSQL migration layout, domain boundaries, and container-only build instructions. Commit: `667cfc2`.
- T2 (S3, S4): Implementar ledger en centavos enteros, journals equilibrados, asientos inmutables y proyección de cuentas. Commit: `667cfc2`.
- T3 (S2, S5): Implementar reservas locales y transferencias internas idempotentes con pruebas concurrentes de invariantes. Commit: `667cfc2`.
- T4 (S1, S6): Agregar documentación de API/errores y fixtures de pruebas sintéticos. Commit: `667cfc2`.
- T5 (S1-S8): Ejecutar compilación/pruebas, registrar evidencia, exigir ejecución exclusiva de Docker y seleccionar la siguiente decisión de integración de red. Commit: pending.
- T6 (S8): Eliminar requisitos de toolchains y puertos del host del flujo predeterminado; agregar compilación Docker multietapa y servicio de pruebas Compose. Commit: pending.
- T7 (S1, S3, S5): Reemplazar el estado de transferencias en memoria por persistencia transaccional en PostgreSQL. Commit: pending.
- T8 (S2, S3): Completar la aceptación JDBC concurrente contra PostgreSQL con saldo limitado, sin saldos negativos, sin journals duplicados y con todos los journals equilibrados. Ruta: siguiente slice. Commit: `680395b` y ajuste `d496770`.
- T9 (S2, S3, S4): Endurecer la aceptación concurrente y de conflictos del receiver inbox sin modificar el contrato público ni la migración existente. Ruta: siguiente slice. Commit: `b54ba77` y evidencia `4a4944d`.
- T10 (S1, S3, S5, S8): Implementar el outbox transaccional conforme a `adr/0013-transactional-outbox-contract.md`, con nueva orden inter-red interna, UUID propio, publicación por orquestación y HMAC sintético banco→red. Incluye persistencia atómica, dispatcher, leases, reintentos, DLQ e idempotencia. Ruta: siguiente slice. Commit: pendiente.
- T10a (S1, S3, S5): Implementar el primer slice persistente del outbox: migración V4, envelope inmutable, escritura atómica desde una orden inter-red interna y replay/conflicto idempotente. Dispatcher y publicación HMAC quedan para T10b. Ruta: siguiente slice. Commit: `4fb04b5`.
- T10b (S1, S3, S5, S8): Implementar dispatcher con leases, reintentos acotados, backoff/jitter, DLQ, publicación por orquestación y autenticación HMAC sintética fail-closed, sin escribir tablas de CholoNet. Ruta: siguiente slice. Commit: pendiente.

La evidencia de ejecución exclusiva en contenedores y persistencia PostgreSQL se registra abajo antes del commit posterior.

La evidencia de implementación se registra abajo antes de crear el commit de la fase 1.

## Log

L1. User request: "los commits son en español, ademas deben tener el siguiente formato yyyymmdd y comentarios de lo hecho, cambia los nombres de los commit y guarda esta regla para el proyecto. Luego de eso continua con la fase 1"
L2. Los commits de la fase 0 se reescribieron como `20261109: establece la arquitectura de la fase 0` and `20261109: registra la evidencia de la fase 0`.
L3. La fase 1 comienza con el ledger del núcleo bancario y el slice de transferencias internas; todavía no se implementan el protocolo interbancario ni las interfaces de usuario.
L4. El núcleo bancario usa Spring Boot 3.4.5, Java 21, Flyway, PostgreSQL y únicamente datos sintéticos. Los importes son centavos enteros de USD.
L5. `docker run --rm -v "$PWD/bank-core:/workspace" -w /workspace maven:3.9-eclipse-temurin-21 mvn -q test` passed with exit code 0. `git diff --check` passed. Maven was executed in a container because Maven and Gradle are not installed on the host.
L6. Las pruebas cubren rechazo de journals desequilibrados, replay idempotente con la misma clave, conflicto de payload diferente, transferencias concurrentes contra un saldo limitado, disponibilidad no negativa y exactamente diez transferencias exitosas de 100 centavos desde una cuenta de 1.000 centavos.
L7. Luego la persona usuaria requirió que todo el trabajo del proyecto se ejecutara dentro de contenedores y que un checkout nuevo requiriera únicamente Docker. La pila Compose predeterminada ahora no publica puertos del host; la compilación y las pruebas usan `bank-core/Dockerfile` multietapa y el perfil Compose `bank-core-test`. El siguiente paso es la persistencia transaccional en PostgreSQL, también compilada y probada únicamente dentro de contenedores.
L8. `docker compose ... config --quiet` passed for the default and test profiles. `./ci/phase-0.sh` passed using a Python validation container and Docker Compose; no host language runtime was used for project checks.
L9. `docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile test run --rm bank-core-test` pasó con código de salida 0. El perfil de pruebas Compose inició PostgreSQL dentro de Docker, aplicó las migraciones Flyway y superó la prueba de integración JDBC de persistencia, asientos equilibrados y replay idempotente.
L10. Se agregaron proyecciones de cuentas Flyway V2 y hashes de solicitudes, `JdbcInternalTransferService` transaccional, cableado del controlador y una prueba de integración. La pila predeterminada no publica puertos del host; por lo tanto, no se requirió preflight de puertos en esta ejecución. El siguiente paso es exponer la API bancaria persistente únicamente mediante una ruta de red interna revisada explícitamente antes de la integración interbancaria.
L11. Solicitud vigente: "dale continua con los specs, dime donde puedo ver todas las paginas ya para probar". Las cuatro páginas web se levantaron en Docker con puertos temporales 8081-8084; el siguiente slice técnico seleccionado es T8 porque la aceptación JDBC concurrente aún no está demostrada en la evidencia actual.
L12. T8 implementado y verificado en Docker: 100 transferencias concurrentes de 100 centavos contra una cuenta de 1.000 centavos producen exactamente 10 éxitos, saldo disponible no negativo, 10 journals/transacciones/idempotency keys únicos y todos los journals equilibrados en USD. La prueba focalizada `JdbcInternalTransferIntegrationTest` pasó con código 0; `git diff --check` pasó.
L13. Se atendieron las advertencias informativas de la review: la prueba ahora comprueba explícitamente el mensaje de fondos insuficientes y espera cada `Future` con timeout de cinco segundos. La prueba Docker focalizada y `git diff --check` volvieron a pasar.
L14. Solicitud vigente: "arregla y continua con las specs". T8 quedó revisado y aprobado; el siguiente slice no especulativo es T9, endurecimiento del receiver inbox existente, sin inventar todavía un contrato de outbox.
L15. T9 implementado y verificado en Docker: el receiver inbox bloquea las cuentas de destino y compensación en orden determinista antes de consultar la entrega, serializa duplicados concurrentes y conserva replay/conflicto sin efectos financieros duplicados. Las pruebas focalizadas de integración y HTTP pasaron con código 0; `git diff --check` pasó.
L16. La suite completa de integración `mvn -q -Pintegration test` pasó dentro del perfil Docker `bank-core-test`; `git diff --check` pasó. T9 queda cerrado en `b54ba77` y no modifica migraciones ni endpoints públicos.
L17. Solicitud vigente: "define el contrato". Se aceptó `adr/0013-transactional-outbox-contract.md`, que define `TRANSFER_REQUESTED` versión `1.0`, identidades estables, payload, estados `PENDING/IN_FLIGHT/PUBLISHED/DEAD_LETTER`, reintentos, límites de propiedad y recuperación. La implementación T10 aún no está ejecutada.
L18. Decisiones de integración autorizadas: una nueva orden inter-red interna crea el evento; el adaptador genera un UUID propio como `transfer_id`; se publica por `/internal/v1/inter-network/transfers`; y se usa HMAC sintético banco→red separado de `NETWORK_ADAPTER_SECRET`. La ruta local permanece local-only.
L19. Se inicia T10a como primer slice revisable: persistencia atómica e idempotencia del outbox antes de añadir dispatcher, leases y publicación HMAC.
L20. T10a implementado y verificado dentro de Docker: Flyway V4 crea `inter_network_outbox`; la nueva orden inter-red persiste el envelope `TRANSFER_REQUESTED` junto con el journal local; replay idéntico conserva `transfer_id`/`event_id`/hash; conflicto no altera saldos, journals ni outbox. `docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile test run --rm bank-core-test` terminó con código 0 y `git diff --check` pasó. T10a queda cerrado en `4fb04b5`.
L21. T10b implementado y verificado en Docker: el dispatcher reclama eventos con `FOR UPDATE SKIP LOCKED`, recupera leases vencidos con `lease_until <= now`, conserva el envelope serializado, clasifica éxito/replay, reintentos ambiguos y DLQ, y firma/verifica HMAC fail-closed. Las pruebas de repositorio cubren exclusividad concurrente, recuperación de lease, reemplazo de token y transiciones protegidas; suites bank-core y network pasaron. T10b queda listo para registrar en un commit separado.
L22. La review detectó que `dispatchOnce` no tenía invocación operativa. Se añadió `OutboxScheduler`, `@EnableScheduling` y `OUTBOX_ENABLED` con valor predeterminado `false`, más `OUTBOX_POLL_DELAY_MS`; la activación queda explícita y fail-closed por entorno Docker. La suite bank-core volvió a pasar. La corrección quedó registrada en `0823154`.
L23. La review detectó que V4 había sido modificada después de su publicación. V4 fue restaurada sin cambios y V5 agrega de forma idempotente las columnas del dispatcher, backfill de `serialized_envelope`, constraint de estados e índices, tolerando un esquema parcialmente actualizado. La suite `docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile test run --build --rm bank-core-test` pasó y `git diff --check` pasó; la corrección queda pendiente de commit.

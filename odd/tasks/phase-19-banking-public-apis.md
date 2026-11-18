# Phase 19 — APIs bancarias públicas
Rama: `phase-19-banking-public-apis` · alcance: P3 de `docs/spec-pending-work.md` · estado: en progreso
Fuente: `spec.md`, P3 pendiente y decisión del usuario de continuar con el siguiente pendiente.

## Specs
1. **S1 — P3 completo:** "El contrato bancario versionado existe, pero todavía faltan o están incompletas superficies para customers/onboarding, cuentas, balances, movimientos, journals, payment intents públicos completos, refunds, participantes, CholoKey, paginación cursor y Problem Details uniforme."
2. **S2 — Evidencia:** "implementación pública, autorización, OpenAPI actualizado y validación automática del contrato en CI Docker."
3. **S3 — Límites:** El proyecto continúa siendo educativo, sintético y Docker-only; no se implementa dinero real, liquidación productiva ni compatibilidad normativa real.
4. **S4 — Continuidad:** "Se resolverán uno por uno"; este work unit empieza por una base pública participant-scoped para customers, cuentas y balances, manteniendo payment intents legacy compatibles.

## Tasks
- T1 [S1-S4] Implementar customers sintéticos, cuentas y balance público con autorización y aislamiento. Ruta: bank-core + migración + pruebas Docker. Commit: `4cd6709`.
- T2 [S1-S2] Añadir movimientos/journals, paginación cursor y Problem Details uniforme. Ruta: bank-core + OpenAPI + pruebas Docker. Commit: `7b50046`.
- T3 [S1-S2] Completar payment intents públicos, refunds y participantes. Ruta: bank-core + OpenAPI + pruebas Docker. Commit: `17b46a3`. Autorización: token de cliente para consultas/refunds públicos; `BANK_API_TOKEN` para operaciones administrativas.
- T4a [S1-S2] Completar lifecycle del directorio CholoKey: creación, suspensión/bloqueo, baja, límites, unicidad e historial append-only. Ruta: network + migración + pruebas Docker. Commit: pendiente.
- T4b [S1-S2] Añadir facade bancaria participant/customer-scoped para CholoKey y QR sintético verificable. Ruta: bank-core/network + OpenAPI + pruebas Docker. Commit: `PENDIENTE_T4B`.
- T4c [S1-S2] Validar OpenAPI de bank/network dentro de Docker y actualizar evidencia P4. Ruta: infra + docs + pruebas Docker. Commit: pendiente.
- T5 [S1-S4] Integrar evidencia, limitaciones y cierre de P3. Ruta: docs + aceptación Docker + review RDD. Commit: pendiente.

## Log
- L1. Usuario: "continua con el siguiente pendiente"
- L2. La exploración confirmó que `api/bank.openapi.yaml` anuncia customers, cuentas y `/v1/keys`, pero bank-core no tiene implementación pública equivalente; los servicios existentes son principalmente internos u obligación-específicos.
- L3. El primer slice implementará customers sintéticos, cuentas y balances participant-scoped antes de ampliar payment intents, refunds, participantes y CholoKey.
- L4. T1 quedó en `4cd6709 20261118: habilita cuentas públicas sintéticas`. `job-2` pasó la suite bancaria Docker estándar, `job-4` pasó `mvn -Pintegration test` en Docker y `git diff --check` pasó. La API devuelve tokens sintéticos de cliente y aplica aislamiento por participante/cliente; no representa autenticación productiva.
- L5. T2 implementa movimientos y journals participant/customer-scoped, cursor Base64URL opaco con orden estable `(created_at,id)`, timestamps persistidos y Problem Details para las rutas nuevas. `job-8` pasó la suite bancaria estándar y `job-10` pasó `mvn -Pintegration test` en Docker. Los endpoints legacy conservan `{error,message}`.
- L6. Tras normalizar títulos y detalles Problem Details al español, `job-12` pasó la suite estándar y `job-13` pasó `mvn -Pintegration test` en Docker. Las pruebas verifican cursor de dos páginas, aislamiento, límites, autorización y contenido Problem Details.
- L7. La exploración de T3 confirmó que bank-core solo expone POST de obligation payment-intents; no existen consulta de estado, cancelación, refunds ni recursos públicos de participantes. Decisión del usuario: separar autorización; consultas/refunds públicos usarán token de cliente y operaciones administrativas usarán `BANK_API_TOKEN`.
- L8. T3 añade consulta pública de intents, refunds sintéticos full-only idempotentes con `ledger_boundary: not_claimed` y consulta administrativa del participante configurado. Se excluyó cancelación porque `PENDING_APPLICATION` ya tiene débito/journal local y no admite una transición segura sin compensación. `job-17` y `job-18` pasaron las suites Docker; `job-20` validó el YAML OpenAPI después de corregir CustomerCreated y AccountList.
- L9. La exploración de P4 confirmó que el directorio CholoKey ya tiene límites 5/20, unicidad, suspensión y portabilidad internas en `network`, pero faltan creación/gestión pública, estados de baja/bloqueo, historial append-only, facade bancaria, QR backend, protección contra enumeración y validación OpenAPI Docker. T4 se divide en T4a lifecycle del directorio, T4b facade/QR y T4c contrato/evidencia.
- L10. T4a implementa creación, suspensión, bloqueo, baja/tombstone, autorización por participante, unicidad activa, límites 5/20 e historial append-only con V9. `job-24` y `job-25` detectaron y permitieron corregir colisiones de idempotencia vacía y una inserción de historial de portabilidad mal formada; `job-26` pasó la suite completa `network-test` en Docker.
- L11. T4b añade facade bancaria participant/customer-scoped, delegación autenticada al directorio, lifecycle público, QR sintético estático/dinámico versionado y replay/expiración durable mediante V15. `job-28` y `job-29` pasaron la suite bancaria estándar e integración Docker; `job-30` pasó `network-test` con las rutas internas del directorio. No se modificó el ledger.

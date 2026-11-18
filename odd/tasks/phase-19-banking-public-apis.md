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
- T2 [S1-S2] Añadir movimientos/journals, paginación cursor y Problem Details uniforme. Ruta: bank-core + OpenAPI + pruebas Docker. Commit: pendiente.
- T3 [S1-S2] Completar payment intents públicos, refunds y participantes. Ruta: bank-core + OpenAPI + pruebas Docker. Commit: pendiente.
- T4 [S1-S2] Completar CholoKey público y validación automática del contrato. Ruta: bank-core + OpenAPI + CI Docker. Commit: pendiente.
- T5 [S1-S4] Integrar evidencia, limitaciones y cierre de P3. Ruta: docs + aceptación Docker + review RDD. Commit: pendiente.

## Log
- L1. Usuario: "continua con el siguiente pendiente"
- L2. La exploración confirmó que `api/bank.openapi.yaml` anuncia customers, cuentas y `/v1/keys`, pero bank-core no tiene implementación pública equivalente; los servicios existentes son principalmente internos u obligación-específicos.
- L3. El primer slice implementará customers sintéticos, cuentas y balances participant-scoped antes de ampliar payment intents, refunds, participantes y CholoKey.
- L4. T1 quedó en `4cd6709 20261118: habilita cuentas públicas sintéticas`. `job-2` pasó la suite bancaria Docker estándar, `job-4` pasó `mvn -Pintegration test` en Docker y `git diff --check` pasó. La API devuelve tokens sintéticos de cliente y aplica aislamiento por participante/cliente; no representa autenticación productiva.
- L5. T2 implementa movimientos y journals participant/customer-scoped, cursor Base64URL opaco con orden estable `(created_at,id)`, timestamps persistidos y Problem Details para las rutas nuevas. `job-8` pasó la suite bancaria estándar y `job-10` pasó `mvn -Pintegration test` en Docker. Los endpoints legacy conservan `{error,message}`.
- L6. Tras normalizar títulos y detalles Problem Details al español, `job-12` pasó la suite estándar y `job-13` pasó `mvn -Pintegration test` en Docker. Las pruebas verifican cursor de dos páginas, aislamiento, límites, autorización y contenido Problem Details.

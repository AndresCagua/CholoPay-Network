# Phase 19 — APIs bancarias públicas
Rama: `phase-19-banking-public-apis` · alcance: P3 de `docs/spec-pending-work.md` · estado: en progreso
Fuente: `spec.md`, P3 pendiente y decisión del usuario de continuar con el siguiente pendiente.

## Specs
1. **S1 — P3 completo:** "El contrato bancario versionado existe, pero todavía faltan o están incompletas superficies para customers/onboarding, cuentas, balances, movimientos, journals, payment intents públicos completos, refunds, participantes, CholoKey, paginación cursor y Problem Details uniforme."
2. **S2 — Evidencia:** "implementación pública, autorización, OpenAPI actualizado y validación automática del contrato en CI Docker."
3. **S3 — Límites:** El proyecto continúa siendo educativo, sintético y Docker-only; no se implementa dinero real, liquidación productiva ni compatibilidad normativa real.
4. **S4 — Continuidad:** "Se resolverán uno por uno"; este work unit empieza por una base pública participant-scoped para customers, cuentas y balances, manteniendo payment intents legacy compatibles.

## Tasks
- T1 [S1-S4] Implementar customers sintéticos, cuentas y balance público con autorización y aislamiento. Ruta: bank-core + migración + pruebas Docker. Commit: pendiente.
- T2 [S1-S2] Añadir movimientos/journals, paginación cursor y Problem Details uniforme. Ruta: bank-core + OpenAPI + pruebas Docker. Commit: pendiente.
- T3 [S1-S2] Completar payment intents públicos, refunds y participantes. Ruta: bank-core + OpenAPI + pruebas Docker. Commit: pendiente.
- T4 [S1-S2] Completar CholoKey público y validación automática del contrato. Ruta: bank-core + OpenAPI + CI Docker. Commit: pendiente.
- T5 [S1-S4] Integrar evidencia, limitaciones y cierre de P3. Ruta: docs + aceptación Docker + review RDD. Commit: pendiente.

## Log
- L1. Usuario: "continua con el siguiente pendiente"
- L2. La exploración confirmó que `api/bank.openapi.yaml` anuncia customers, cuentas y `/v1/keys`, pero bank-core no tiene implementación pública equivalente; los servicios existentes son principalmente internos u obligación-específicos.
- L3. El primer slice implementará customers sintéticos, cuentas y balances participant-scoped antes de ampliar payment intents, refunds, participantes y CholoKey.

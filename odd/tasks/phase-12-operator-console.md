# Phase 12 — Consola central de supervisión simulada
Fuente: `spec.md`, secciones 25, 26.9 y 27.
Alcance: proyecciones agregadas de solo lectura sobre CholoNet, bancos y facturadores; no mutación de cuentas, ledgers, facturas ni multas.

## Specs

1. **S1 — Supervisión**
   > El panel de autoridad **simulada** debe representar la función de operación y vigilancia técnica de la red; no debe presentarse como el sistema real del Banco Central del Ecuador ni atribuirse competencias legales.

2. **S2 — Incidencias**
   > Incidencias: caídas de bancos o facturadores, pagos liquidados pendientes de aplicar, duplicados rechazados, fallos de webhook, retrasos y desbalances.

3. **S3 — Roles**
   > Roles: `NETWORK_ADMIN`, `SETTLEMENT_OPERATOR`, `BILLER_SUPERVISOR`, `AUDITOR`, `READ_ONLY`; segregación de funciones y trazabilidad de toda consulta sensible.

4. **S4 — Privacidad**
   > vistas agregadas por defecto, búsqueda individual solo por rol/propósito autorizado, identificadores enmascarados, retención definida y prohibición de alterar facturas o multas desde el panel central.

5. **S5 — Fuente de datos**
   > Fuente de datos: eventos confirmados y proyecciones de solo lectura con eventual consistencia; cada facturador conserva la autoridad sobre su obligación y cada banco sobre su ledger. Mostrar fecha de última sincronización y estado de conciliación.

## Invariantes

- La consola nunca escribe en ledgers bancarios ni en bases de facturadores.
- Las consultas sensibles son participant-scoped, agregadas o enmascaradas.
- Cada proyección muestra origen, última sincronización y estado de conciliación.
- Fallos de una fuente no deben inventar saldos, estados ni obligaciones.

## Tareas

- T1 (S1-S5): Inventariar fuentes y contratos de solo lectura existentes, incluyendo autorización y alcance de datos. Ruta: inline. Commit: pending.
- T2 (S1-S5): Implementar endpoint/proyección agregada de incidencias y salud de participantes. Ruta: worker inline. Commit: pending.
- T3 (S1-S5): Añadir regresiones de aislamiento, enmascaramiento, datos obsoletos y no-mutación. Ruta: worker inline. Commit: pending.
- T4 (S1-S5): Documentar consola simulada y evidencia Docker. Ruta: inline. Commit: pending.

## Log

L1. Usuario: "papito hermoso pero como vergas me puedes tener ya una hora detenido en la misma guebada. avanza termina inventa, no lo tomes en cuenta, haz una puta cosa pero avanza"
L2. La revisión RDD de Phase 11 fue abandonada explícitamente por disposición del operador; el target queda sin aprobación nativa. Se continúa con la siguiente especificación: consola central simulada read-only.
L3. Se añadió `GET /internal/v1/operator/overview`, una proyección agregada de solo lectura con `schema_version`, origen, timestamp de sincronización, estado de conciliación y conteos de redes, participantes y CholoKeys activas/suspendidas.
L4. Evidencia Docker: `docker compose config` PASS; suite completa de `network` PASS; `gofmt` y `git diff --check` PASS.
L5. T2 implementó salud agregada e incidencias desde tablas propias de CholoNet, mantuvo compatibilidad de campos existentes, devolvió `503` sin valores inventados cuando la proyección no está disponible y documentó el contrato OpenAPI. Evidencia Docker aislada: `docker compose -p cholopay-operator-verify --profile network-test -f infra/compose.yaml run --rm network-test` PASS; `docker compose config` y `git diff --check` PASS.
L6. T3 añadió regresiones deterministas para autorización, agregación, privacidad, ausencia de datos fuente, no-mutación y consultas exclusivamente `SELECT`. Evidencia Docker aislada: `docker compose -p cholopay-operator-t3 --profile network-test -f infra/compose.yaml run --rm network-test` PASS.
L7. T4 documentó la consola simulada, sus límites de privacidad, el comportamiento fail-closed ante fuentes no disponibles y la evidencia de ejecución Docker en `docs/runbooks/phase-12-operator-console.md`.

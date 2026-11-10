# Runbook — Consola central simulada

La consola central de CholoPay es una proyección educativa, agregada y de solo lectura. No representa al Banco Central del Ecuador ni modifica ledgers bancarios, obligaciones de facturadores o facturas.

## Quick path

1. Configurá `OPERATOR_CONSOLE_TOKEN` en el entorno Docker.
2. Consultá `GET /internal/v1/operator/overview` desde una red Docker autorizada con `Authorization: Bearer <token>`.
3. Verificá `schema_version`, `source`, `last_synchronized_at`, `health` e `incidents`.
4. Si una fuente de la proyección no está disponible, esperá `503`; no se deben mostrar ceros ni datos anteriores como si fueran actuales.

## Proyección disponible

| Campo | Significado |
|---|---|
| `source` | Read model propiedad de CholoNet; no es una lectura directa de un banco o facturador. |
| `last_synchronized_at` | Momento en que la red generó la proyección. |
| `reconciliation_status` | Estado agregado de conciliación; `UNKNOWN` no significa que no existan diferencias. |
| `health` | Totales y estados agregados de redes, participantes y CholoKeys. |
| `incidents` | Conteos por código y fuente, sin identificadores individuales. |

Los incidentes actualmente observables desde las fuentes de red incluyen participantes o redes suspendidas, entregas de crédito pendientes, diferencias de conciliación y transferencias rechazadas. Los fallos de webhooks o de facturadores no se inventan porque esas fuentes no pertenecen al esquema de CholoNet actual.

## Privacidad y límites

- El endpoint requiere bearer token y está destinado a roles operatorios autorizados.
- La respuesta es agregada: no incluye `participant_id`, cuentas, secretos, tokens, direcciones de wallet ni saldos bancarios.
- La consola no tiene rutas de escritura y sus consultas SQL son de lectura.
- Cada banco conserva la autoridad sobre su ledger; cada facturador conserva la autoridad sobre sus obligaciones.
- La ausencia o caída de una fuente produce una respuesta no disponible, no una proyección fabricada.

## Evidencia Docker

Desde la raíz del repositorio:

```bash
docker compose -p cholopay-operator-t3 \
  --profile network-test \
  -f infra/compose.yaml \
  run --rm network-test
```

Resultado esperado: todos los paquetes Go de `network` terminan con `ok` y código de salida `0`.

Validaciones complementarias:

```bash
docker compose -f infra/compose.yaml config
git diff --check
```

Estas verificaciones no publican puertos y usan una base PostgreSQL aislada para la ejecución de tests.

## Fuera de alcance

- Alta, suspensión o modificación de participantes desde la consola.
- Ajustes de balances o escrituras en ledgers.
- Alteración de facturas, multas u obligaciones.
- Consulta individual sin propósito y autorización explícitos.
- Integración con sistemas financieros reales o compatibilidad con protocolos regulatorios.

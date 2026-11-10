# Phase 3 — contrato de paneles web y autorización

## Alcance

La fase 3 agrega cuatro superficies web sin mover lógica financiera al navegador:

| Aplicación | Stack | Backend autorizado | Datos que puede consultar |
|---|---|---|---|
| Operador central | React + TypeScript | CholoNet | Participantes, estados de transferencias, liquidación y reconciliación según rol |
| Banco Costa | Next.js + React + TypeScript | instancia Costa de `bank-core` | Clientes, cuentas y transferencias de Costa |
| Banco Sierra | Angular + TypeScript | instancia Sierra de `bank-core` | Clientes, cuentas y transferencias de Sierra |
| Banco Oriente | React + TypeScript + Vite | instancia Oriente de `bank-core` | Clientes, cuentas y transferencias de Oriente |

Ninguna aplicación web accede directamente a PostgreSQL, al ledger de otro banco ni a tablas internas de CholoNet.

## Identidad y roles

La primera implementación usa tokens bearer sintéticos exclusivamente dentro de Docker. No representa OIDC ni autenticación de producción. Cada token contiene una identidad ficticia, `participant_id`, `roles`, `scopes`, `issuer`, `audience`, `issued_at` y `expires_at`.

| Rol | Operador | Costa | Sierra | Oriente |
|---|---:|---:|---:|---:|
| `NETWORK_ADMIN` | administrar participantes | no | no | no |
| `SETTLEMENT_OPERATOR` | consultar y operar flujos de liquidación autorizados | no | no | no |
| `BANK_OPERATOR` | no | sí | sí | sí |
| `CUSTOMER` | no | solo recursos propios | solo recursos propios | solo recursos propios |
| `AUDITOR` | lectura autorizada y reportes | lectura autorizada | lectura autorizada | lectura autorizada |
| `READ_ONLY` | lectura limitada | lectura limitada | lectura limitada | lectura limitada |

La autorización se verifica en backend por `participant_id` y recurso. La sesión de Costa nunca autoriza acceso a Sierra u Oriente.

## Lectura de transferencias

Las vistas consultan los contratos existentes:

- CholoNet: `GET /v1/transfers/{id}` para estado de red y liquidación.
- Banco: `GET /v1/accounts` y endpoints de movimientos/transferencias para proyección bancaria.
- La UI muestra por separado aceptación de instrucción, reserva, `SETTLED`, `CREDIT_PENDING` y `COMPLETED`.
- `202 Accepted` no se interpreta como liquidación.
- La vista combina únicamente respuestas autorizadas del backend; no lee bases de datos ni infiere estados.

El modelo de vista común contiene `transfer_id`, `state`, `amount_minor`, `currency`, `source_participant_id`, `destination_participant_id` cuando corresponda, `correlation_id`, `causation_id`, `idempotency_key` enmascarada, `created_at`, `settled_at` nullable y `last_updated_at`.

## Actualización y reconexión

La primera entrega usa polling explícito y revalidación después de reconectar. SSE/WebSocket queda como extensión posterior. La UI no crea otra transferencia al reintentar una consulta o una acción: conserva la misma `Idempotency-Key` y muestra el resultado existente.

## Contratos nuevos

Cualquier endpoint o evento de esta fase debe incluir:

- `schema_version`;
- `correlation_id`;
- `causation_id` cuando derive de otro evento;
- `idempotency_key` para mutaciones;
- timestamp UTC RFC 3339;
- `trace_id` y `participant_id` cuando el ámbito lo requiera.

Los clientes deben tratar `application/problem+json`, `401`, `403`, `409`, `422`, `429` y estados desconocidos como resultados explícitos, nunca como éxito implícito.

## Docker

Cada aplicación tendrá su propio build/test target y se ejecutará mediante perfiles Compose privados. No se publican puertos por defecto y no se instalan toolchains JavaScript en el host.

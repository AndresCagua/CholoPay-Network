# Fase 4 — contrato de billetera, CholoKey y QR

## Límite

La wallet Flutter es un cliente de demostración. Consulta APIs del banco autenticado y CholoNet mediante contratos HTTP; nunca accede a PostgreSQL ni escribe directamente en ledgers.

## Sesión de billetera

Una sesión sintética exclusiva de Docker contiene:

```json
{
  "session_id": "sess-0001",
  "customer_id": "customer-costa-001",
  "participant_id": "COSTA",
  "account_ids": ["account-costa-001"],
  "roles": ["CUSTOMER"],
  "scopes": ["accounts:read", "transfers:create", "transfers:read", "keys:read", "qr:read"],
  "issuer": "cholopay-bank-costa-local",
  "audience": "cholopay-wallet",
  "issued_at": "2026-11-09T00:00:00Z",
  "expires_at": "2026-11-09T01:00:00Z"
}
```

Una sesión está delimitada a un participante y no puede autorizar recursos de Sierra, Oriente ni del operador central.

## Resolución de CholoKey

La billetera muestra una referencia de destino minimizada devuelta por la API de directorio autorizada:

```json
{
  "schema_version": "1.0",
  "key": "ck_demo_receiver_001",
  "participant_id": "SIERRA",
  "account_ref": "acct-ref-sierra-001",
  "display_name": "Synthetic Receiver",
  "correlation_id": "00000000-0000-0000-0000-000000000001",
  "trace_id": "trace-wallet-001",
  "resolved_at": "2026-11-09T00:00:00Z"
}
```

La billetera no debe exponer datos bancarios privados más allá de los campos de visualización minimizados.

## Payload QR de CholoPay

Este es un formato propio de CholoPay y no es compatible con Pix:

```json
{
  "format": "cholopay.qr",
  "version": 1,
  "mode": "static",
  "participant_id": "SIERRA",
  "destination_key": "ck_demo_receiver_001",
  "merchant_ref": "merchant-demo-001",
  "amount_minor": null,
  "currency": "USD",
  "expires_at": null,
  "correlation_id": "00000000-0000-0000-0000-000000000002",
  "payload_id": "qr-payload-001",
  "created_at": "2026-11-09T00:00:00Z"
}
```

El QR dinámico usa un `amount_minor` positivo y una caducidad. La billetera valida el formato, la versión, el participante, el importe, la moneda, la caducidad y el destino antes de mostrar la confirmación.

## Confirmación de transferencia

Cada confirmación crea o reutiliza una única clave de idempotencia. Los reintentos de la interfaz nunca generan un identificador nuevo de transferencia:

```json
{
  "schema_version": "1.0",
  "transfer_id": "00000000-0000-0000-0000-000000000010",
  "idempotency_key": "wallet-confirm-000000000000001",
  "amount_minor": 2500,
  "currency": "USD",
  "source_account_id": "account-costa-001",
  "destination_key": "ck_demo_receiver_001",
  "correlation_id": "00000000-0000-0000-0000-000000000003",
  "causation_id": "00000000-0000-0000-0000-000000000004",
  "created_at": "2026-11-09T00:00:00Z",
  "trace_id": "trace-wallet-confirm-001"
}
```

La interfaz representa `CREATED`, `VALIDATING`, `FUNDS_RESERVED`, `READY_FOR_SETTLEMENT`, `SETTLED`, `CREDIT_PENDING`, `COMPLETED`, `REJECTED` y `UNKNOWN_PENDING_RECONCILIATION` de forma diferenciada. `SETTLED` no equivale a `COMPLETED`.

## Modelo de lectura de movimientos

Los movimientos son proyecciones de solo lectura de la API bancaria autenticada e incluyen `transaction_id`, `account_id`, `amount_minor`, `currency`, `direction`, `state`, `created_at`, `settled_at`, `correlation_id` y `trace_id`. La reconexión vuelve a validar la proyección del servidor antes de actualizar la interfaz.

## Metadatos contractuales obligatorios

Las solicitudes y los eventos nuevos incluyen `schema_version`, `correlation_id`, `idempotency_key` cuando corresponda, marcas de tiempo UTC y trazabilidad. Los payloads QR son sintéticos y versionados; no se afirma compatibilidad con redes externas.

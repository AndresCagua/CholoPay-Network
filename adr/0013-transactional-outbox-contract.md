# ADR 0013: contrato de outbox transaccional para eventos bancarios

- Estado: Aceptado como contrato de integración
- Fecha: 2026-11-10

## Contexto

La arquitectura exige que los servicios publiquen eventos mediante un outbox transaccional y que los consumidores apliquen efectos mediante inboxes duraderos e idempotentes. PostgreSQL es la fuente de verdad; Kafka, cuando se habilite, solo publica después del commit.

`bank-core` ya mantiene transacciones locales, journals equilibrados, claves de idempotencia y un inbox durable para abonos del receptor. La red mantiene el ciclo de vida de `transfer_instruction`, mensajes inter-redes, liquidación central y entregas al receptor. El transporte es al menos una vez y no existe una transacción ACID global entre banco y red.

Este ADR define el contrato que habilita una implementación posterior del outbox de `bank-core`. No implementa todavía la tabla, el dispatcher ni un nuevo endpoint público.

## Decisión

### Evento y versión

El evento bancario interoperable usa el nombre y la versión existentes en la red:

```text
message_type: TRANSFER_REQUESTED
schema_version: 1.0
```

`bank-core` no emite eventos de liquidación, abono o completitud como consecuencia de aceptar localmente una solicitud. La aceptación local no equivale a liquidación ni a crédito final.

### Identidad estable

Cada evento conserva estas identidades inmutables:

```text
event_id: UUID
aggregate_type: TRANSFER
aggregate_id: transfer_id
idempotency_key: clave estable de la operación
```

`transfer_id` y `event_id` no cambian durante reintentos, timeouts, reinicios ni recuperación administrativa. Un reintento nunca crea una nueva operación financiera.

El evento también conserva `correlation_id`, `causation_id`, `trace_id`, `occurred_at` en UTC y `version: 1`.

### Envelope y payload

El envelope canónico es:

```json
{
  "event_id": "UUID",
  "aggregate_type": "TRANSFER",
  "aggregate_id": "UUID",
  "event_name": "TRANSFER_REQUESTED",
  "schema_version": "1.0",
  "version": 1,
  "occurred_at": "2026-11-10T00:00:00Z",
  "correlation_id": "UUID",
  "causation_id": "UUID",
  "idempotency_key": "stable-operation-key",
  "trace_id": "synthetic-trace-id",
  "payload": {}
}
```

El payload usa importes enteros en centavos y conserva los campos ya soportados por la orquestación de la red:

```json
{
  "transfer_id": "UUID",
  "source_participant_id": "UUID",
  "destination_participant_id": "UUID",
  "amount_minor": 250,
  "currency": "USD",
  "route_kind": "INTER_NETWORK",
  "source_network_id": "UUID",
  "destination_network_id": "UUID"
}
```

La representación inmutable completa participa en un hash SHA-256. Cambiar participante, importe, moneda, transferencia, correlación, causalidad o payload usando la misma identidad es un conflicto y no produce efectos financieros.

### Estados del outbox

Una implementación futura usará estos estados:

```text
PENDING
IN_FLIGHT
PUBLISHED
DEAD_LETTER
```

- `PENDING`: el evento quedó confirmado junto con la transacción bancaria y puede ser enviado.
- `IN_FLIGHT`: un dispatcher posee un lease temporal para intentar el envío.
- `PUBLISHED`: la red aceptó el mismo evento y su misma identidad. No significa liquidación ni completitud.
- `DEAD_LETTER`: se agotaron los reintentos o existe un error contractual, de autenticación o de esquema que requiere intervención.

Un lease vencido devuelve `IN_FLIGHT` a `PENDING`. El outbox conserva `attempt_count`, `last_attempt_at`, `next_attempt_at` y `last_error`.

### Reintentos y resultados ambiguos

Los reintentos usan backoff exponencial con jitter y un límite configurable. Los valores iniciales sintéticos serán responsabilidad de la implementación y la configuración operacional; este ADR fija el comportamiento, no valores productivos.

- Timeout o pérdida de respuesta después de enviar: repetir exactamente el mismo evento.
- Respuesta exitosa o replay idempotente de la red: marcar `PUBLISHED`.
- Error de validación, autenticación, esquema o conflicto de payload: no reintentar ciegamente; mover a `DEAD_LETTER`.
- Replay administrativo: reusar el registro original, nunca crear otra transferencia.
- Resultado financiero incierto: conservar el identificador original y mantener el flujo en `UNKNOWN_PENDING_RECONCILIATION`; no emitir una operación sustituta.

### Límites de propiedad

`bank-core` es propietario de:

- sus cuentas, reservas, journals y ledger;
- la creación atómica del registro outbox junto con el cambio bancario local;
- los intentos de entrega, leases, reintentos y estado de transporte;
- la evidencia bancaria y el replay administrativo.

CholoNet es propietario de:

- `transfer_instruction` y su máquina de estados;
- los journals y cuentas de liquidación central;
- los registros de entrega al receptor;
- la reconciliación de red y el estado final de la transferencia.

El dispatcher de `bank-core` no escribe tablas de la red, no marca `SETTLED` o `COMPLETED` y no interpreta una respuesta HTTP como prueba de crédito al cliente.

### Idempotencia y restricciones

La futura tabla outbox tendrá unicidad al menos para `event_id` e `idempotency_key`. También conservará el hash del payload y rechazará cualquier reutilización con contenido o metadatos distintos.

Una repetición idéntica devuelve o conserva el resultado existente. Un conflicto no modifica saldos, journals, reservas ni eventos existentes.

### Fallos y recuperación

- Rollback de la transacción bancaria: no queda registro outbox y no se publica nada.
- Commit seguido de crash antes del dispatch: el registro `PENDING` se recupera al reiniciar.
- Crash después del envío y antes de persistir la respuesta: el mismo evento se reintenta y la red resuelve por idempotencia.
- Red no disponible: se mantiene `PENDING` o se recupera desde `IN_FLIGHT` con backoff.
- Reintentos agotados: `DEAD_LETTER` y revisión operativa, sin cambiar la operación financiera.
- Receptor no disponible después de la liquidación: la red conserva `CREDIT_PENDING`, reintenta la entrega y reconcilia; el inbox del receptor evita un segundo efecto financiero.

## Decisiones de integración T10

Las decisiones operativas para el primer slice de implementación son:

- Una nueva orden inter-red interna, separada de la ruta de transferencias locales, crea `TRANSFER_REQUESTED`.
- El adaptador genera un UUID propio e inmutable como `transfer_id`; no reutiliza el `transaction_id` local.
- `bank-core` publica hacia `/internal/v1/inter-network/transfers`, dejando la creación de la instrucción y del mensaje bajo ownership de CholoNet.
- El enlace banco→red usa un secreto HMAC sintético separado de `NETWORK_ADAPTER_SECRET`, con timestamp y request ID; la validación es fail-closed.

La ruta `/v1/transfers/internal` permanece local-only y no genera eventos inter-red.

## No decidido aquí

Este ADR no define:

- el fondeo real de cuentas de liquidación;
- la liquidación interbancaria;
- los valores productivos de backoff, jitter, lease o máximo de intentos;
- un broker obligatorio;
- un nuevo endpoint público;
- la semántica de reservas más allá de conservar la identidad y el estado que ya gobierna cada servicio.

Esas decisiones requieren sus propios ADR o configuración operacional antes de una implementación productiva. El ecosistema actual sigue siendo sintético y educativo.

## Consecuencias

La implementación posterior puede añadir un outbox sin cambiar la semántica de la red ni duplicar contratos de eventos. Los reintentos son seguros porque todas las capas conservan identidades y hashes estables. La separación de propietarios impide que un dispatcher bancario convierta la aceptación de transporte en liquidación o crédito final.

El contrato añade trabajo de implementación: migración, escritura transaccional, dispatcher con leases, reintentos, DLQ, métricas y pruebas de recuperación. Esa implementación queda fuera de este ADR.

## Referencias

- `adr/0005-outbox-inbox-and-foss-tooling.md`
- `adr/0010-bank-receiver-inbox.md`
- `spec.md`, secciones de estados, eventos y observabilidad
- `network/internal/orchestration/service.go`
- `network/internal/message/store.go`
- `network/internal/transfer/state.go`

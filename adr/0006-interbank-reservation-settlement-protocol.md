# ADR 0006: protocolo interbancario de reserva, liquidación y recuperación

- Estado: Aceptado como límite del protocolo de la fase 2; la implementación de la liquidación financiera sigue siendo incremental
- Fecha: 2026-11-09

## Contexto

Costa, Sierra y Oriente tienen bases de datos independientes. No existe una transacción ACID distribuida entre bancos y la red no debe escribir en tablas bancarias. Un timeout puede dejar el resultado incierto; reintentar con un identificador de negocio nuevo podría duplicar fondos.

## Decisión

Usar un protocolo de instrucciones duradero con cuatro fases de propiedad:

1. **Preparar en el emisor:** el emisor autentica la solicitud, valida la referencia de destino y crea una única reserva duradera indexada por `transfer_id` e `idempotency_key`. El emisor informa `FUNDS_RESERVED` solo después de confirmar su transacción local.
2. **Liquidación central:** CholoNet valida el estado del participante, el importe, la moneda y el estado único de la instrucción. Bloquea las dos cuentas de liquidación en orden canónico de participantes y confirma un único journal equilibrado de débito/crédito con un `transfer_id` único. Solo ese commit establece `SETTLED`.
3. **Aplicación en el receptor:** la red entrega al receptor un comando de abono duradero, al menos una vez. El inbox del receptor y la operación de cuenta forman una única transacción local. Un comando duplicado devuelve el resultado original sin un segundo abono.
4. **Finalización y conciliación del emisor:** después de confirmar la liquidación, el emisor captura la reserva y registra el débito final. Si falta alguna confirmación, se reintenta el mismo comando; nunca se crea un ID de transferencia nuevo. Un resultado contradictorio o no disponible pasa a `UNKNOWN_PENDING_RECONCILIATION` y bloquea la acción duplicada automática.

El contrato de estados es:

```text
CREATED -> VALIDATING -> FUNDS_RESERVED -> READY_FOR_SETTLEMENT
READY_FOR_SETTLEMENT -> SETTLED
SETTLED -> CREDIT_PENDING -> COMPLETED
READY_FOR_SETTLEMENT -> UNKNOWN_PENDING_RECONCILIATION
SETTLED -> UNKNOWN_PENDING_RECONCILIATION (only when evidence is incomplete)
UNKNOWN_PENDING_RECONCILIATION -> SETTLED | MANUAL_REVIEW
CREATED|VALIDATING|FUNDS_RESERVED -> REJECTED | EXPIRED (before settlement only)
SETTLED -> REFUNDED (derived from a new compensating operation)
```

## Invariantes y recuperación

- Un identificador de transferencia produce como máximo un journal central de liquidación.
- La reserva, la liquidación central y el abono al receptor son observables por separado.
- El journal central es inmutable; las devoluciones son asientos compensatorios.
- Una caída bancaria después de la reserva se recupera consultando el mismo ID de transferencia y repitiendo el mismo comando, no emitiendo una instrucción nueva.
- Una caída bancaria después de la liquidación central deja los fondos en un estado liquidado pero aún no abonado hasta que la aplicación en el receptor o una conciliación manual demuestre el resultado.
- La conciliación compara la reserva/débito del emisor, los asientos de liquidación de la red, el inbox/abono del receptor y los identificadores de correlación; no modifica el historial silenciosamente.

## Alternativas rechazadas

- Commit global de dos fases entre bases bancarias: no está disponible entre servicios independientes y aumenta el acoplamiento de bloqueos y fallos.
- Transporte exactamente una vez: no es una garantía de entrega realista; usar transporte al menos una vez más idempotencia duradera.
- HTTP 200 como prueba de liquidación: una confirmación no es el commit del ledger central.
- Repetir una transferencia con timeout usando un ID nuevo: arriesga una doble liquidación.

## Consecuencias

La red necesita una máquina de estados persistente para instrucciones, un ledger de liquidación, restricciones únicas, registros outbox/inbox y trabajos de conciliación. La fase 2 implementa inicialmente el límite de estados y la persistencia idempotente de instrucciones; los efectos de liquidación y los adaptadores bancarios siguen como slices revisados por separado.

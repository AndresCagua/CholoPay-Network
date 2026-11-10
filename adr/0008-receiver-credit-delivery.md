# ADR 0008: entrega y finalización del abono al receptor

- Estado: Aceptado para el primer slice de entrega
- Fecha: 2026-11-09

## Contexto

La liquidación central y el abono al receptor son commits diferentes, propiedad de sistemas distintos. CholoNet no puede escribir en el ledger de clientes de un banco. La entrega puede duplicarse, retrasarse, reordenarse o interrumpirse después de la liquidación.

## Decisión

Después de confirmar un journal central, CholoNet crea un registro de entrega duradero `CREDIT_PENDING` indexado por `transfer_id` y participante receptor. Un worker de outbox puede entregar el mismo comando repetidamente. El banco receptor aplica el comando en su propia transacción local y confirma con los mismos `transfer_id`, `delivery_id` y `event_id`; sus restricciones de inbox/idempotencia impiden un segundo efecto financiero.

CholoNet acepta una confirmación únicamente para una entrega existente cuya transferencia esté en `SETTLED` o `CREDIT_PENDING`. La primera confirmación válida marca la entrega como `APPLIED` y la transferencia como `COMPLETED` en una única transacción local. Las confirmaciones idénticas repetidas devuelven el resultado existente. Un evento o identidad de receptor en conflicto se rechaza y no modifica el estado. Las confirmaciones faltantes permanecen en `CREDIT_PENDING` y se reintentan con backoff acotado; nunca se declaran fallidas solo por timeout.

## Invariantes

- A settled transfer has at most one delivery record.
- Delivery transport is at-least-once; receiver financial effect is idempotent by durable identifiers.
- `COMPLETED` requires receiver evidence; `SETTLED` alone is insufficient.
- CholoNet never updates a bank customer ledger or invents receiver credit.
- A duplicate acknowledgement cannot create a second completion or journal.
- A refund is a new compensating operation and does not erase delivery history.

## Consecuencias

CholoNet es propietario de la intención de entrega, los intentos y la evidencia de confirmación. Cada banco deberá implementar posteriormente su propia transacción de aplicación en inbox y ledger. La conciliación compara la liquidación central, la entrega, el inbox del receptor y la evidencia del ledger bancario.

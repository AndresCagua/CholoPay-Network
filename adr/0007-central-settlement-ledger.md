# ADR 0007: ledger central de liquidación

- Estado: Aceptado para el primer slice de liquidación
- Fecha: 2026-11-09

## Contexto

Los tres ledgers bancarios son independientes. CholoNet necesita un ledger central que registre cambios de posición de participantes sin acceder a tablas bancarias. Una transferencia no debe considerarse liquidada porque se aceptó una solicitud HTTP.

## Decisión

Crear una cuenta de liquidación por participante y un journal inmutable de liquidación de partida doble. Una operación de liquidación:

1. Bloquea las cuentas de liquidación de origen y destino en orden canónico de UUID.
2. Verifica que la posición disponible de origen cubra el importe.
3. Crea exactamente un journal identificado por `transfer_id` y el hash de la solicitud.
4. Escribe un asiento de débito para el origen y uno de crédito para el destino en centavos de USD.
5. Actualiza ambas proyecciones de posición en la misma transacción de PostgreSQL.
6. Devuelve `SETTLED` solo después de confirmar la transacción.

Un `transfer_id` o una clave de idempotencia duplicados devuelven el journal original. Reutilizar la clave con un hash de solicitud diferente se rechaza. Los journals y asientos publicados son inmutables; las devoluciones son journals compensatorios separados. La red nunca escribe en el ledger de clientes de un banco.

## Invariantes

- Every journal has exactly two entries for this slice and their signed sum is zero.
- Source and destination positions conserve their aggregate sum.
- No source position becomes negative.
- One transfer ID creates at most one journal.
- A failed transaction leaves accounts and journal tables unchanged.
- `SETTLED` is derived only from the committed central journal.

## Consecuencias

El ledger central es autoritativo únicamente para las posiciones de liquidación entre participantes, no para los saldos de clientes. El abono al receptor sigue siendo una operación bancaria posterior, al menos una vez. Este slice no implementa fondeo de liquidez, reversiones, trabajos de conciliación ni adaptadores bancarios; requieren sus propias pruebas y evidencias.

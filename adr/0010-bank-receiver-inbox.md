# ADR 0010: inbox del receptor propiedad del banco y abono local

- Estado: Aceptado para el primer slice del adaptador
- Fecha: 2026-11-09

## Contexto

CholoNet no puede escribir en el ledger de clientes de un banco participante. El banco receptor debe aplicar localmente un abono liquidado y proporcionar evidencia duradera. La entrega es al menos una vez, por lo que se esperan comandos duplicados.

## Decisión

`bank-core` es propietario de un inbox duradero del receptor indexado por `delivery_id` y `event_id`. Un comando del receptor se acepta únicamente cuando la cuenta de destino pertenece a la instancia bancaria y la moneda/el importe coinciden con el payload inmutable del comando. La inserción en el inbox, el débito de la cuenta de compensación, el abono de la cuenta del cliente y el journal local equilibrado forman una única transacción de PostgreSQL.

Una entrega duplicada devuelve el resultado original de aplicación sin otro journal ni cambio de saldo. Un identificador de entrega asociado a un payload, cuenta receptora o evento diferente se rechaza. El banco devuelve una confirmación que contiene `delivery_id`, `event_id`, `transfer_id` y el ID del journal local resultante. CholoNet usa esa evidencia para marcar la entrega como aplicada.

El banco no marca un pago de cliente como completado a partir de una redirección HTTP o un comando duplicado. La cuenta de compensación es una cuenta sintética interna financiada por el adaptador de liquidación auditado por separado; este slice no implementa el fondeo interbancario.

## Consecuencias

Cada instancia bancaria tiene tablas independientes de inbox y ledger. El límite del adaptador es explícito y comprobable. La aplicación en el receptor sigue siendo atómica localmente aunque la red reintente el comando.

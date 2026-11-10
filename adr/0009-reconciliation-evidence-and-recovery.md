# ADR 0009: evidencia y recuperación de conciliación

- Estado: Aceptado para el primer slice de conciliación
- Fecha: 2026-11-09

## Contexto

La liquidación central, la entrega al receptor y la finalización bancaria son hechos duraderos separados. Un servicio puede fallar después de la liquidación, los mensajes pueden retrasarse y las proyecciones de estado pueden discrepar. La conciliación debe detectar diferencias sin reescribir el historial financiero.

## Decisión

Una ejecución de conciliación registra sus horas de inicio y fin y filas de diferencias inmutables. El ejecutor compara:

- every central journal to its delivery record;
- every delivery to its instruction state;
- every completion state to an applied delivery;
- delivery/settlement identifiers and participant ownership.

Findings use explicit codes such as `SETTLED_WITHOUT_DELIVERY`, `DELIVERY_PENDING`, `APPLIED_WITHOUT_COMPLETED_STATE`, and `COMPLETED_WITHOUT_APPLIED_DELIVERY`. Una ejecución limpia tiene cero filas de diferencias. El ejecutor es de solo lectura respecto del estado financiero y de transferencias; la remediación repite el mismo ID de entrega o abre una revisión manual y nunca edita un journal.

Las ejecuciones y los hallazgos se pueden consultar de forma idempotente por ID de ejecución. La evidencia de conciliación es append-only para la ejecución. Una ejecución posterior puede mostrar que una diferencia se resolvió, pero no elimina el hallazgo anterior.

## Consecuencias

La red puede distinguir trabajo faltante de pérdida financiera, preservar el historial de auditoría y recuperarse repitiendo una entrega existente. Los adaptadores bancarios deberán exportar posteriormente evidencia de inbox/ledger con los mismos identificadores de transferencia, entrega y evento.

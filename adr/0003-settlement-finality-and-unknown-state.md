# ADR 0003: Finalidad de liquidación y estado desconocido

- Estado: Aceptado para la fase 0; el detalle del protocolo se difiere a la fase 2
- Fecha: 2026-11-09

## Decisión

Solo un commit atómico del ledger central de liquidación establece `SETTLED`. Un timeout con resultado incierto se convierte en `UNKNOWN_PENDING_RECONCILIATION`; los clientes no deben crear un segundo ID de transferencia. `REFUNDED` es una operación compensatoria, nunca una eliminación o desliquidación.

## Consecuencias

El sistema necesita IDs de correlación duraderos, trabajos de conciliación, estados intermedios recuperables e incertidumbre explícita para la persona usuaria. La coreografía exacta de recuperación de reserva/commit requiere un ADR posterior antes de implementarse.

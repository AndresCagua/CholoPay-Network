# ADR 0002: Partida doble y ledgers autoritativos

- Estado: Aceptado para la fase 0
- Fecha: 2026-11-09

## Decisión

Los ledgers bancarios y de liquidación de la red usan journals inmutables de partida doble con centavos enteros de USD. PostgreSQL es autoritativo. Los saldos son proyecciones validadas; Redis y Kafka nunca son autoritativos. Las correcciones usan asientos compensatorios.

## Consecuencias

Toda operación financiera debe ser transaccional localmente, idempotente, auditable y conciliable. Esto prioriza la corrección frente a fingir que existe ACID distribuido.

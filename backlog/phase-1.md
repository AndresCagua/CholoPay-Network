# Vista previa del backlog de la fase 1

1. Definir el esquema bancario y la estrategia de migración.
2. Implementar el journal y las restricciones de asientos equilibrados.
3. Implementar reservas de cuentas con orden determinista de bloqueos.
4. Implementar transferencias internas con idempotencia y conflicto de hash del payload.
5. Agregar pruebas concurrentes para fondos insuficientes y solicitudes duplicadas.
6. Agregar tablas outbox/inbox sin tratar Kafka como fuente de verdad.
7. Documentar errores de API, permisos y semilla de datos sintéticos.
8. Ejecutar pruebas de integración enfocadas contra PostgreSQL efímero.

La liquidación interbancaria se difiere deliberadamente hasta que los ADR del protocolo de la fase 2 respondan las preguntas financieras de la sección 19 de `spec.md`.

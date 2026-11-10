# Backlog de la fase 2

| ID | Ticket | Depends on | Acceptance |
|---|---|---|---|
| P2-01 | Adoptar protocolo de reserva/liquidación/conciliación | Fase 1 | ADR 0006 y contrato de estados revisados |
| P2-02 | Crear esqueleto del servicio de red Go y compilación Docker | P2-01 | el endpoint de salud y la prueba del contenedor pasan |
| P2-03 | Persistir instrucciones y transiciones de transferencia | P2-02 | las transiciones inválidas y los ID duplicados se rechazan |
| P2-04 | Agregar esquema de participantes y cuentas de liquidación | P2-03 | sin acceso a tablas bancarias; migración aislada |
| P2-05 | Implementar journal central equilibrado de liquidación | P2-04 | un journal por transferencia y posiciones conservadas |
| P2-06 | Agregar contratos de adaptador bancario y outbox/inbox | P2-05 | replay al menos una vez con un único efecto financiero |
| P2-07 | Escenarios de fallos/conciliación | P2-06 | los estados de caída del emisor/receptor se recuperan sin duplicados |
| P2-08 | Integrar directorio CholoKey | P2-03 | resolución minimizada y restricciones de unicidad |

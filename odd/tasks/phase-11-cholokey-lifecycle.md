# Phase 11 — Ciclo de vida y portabilidad de CholoKey
Fuente: `spec.md`, secciones 23.2, 23.3, 26.2, 26.3, 26.8 y 27.
Scope: suspensión determinista, resolución durante transición y portabilidad sintética entre participantes; no integra redes reales ni implementa reclamaciones regulatorias.

## Specs

1. **S1 — Portabilidad sintética**
   > Portar una clave de Costa a Oriente, verificar que durante la transición no existan dos destinatarios activos y que luego resuelva únicamente a Oriente.

2. **S2 — Unicidad y transición**
   > Cada clave activa y única se vincula exactamente a una cuenta transaccional y a un participante en cada instante.

3. **S3 — Suspensión**
   > Suspender Banco Oriente y comprobar estados, reintentos, límites temporales y conciliación sin creación ni pérdida de fondos.

4. **S4 — Fuente de verdad**
   > cada banco sobre su ledger; cada facturador conserva la autoridad sobre su obligación
   > La red no accede a sus tablas internas.

5. **S5 — Compatibilidad**
   > Todo contrato o evento nuevo deberá incluir `schema_version`, identificador de correlación, `idempotency_key`, timestamp UTC y trazabilidad.

## Invariantes

- Una clave no puede tener dos destinos activos durante una transición.
- La resolución debe ser determinista por `network_id`, tipo y valor de clave.
- Reintentos de la misma operación son idempotentes; payloads distintos con la misma idempotency key se rechazan.
- Una suspensión bloquea nuevas operaciones aplicables, pero no borra historial ni crea movimientos financieros.
- La portabilidad no escribe directamente en ledgers bancarios.

## Tasks

- T1 (S1-S2): Definir estados de ciclo de vida y contrato de transición/portabilidad en el directorio. Route: inline after exploration. Commit: pending.
- T2 (S1-S5): Implementar transición atómica de propietario con idempotencia y resolución durante suspensión/transición. Route: inline. Commit: pending.
- T3 (S1-S5): Añadir regresiones Docker de doble destino, replay, conflicto y suspensión. Route: inline. Commit: pending.
- T4 (S1-S5): Actualizar README, contrato y evidencia. Route: inline. Commit: pending.

## Log

L1. Usuario: "bueno continua con los specs"
L2. Se continúa después de la gobernanza básica con el ciclo de vida de CholoKey requerido por la aceptación v1.1: suspensión determinista y portabilidad sintética.

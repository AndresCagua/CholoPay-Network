# Backlog de la fase 0

| ID | Ticket | Depende de | Aceptación / prueba |
|---|---|---|---|
| P0-01 | Fijar límites de servicios y modelo de confianza | — | ADR 0001, diagramas C4, sin acceso cruzado a BD bancarias |
| P0-02 | Definir invariantes del ledger y vocabulario de liquidación | P0-01 | ADRs 0002–0003, máquina de estados, lista de invariantes |
| P0-03 | Definir el alta propiedad del banco y el límite de CholoKey | P0-01 | ADR 0004 y modelo de amenazas |
| P0-04 | Preparar inventario de licencias FOSS y política de código fuente | — | SPDX inventory, no paid runtime dependency |
| P0-05 | Crear perfil Compose local de PostgreSQL aislado | P0-01 | Four DB services, separate volumes, readiness checks |
| P0-06 | Agregar preflight obligatorio de puertos del host y prueba smoke | P0-05 | Occupied port prevents startup; health checks pass when available |
| P0-07 | Agregar perfil opcional de Kafka y guía de outbox | P0-02 | El perfil predeterminado funciona sin Kafka; la documentación explica la fuente de verdad |
| P0-08 | Agregar checks estructurales de CI | P0-04–P0-06 | Los checks de YAML/OpenAPI/Compose se ejecutan localmente y en CI |
| P0-09 | Publicar alta y runbook | P0-05–P0-08 | Clean Linux checkout can render, start, smoke, stop |
| P0-10 | Revisar evidencia de la fase 0 y autorizar el slice vertical | P0-01–P0-09 | Toda la evidencia de aceptación registrada; decisiones financieras pendientes enumeradas |

## Vista previa de la fase 1

Implementar un núcleo bancario Spring Boot parametrizado con journal inmutable de partida doble, reservas, idempotencia de transferencias internas, migraciones y pruebas concurrentes de invariantes antes de agregar la red o las interfaces.

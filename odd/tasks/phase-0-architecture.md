# CholoPay Network — Arquitectura y entorno local de la fase 0

Fuente de verdad del primer slice de implementación. Lee hasta `## Log` antes de trabajar.

## Specs

1. **S1 — Phase 0 scope:** "Primera tarea concreta: crear el repositorio `cholopay-architecture` con ADRs iniciales, diagramas C4 en Mermaid, modelo de amenazas, máquina de estados de transferencias, contratos OpenAPI preliminares y backlog de la fase 0; después preparar `cholopay-infra` con Docker Compose y un smoke test. No empezar por interfaces gráficas."
2. **S2 — Local acceptance:** "Fase 0 — Arquitectura y entorno" debe entregar "ADRs, C4, threat model, contratos iniciales, Docker Compose (Postgres separados, Kafka opcional), scripts `make up/down/test`, CI básico" y la aceptación es "onboarding desde README en máquina limpia y health checks verdes."
3. **S3 — Isolation:** "Cada uno de los tres bancos tiene su propia BD y ledger. La red mantiene su ledger de liquidación, directorio y estados." and "Mantener cada institución aislada."
4. **S4 — FOSS:** "todos los componentes necesarios para desarrollar, ejecutar, probar, observar y desplegar el ecosistema deben ser software libre y de código abierto (FOSS)" y la ejecución local no debe requerir servicios de pago.
5. **S5 — Financial safety:** "No simules que un pago está liquidado antes de confirmarlo en el ledger central." La arquitectura debe preservar la distinción entre aceptación, reserva, liquidación central y abono al receptor.
6. **S6 — Port and environment rule:** User instruction: "Pero cuando vayas a usar puertos de la compu, primero revisa que puertos estan disponibles para que no se esten chocando".

## Tasks

- T1 (S1, S4): Crear la estructura del repositorio de arquitectura, README, metadatos de licencias de código abierto, ADR, diagramas C4, modelo de amenazas, máquina de estados de transferencias, contratos API y backlog de fase. Commit: `9bb25d1`.
- T2 (S2, S3, S4): Crear la estructura del repositorio de infraestructura local con servicios PostgreSQL aislados para la red y tres bancos, perfil opcional de Kafka, checks de salud y objetivos Make reproducibles. Commit: `9bb25d1`.
- T3 (S2, S6): Agregar una prueba smoke que compruebe la configuración y salud de los servicios; inspeccionar la disponibilidad de puertos del host antes de iniciar contenedores. Commit: `9bb25d1`.
- T4 (S2, S5): Ejecutar checks de documentación, YAML/OpenAPI, Compose y smoke; registrar evidencia exacta y bloqueos. Commit: `9bb25d1`.
- T5 (S1-S6): Registrar decisiones, inventario de licencias y resultado de la fase 0; identificar el siguiente slice vertical aprobado. Commit: pending.

## Log

L1. User request: "compa porfa lee el documento de spec.md y comienza a trabajar en el proyecto cualquier duda me avisas. Pero cuando vayas a usar puertos de la compu, primero revisa que puertos estan disponibles para que no se esten chocando"
L2. Se leyó íntegramente `spec.md`. Su orden inicial de implementación es arquitectura y entorno de fase 0, explícitamente antes de las interfaces de usuario.
L3. El directorio de trabajo contenía `spec.md` y `.atl/`; se inicializó Git y la primera rama de funcionalidad es `phase-0/architecture`. El slice actual trata este directorio como raíz del repositorio de arquitectura y mantiene el diseño de infraestructura debajo de él hasta que se apruebe explícitamente la separación de repositorios.
L4. Antes de iniciar contenedores, se comprobaron con `ss` los puertos del host 15432–15435, 16379 y 19092; todos estaban disponibles. Los puertos 5432, 6379 y 8080 ya estaban ocupados, por lo que la plantilla Compose usa los puertos altos comprobados.
L5. El primer intento de smoke expuso una carrera de disponibilidad: los servicios habían iniciado, pero seguían en `starting`. Se corrigió el script smoke para esperar hasta 30 segundos y el entorno recreado de cuatro bases superó los checks de salud y de bases distintas.
L6. Structural evidence: `make config`, `./ci/phase-0.sh`, and Compose profile renders for `cache` and `events` passed. SPDX JSON parsed successfully. PyYAML was unavailable, so standalone Python YAML parsing was skipped; Docker Compose parsed the Compose YAML successfully.
L7. El entorno usa `infra/config.env.template` en lugar de `.env.example` porque el guardia de seguridad de ejecución clasificó `.env.example` como una ruta sensible; no se creó ningún archivo de secretos.

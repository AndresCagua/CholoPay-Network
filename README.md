# CholoPay Network

CholoPay Network es un ecosistema educativo y completamente simulado de pagos interoperables. No mueve dinero real, no se conecta con Pix/SPI/DICT, no representa a un banco ni a una autoridad pública y no almacena datos personales reales.

Este repositorio contiene la arquitectura de la fase 0 y el primer slice de núcleo bancario de la fase 1. Las interfaces de usuario están deliberadamente fuera de alcance hasta aprobar el slice vertical.

## Regla de uso exclusivo de Docker

El proyecto está diseñado para que un checkout nuevo necesite únicamente **Docker Engine y el complemento Compose**. No instales Java, Maven, Gradle, Node, Go, PostgreSQL, Redis, Kafka, Python ni clientes de bases de datos en el host para compilar, probar o ejecutar CholoPay. Las imágenes de compilación y prueba contienen sus propias cadenas de herramientas.

La pila Compose predeterminada no vincula ninguna base de datos, Redis, Kafka ni puerto de aplicación al host. Los servicios se comunican en una red Docker privada, por lo que el flujo normal no puede colisionar con puertos del host:

```bash
docker compose --env-file infra/config.env.template -f infra/compose.yaml config
docker compose --env-file infra/config.env.template -f infra/compose.yaml up -d network-db costa-db sierra-db oriente-db bank-core
docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile test run --rm bank-core-test
./infra/scripts/smoke.sh
docker compose --env-file infra/config.env.template -f infra/compose.yaml down
```

`make` es solamente un envoltorio opcional de conveniencia; no es un prerrequisito del proyecto. Usa `docker compose` directamente en un checkout exclusivo de Docker. La compilación de la imagen usa varias etapas y ejecuta Maven dentro de `bank-core/Dockerfile`.

## Límites

- `cholopay-architecture`: este repositorio: ADR, diagramas, contratos, modelo de amenazas, runbooks y backlog.
- `cholopay-infra`: entorno FOSS local; el primer diseño Compose está en `infra/` mientras se completa la separación de repositorios.
- `cholopay-network`: red central en Go y ledger de liquidación (repositorio futuro).
- `cholopay-bank-core`: núcleo bancario parametrizado en Java/Spring Boot, desplegado de forma independiente para Costa, Sierra y Oriente (repositorio futuro).

Las tres instancias bancarias tienen bases PostgreSQL, credenciales, claves y ledgers separados. La red nunca escribe directamente en una base bancaria. Kafka es un transporte opcional de eventos; los ledgers PostgreSQL siguen siendo la fuente de verdad.

## Documentación

- [Visión y límites](docs/vision.md)
- [Contexto C4](docs/c4-context.md) y [contenedores C4](docs/c4-containers.md)
- [Protocolo de transferencias y máquina de estados](docs/transfer-protocol.md)
- [Invariantes del ledger](docs/ledger-invariants.md)
- [Modelo de amenazas](docs/security-threat-model.md)
- [Estrategia de pruebas](docs/testing-strategy.md)
- [Inventario FOSS](sbom/dependencies.spdx.json)
- [Backlog de la fase 0](backlog/phase-0.md)
- [Núcleo bancario de la fase 1](bank-core/README.md)
- [Máquina de estados interbancaria](docs/interbank-state-machine.md)
- [Esqueleto de red de la fase 2](network/README.md)
- [Backlog de la fase 2](backlog/phase-2.md)

## Licencia

El código está bajo Apache-2.0 salvo que un repositorio futuro indique lo contrario. La documentación está bajo CC BY 4.0. Todos los datos de demostración deben ser sintéticos. Consulta `LICENSE`, `NOTICE` y `SECURITY.md`.

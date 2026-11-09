# CholoPay Network

CholoPay Network is an educational, fully simulated interoperable-payments ecosystem. It does not move real money, connect to Pix/SPI/DICT, represent a bank or public authority, or store real personal data.

This repository contains Phase 0 architecture and the first Phase 1 bank-core slice. User interfaces are intentionally out of scope until the vertical slice is approved.

## Docker-only rule

The project is designed so a fresh checkout needs **Docker Engine and the Compose plugin only**. Do not install Java, Maven, Gradle, Node, Go, PostgreSQL, Redis, Kafka, Python, or database clients on the host to build, test, or run CholoPay. Build and test images contain their own toolchains.

The default Compose stack binds no database, Redis, Kafka, or application port to the host. Services communicate on a private Docker network, so the normal workflow cannot collide with host ports:

```bash
docker compose --env-file infra/config.env.template -f infra/compose.yaml config
docker compose --env-file infra/config.env.template -f infra/compose.yaml up -d network-db costa-db sierra-db oriente-db bank-core
docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile test run --rm bank-core-test
./infra/scripts/smoke.sh
docker compose --env-file infra/config.env.template -f infra/compose.yaml down
```

`make` is an optional convenience wrapper only; it is not a project prerequisite. Use `docker compose` directly for a Docker-only checkout. The image build is multi-stage and runs Maven inside `bank-core/Dockerfile`.

## Boundaries

- `cholopay-architecture`: this repository: ADRs, diagrams, contracts, threat model, runbooks, and backlog.
- `cholopay-infra`: local FOSS environment; the first Compose layout is under `infra/` while repository separation is pending.
- `cholopay-network`: Go central network and settlement ledger (future repository).
- `cholopay-bank-core`: parametrized Java/Spring Boot bank core, deployed independently for Costa, Sierra, and Oriente (future repository).

The three bank instances have separate PostgreSQL databases, credentials, keys, and ledgers. The network never writes directly to a bank database. Kafka is optional event transport; PostgreSQL ledgers remain the source of truth.

## Documentation

- [Vision and boundaries](docs/vision.md)
- [C4 context](docs/c4-context.md) and [C4 containers](docs/c4-containers.md)
- [Transfer protocol and state machine](docs/transfer-protocol.md)
- [Ledger invariants](docs/ledger-invariants.md)
- [Threat model](docs/security-threat-model.md)
- [Testing strategy](docs/testing-strategy.md)
- [FOSS inventory](sbom/dependencies.spdx.json)
- [Phase 0 backlog](backlog/phase-0.md)
- [Phase 1 bank-core](bank-core/README.md)
- [Interbank state machine](docs/interbank-state-machine.md)
- [Phase 2 network skeleton](network/README.md)
- [Phase 2 backlog](backlog/phase-2.md)

## License

Code is Apache-2.0 unless a future repository states otherwise. Documentation is CC BY 4.0. All demo data must be synthetic. See `LICENSE`, `NOTICE`, and `SECURITY.md`.

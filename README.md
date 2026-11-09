# CholoPay Network

CholoPay Network is an educational, fully simulated interoperable-payments ecosystem. It does not move real money, connect to Pix/SPI/DICT, represent a bank or public authority, or store real personal data.

This repository currently contains Phase 0 architecture and local-environment specifications. User interfaces are intentionally out of scope until the vertical slice is approved.

## Repositories and boundaries

- `cholopay-architecture`: this repository: ADRs, diagrams, contracts, threat model, runbooks, and backlog.
- `cholopay-infra`: local FOSS environment; the first Compose layout is under `infra/` while repository separation is pending.
- `cholopay-network`: Go central network and settlement ledger (future repository).
- `cholopay-bank-core`: parametrized Java/Spring Boot bank core, deployed independently for Costa, Sierra, and Oriente (future repository).

The three bank instances have separate PostgreSQL databases, credentials, keys, and ledgers. The network never writes directly to a bank database. Kafka is optional event transport; PostgreSQL ledgers remain the source of truth.

## Phase 0 quick start

Prerequisites: Linux, Git, Docker Engine or Podman, Compose plugin, GNU Make, and a free host-port range. Before starting containers, inspect availability:

```bash
./infra/scripts/check-ports.sh
make config
make up
make smoke
make down
```

`check-ports.sh` is mandatory before `make up`; it checks the configured host ports and fails without starting services when one is occupied. The default ports are documented in `infra/.env.example` and can be changed without editing Compose.

## Documentation

- [Vision and boundaries](docs/vision.md)
- [C4 context](docs/c4-context.md) and [C4 containers](docs/c4-containers.md)
- [Transfer protocol and state machine](docs/transfer-protocol.md)
- [Ledger invariants](docs/ledger-invariants.md)
- [Threat model](docs/security-threat-model.md)
- [Testing strategy](docs/testing-strategy.md)
- [FOSS inventory](sbom/dependencies.spdx.json)
- [Phase 0 backlog](backlog/phase-0.md)

## License

Code is Apache-2.0 unless a future repository states otherwise. Documentation is CC BY 4.0. All demo data must be synthetic. See `LICENSE`, `NOTICE`, and `SECURITY.md`.

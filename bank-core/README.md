# CholoPay Bank Core — Phase 1

Parametrized Spring Boot bank core for Banco Costa, Banco Sierra, and Banco Oriente. This first slice implements synthetic accounts, integer-cent balances, balanced immutable journals, reservations, and idempotent internal transfers. It does not implement interbank settlement.

## Docker-only build and test

The host needs Docker Engine and the Compose plugin only. Java, Maven, Gradle, and PostgreSQL are supplied by the project containers and must not be installed or invoked on the host.

From the repository root:

```bash
docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile test run --rm bank-core-test
```

The test image is built from `bank-core/Dockerfile` using Maven and JDK 21 inside the container. The runtime image contains only the JRE and the packaged application.

To run the service with the Costa database entirely inside Compose:

```bash
docker compose --env-file infra/config.env.template -f infra/compose.yaml up -d network-db costa-db bank-core
```

The service is reachable only by other Compose services at `http://bank-core:8080`; no host port is published by default. If a local development proxy is added later, its host port must be checked before binding.

## Financial behavior

- Amounts are positive integer USD cents.
- A journal must contain balanced debit/credit entries.
- Published journals are immutable in the domain model.
- Internal transfers reserve, capture, debit, and credit within one service critical section.
- Reusing an idempotency key with the same amount returns the original transaction; a different amount is rejected.
- A successful internal transfer is not an interbank settlement.

The PostgreSQL migration is under `src/main/resources/db/migration`. The next implementation slice replaces the in-memory service repository with transactional JDBC persistence and public error mapping before claiming production-like behavior.

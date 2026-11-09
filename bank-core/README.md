# CholoPay Bank Core — Phase 1

Parametrized Spring Boot bank core for Banco Costa, Banco Sierra, and Banco Oriente. This first slice implements synthetic accounts, integer-cent balances, balanced immutable journals, reservations, and idempotent internal transfers. It does not implement interbank settlement.

## Build

A Maven wrapper is not included yet. Use Maven 3.9+ with JDK 21, or run the project with an equivalent FOSS Maven container:

```bash
mvn test
mvn spring-boot:run -Dspring-boot.run.arguments=--server.port=8081
```

The default database is Banco Costa on the Phase 0 port `15433`; override `DB_URL`, `DB_USER`, `DB_PASSWORD`, `PARTICIPANT_ID`, and `SERVER_PORT` per bank instance. Do not use real credentials or identities.

## Financial behavior

- Amounts are positive integer USD cents.
- A journal must contain balanced debit/credit entries.
- Published journals are immutable in the domain model.
- Internal transfers reserve, capture, debit, and credit within one service critical section.
- Reusing an idempotency key with the same amount returns the original transaction; a different amount is rejected.
- A successful internal transfer is not an interbank settlement.

The PostgreSQL migration is under `src/main/resources/db/migration`. The next slice must replace the in-memory service repository with transactional persistence and public error mapping before claiming production-like behavior.

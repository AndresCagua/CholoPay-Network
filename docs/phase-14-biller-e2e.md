# Phase 14 — aceptación E2E de obligaciones

La aceptación T7 se ejecuta solo dentro de Docker y atraviesa `transit -> gateway -> bank-core -> utilities/transit` mediante HTTP. La matrícula se crea con `POST transit/v1/registrations` y produce una obligación sintética estable; la multa se crea con `POST transit/v1/fines`. El runner consulta la proyección del gateway, paga las seis clases por el endpoint participant-scoped de bank-core, repite la misma clave y verifica conflicto con una carga distinta.

```sh
docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile acceptance up --build --abort-on-container-exit --exit-code-from obligations-e2e obligations-e2e
```

Evidencia esperada: `T7 PASS: six synthetic obligations queried and paid through gateway -> bank-core -> facturers`. La evidencia se limita a estados públicos sintéticos (`APPLIED` y `PAID`); no demuestra liquidación bancaria real, balances, asientos ni certificados oficiales.

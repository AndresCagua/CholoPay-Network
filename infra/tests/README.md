# Pruebas de aceptación HTTP

` t7_obligations_e2e.py` es un runner exclusivo de Docker. Usa únicamente alias Compose (`transit`, `utilities`, `gateway`, `bank-core`) y sus APIs HTTP públicas; no abre conexiones PostgreSQL, no usa contenedores de base de datos, fixtures de filesystem ni runtimes del host.

## Ejecución

Desde la raíz:

```sh
docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile acceptance up --build --abort-on-container-exit --exit-code-from obligations-e2e obligations-e2e
```

La prueba crea matrícula y multa sintéticas por HTTP, exige las seis clases en la proyección participant-scoped, repite cada pago con la misma clave y comprueba conflicto con una carga distinta. `APPLIED`/`PAID` son estados de los facturadores sintéticos: no implican liquidación bancaria real, movimiento de ledger ni constancia oficial.

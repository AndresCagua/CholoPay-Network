# API sintética de autoridad de tránsito

`cholopay-transit` es una API exclusiva para Docker y local al proceso para flujos educativos de CholoPay. Usa registros, multas, obligaciones, estados y constancias sintéticos. **No es una autoridad real, no emite certificados legales y no debe usarse con datos personales o de vehículos reales.** No se conecta a bancos, cuentas, SPI/DICT ni infraestructura externa.

## Uso exclusivo para Docker

```sh
docker compose -f compose.yaml --profile test run --rm transit-test
docker compose -f compose.yaml up --build transit
docker compose -f compose.yaml down
```

De forma predeterminada no se expone ningún puerto al host. Los datos son locales al proceso y desaparecen cuando se detiene el contenedor. Para conectarte desde otro contenedor de la misma red Compose, usa `http://transit:8082`.

## Superficie HTTP

- `GET /health/live`
- `POST /v1/registrations` — `{ "plate": "SYN-001", "vehicle_type": "synthetic-car" }`
- `GET /v1/registrations/{id-or-plate}`
- `POST /v1/fines` — crea una multa sintética y su obligación impagada.
- `GET /v1/fines/{id}`
- `GET /v1/obligations/{id}` y `GET /v1/obligations?registration_id=...&status=unpaid`
- `GET /v1/status?registration_id=...`
- `POST /v1/constancias` — crea una constancia de estado sintética; no es un certificado oficial.
- `POST /v1/payment-intents` — requiere `Idempotency-Key`; solo crea una intención pendiente.
- `GET /v1/payment-intents/{id}`

Las intenciones de pago son un límite explícito de integración. Repetir la misma clave y carga devuelve la misma intención; una discrepancia de carga devuelve `409`. El servicio nunca adeuda una cuenta, cambia una obligación a pagada ni escribe un diario. Todo el estado está en memoria.

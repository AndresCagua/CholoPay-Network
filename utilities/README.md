# CholoPay synthetic utilities biller

API sintética exclusiva para Docker de la empresa ficticia de servicios públicos múltiples (electricidad, agua, internet y telefonía móvil). Solo usa la biblioteca estándar de Python y almacena su pequeño conjunto de datos de prueba en la memoria del proceso.

## Boundary

Este servicio es un simulador, no un banco ni un procesador de pagos. Solo usa identificadores y cantidades sintéticos. Nunca se conecta a SPI, DICT, un banco, una cartera, una cuenta de cliente ni infraestructura externa. Nunca adeuda una cuenta, liquida dinero ni escribe un diario. `POST /api/v1/payment-intents` solo crea una **intención pendiente** idempotente para un límite de integración futuro.

La configuración de Docker expone el puerto 8080 únicamente a la red de Docker; de forma predeterminada no tiene un mapeo `ports` al host. Para llamarla desde un host local, usa un mapeo temporal de puerto autorizado explícitamente en lugar de cambiar el archivo compose predeterminado.

## Ejecución

```sh
docker compose up --build
```

La API está disponible para otros contenedores en `http://utilities:8080`.

## Contrato

La API nueva de obligaciones usa `/v1` (las rutas históricas `/api/v1` se conservan):

- `GET /v1/services/catalog` — catálogo de electricidad, agua potable, internet fijo y telefonía móvil, todo en USD.
- `GET /v1/contracts/{contractRef}/bills` — consulta las facturas del contrato.
- `GET /v1/bills/{billId}` — recupera una factura.
- `POST /v1/bills/{billId}/payment-intents` — crea una intención con `Idempotency-Key` y JSON `{ "amount": "...", "currency": "USD" }`.
- `GET /v1/bills/{billId}/payment-status` — consulta el estado de aplicación.
- `GET /api/v1/contracts`, `GET /api/v1/contracts/{contract_id}`, `GET /api/v1/invoices?...` y `GET /api/v1/invoices/{invoice_id}` — contratos y facturas históricos compatibles.
- `POST /api/v1/payment-intents` — forma histórica, con JSON `{ "invoice_id": "...", "amount": "...", "currency": "USD" }`.
- `POST /api/v1/internal/payment-events` — aplica un evento sintético `payment.settled` firmado y protegido contra repetición por identificador de evento. Este endpoint no adeuda cuentas ni escribe un diario.

Una clave de idempotencia repetida y una solicitud idéntica devuelven la intención original. Para las rutas nuevas, dos claves distintas no pueden reservar concurrentemente la misma factura. Reutilizar una clave con una solicitud diferente devuelve `409`. Toda intención informa explícitamente de `debit_performed: false` y `ledger_entry_created: false`; los fondos solo se consideran aplicados tras un evento firmado.

## Pruebas de comportamiento HTTP

Las pruebas usan la interfaz HTTP pública y el mismo controlador de la biblioteca estándar que usa el contenedor:

```sh
docker compose --profile test run --rm utilities-test
```

La misma prueba pública puede ejecutarse localmente sin Docker para depuración:

```sh
PYTHONPATH=utilities python3 -m unittest discover -s utilities/tests -v
```

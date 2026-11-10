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

## Contract

- `GET /health` — respuesta sintética de salud.
- `GET /api/v1/contracts` — lista contratos sintéticos de servicios.
- `GET /api/v1/contracts/{contract_id}` — recupera un contrato.
- `GET /api/v1/invoices?contract_id={contract_id}` — consulta facturas, opcionalmente por contrato.
- `GET /api/v1/invoices/{invoice_id}` — recupera una factura.
- `GET /api/v1/invoices/{invoice_id}/collection-status` — consulta el estado sintético de cobro.
- `GET /api/v1/invoices/{invoice_id}/payment-status` — consulta el estado de aplicación de una liquidación sintética.
- `POST /api/v1/payment-intents` — crea una intención con `Idempotency-Key` y JSON `{ "invoice_id": "...", "amount": "...", "currency": "USD" }`.
- `POST /api/v1/internal/payment-events` — aplica un evento sintético `payment.settled` firmado con protección contra repetición por identificador de evento. Este endpoint no adeuda cuentas ni escribe un diario.

Una clave de idempotencia repetida y una solicitud idéntica devuelven la intención original. Reutilizar la clave con una solicitud diferente devuelve `409`. Toda intención nueva permanece en `pending` e informa explícitamente de `debit_performed: false` y `ledger_entry_created: false`.

## Pruebas de comportamiento HTTP

Las pruebas usan la interfaz HTTP pública y el mismo controlador de la biblioteca estándar que usa el contenedor:

```sh
PYTHONPATH=utilities python -m unittest discover -s utilities/tests -v
```

# Commerce sandbox

Superficie comercial mínima y sintética para el entorno aislado de CholoPay. Es deliberadamente independiente de la pasarela: el servicio solo usa memoria local al proceso, datos de catálogo iniciales y unidades menores sintéticas denominadas en USD. No procesa dinero, llama a servicios externos ni escribe un diario.

## Uso exclusivo para Docker

Desde este directorio:

```sh
docker compose -f compose.yaml up --build commerce
docker compose -f compose.yaml --profile test run --rm commerce-test
docker compose -f compose.yaml down
```

La API está disponible en `http://localhost:18081` de forma predeterminada. Los datos desaparecen cuando se detiene el contenedor.

## API observable

- `GET /health/live`
- `GET /v1/catalog`
- `POST /v1/carts` — crea un carrito vacío.
- `GET /v1/carts/{cart_id}`
- `POST /v1/carts/{cart_id}/items` — JSON `{ "sku": "demo-coffee", "quantity": 2 }`.
- `POST /v1/orders` — JSON `{ "cart_id": "..." }` y la cabecera `Idempotency-Key` requerida.
- `POST /v1/checkout` — JSON `{ "cart_id": "..." }` y la `Idempotency-Key` requerida; crea un pedido y una intención de pago de la pasarela, devolviendo `payment_status: "pending"` or `"succeeded"`.
- `POST /v1/checkouts/{checkout_id}/confirm` — confirmación sintética en el servidor con la `Idempotency-Key` requerida; una redirección del navegador nunca se considera prueba de pago.
- `GET /v1/orders/{order_id}`
- `GET /v1/inventory`

Crear un pedido comprueba y decrementa atómicamente el inventario sintético. Repetir un pedido con la misma clave de idempotencia devuelve el pedido original sin un segundo decremento. El checkout usa `GATEWAY_URL` plus `GATEWAY_API_KEY` para la pasarela HTTP, o el adaptador explícitamente denominado `GATEWAY_ADAPTER=synthetic` adapter (el valor predeterminado cuando no se configura una URL). Solo expone estados de pago pending/succeeded; no tiene una ruta de éxito basada en redirección. Reiniciar el proceso restablece todo el estado.

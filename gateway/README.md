# CholoPay gateway sandbox

Pasarela pública de Node.js/TypeScript exclusiva para Docker (`cholopay-gateway`). Proporciona claves API sintéticas con alcances y un flujo de intenciones de pago en memoria, únicamente para el comportamiento del entorno aislado.

## Ejecución

Construye y ejecuta la imagen desde la raíz del repositorio:

```sh
docker build -f gateway/Dockerfile -t cholopay-gateway ./gateway
docker run --rm -p 8090:8090 cholopay-gateway
```

La compilación de Docker ejecuta la suite de pruebas focalizadas antes de producir la etapa de ejecución. No se requiere instalar Node.js en el host.

## Autenticación sintética

Envía `X-Api-Key` en cada ruta protegida. Estas claves, exclusivas de desarrollo, son deliberadamente sintéticas y no son credenciales:

| Key | Scopes |
| --- | --- |
| `sk_sandbox_merchant_rw` | `payment:read`, `payment:write`, `webhook:write` |
| `sk_sandbox_merchant_ro` | `payment:read` |

Las claves ausentes o desconocidas devuelven `401`; una clave conocida sin el alcance de la ruta devuelve `403`. Las claves se conservan en memoria y no deben usarse fuera del entorno aislado.

## Intenciones de pago sintéticas

`POST /v1/payment-intents` (`payment:write`) crea en memoria una intención a partir de `amount_minor`, la `currency` en mayúsculas de tres letras, `merchant_order_id` y `description` opcional. Las intenciones nuevas comienzan en `requires_confirmation` y reciben el identificador sintético `pi_sandbox_...`.

Cada solicitud de creación y confirmación requiere una cabecera `Idempotency-Key`. Repetir la misma clave con la misma carga JSON reproduce la respuesta original; reutilizarla con una carga diferente devuelve `409 idempotency_key_conflict`.

- `GET /v1/payment-intents/{id}` (`payment:read`) consulta el estado sintético actual.
- `POST /v1/payment-intents/{id}/confirm` (`payment:write`) transiciona de `requires_confirmation` a `succeeded`.
- `POST /v1/refunds` (`payment:write`) acepta `{ "payment_intent_id": "pi_sandbox_...", "amount_minor": 2500, "currency": "USD"` }` solo después de que la intención esté en `succeeded`. Este entorno aislado acotado admite un reembolso completo por intención, devuelve `201` con `status: succeeded`, y nunca escribe un diario. Reutilizar una clave de idempotencia con otra carga devuelve `409`; un segundo reembolso para la misma intención devuelve `409 refund_conflict`.

## Proyección y flujo de obligaciones sintéticas

`GET /v1/participants/{participantId}/obligations` (`payment:read`) agrega obligaciones de utilities y tránsito mediante sus APIs HTTP internas. La pasarela no accede a bases de datos; devuelve únicamente una proyección delimitada por el participante solicitado y responde `502` si algún facturador no está disponible.

`POST /v1/bill-payments` acepta obligaciones de utilities (`electricity`, `water`, `internet`, `mobile`) o tránsito (`biller: transit`, `obligation_id`) mediante URLs internas configurables. Conserva idempotencia, aplicación eventual, firma HMAC y la frontera sin escritura de ledger.

## Flujo de facturación eléctrica de Banco Costa

`POST /v1/bill-payments` es el corte vertical público acotado. Requiere la clave de escritura del comerciante, una `Idempotency-Key` y `{ "participant_id": "BANCO_COSTA", "invoice_id": "invoice-elec-2025-001", "service": "electricity", "amount": "42.50", "currency": "USD" }`. La pasarela valida el límite explícito del llamador de Banco Costa, solicita a CholoServicios una intención pendiente y envía un evento `payment.settled` firmado para su aplicación idempotente. `201 APPLIED` significa que terminó la aplicación del facturador sintético; `202 PENDING_BILLER` o `PENDING_APPLICATION` no es prueba de liquidación. `POST /v1/bill-payments/{id}/apply` reintenta la aplicación eventual y `GET /v1/bill-payments/{id}` lee el estado.

Este es tráfico sintético exclusivo para Docker: no intervienen dinero real, infraestructura externa ni escrituras en los diarios de la pasarela o de servicios públicos.

## Webhooks sintéticos

`POST /v1/webhook-endpoints` (`webhook:write`) registra en memoria un endpoint a partir de una URL HTTP(S) sintética y devuelve un secreto de firma exclusivo del entorno aislado `whsec_...`. No se realiza ninguna entrega de red. El secreto devuelto firma el cuerpo exacto de la solicitud con HMAC-SHA256 usando el formato `${timestamp}.${raw_body}`; envíalo como `Webhook-Signature: t=<unix_seconds>,v1=<hex_digest>` a `POST /v1/webhook-endpoints/{id}/events`. Las marcas de tiempo de más de cinco minutos, las firmas malformadas y las discrepancias devuelven `401`; un par de marca de tiempo y firma ya aceptado devuelve `409 replay_detected`.

Para simular entregas locales, `POST /v1/webhook-endpoints/{id}/deliveries` (`webhook:write`) accepts `{ "event": { "id": "..." }, "outcome": "success" | "fail" }`. Los fallos se limitan a tres intentos: los dos primeros devuelven `retry_scheduled`, y el tercero se admite en una cola de mensajes muertos en memoria. La cola está limitada a 100 entradas y todo el estado es local al proceso.

`GET /health/live` no requiere autenticación y devuelve `200`.

## Límite explícito del diario

Las intenciones de pago solo se almacenan en la memoria del proceso de la pasarela. Este servicio no tiene cliente de diario, base de datos, almacén de saldos, procesador de pagos, estado comercial ni entrega de webhooks, y nunca lee ni modifica saldos. La confirmación es sintética y no crea asientos ni mueve dinero.

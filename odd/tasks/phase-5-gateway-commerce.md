# Phase 5 — Gateway público y comercio sandbox

Fuente: `spec.md`, secciones 8.4 y 9. Incremento autorizado por el usuario: “DALE CONTINUA CON LOS SPECS”.
Scope: implementar el primer slice ejecutable del gateway público y una tienda sandbox, sin conectar dinero real ni alterar infraestructura externa.

## Specs

1. **S1 — Gateway público**
   > `cholopay-gateway` | Node.js + TypeScript | BFF/API developer, auth de clientes externos, checkout y webhooks; no escribe ledger

2. **S2 — Funcionalidades del gateway y comercio**
   > Sandbox con claves de API de alcance limitado; creación de `payment_intents`; checkout; QR dinámico; confirmación de pago; webhooks HMAC firmados con timestamp y protección de replay; reintentos con DLQ; historial de entregas; reembolsos; SDK TypeScript; portal de documentación OpenAPI; tienda de prueba con catálogo, carrito, órdenes, inventario y webhook idempotente.

3. **S3 — Separación del ledger**
   > El gateway nunca modifica balances.

4. **S4 — Contratos API preliminares**
   > **Gateway público (Node.js):**
   > - `POST /v1/payment-intents`
   > - `GET /v1/payment-intents/{id}`
   > - `POST /v1/payment-intents/{id}/confirm`
   > - `POST /v1/refunds`
   > - `POST /v1/webhook-endpoints`

5. **S5 — Formato de intención**
   > ```json
   > {
   >   "amount_minor": 2500,
   >   "currency": "USD",
   >   "merchant_order_id": "ORDER-1001",
   >   "description": "Compra simulada"
   > }
   > ```

6. **S6 — Datos sintéticos y límites del proyecto**
   > No procesar dinero real, conectarse a SPI/DICT reales, simular cumplimiento regulatorio como certificación, copiar interfaces o marcas registradas, almacenar datos personales reales, operar como banco.

## Tasks

- T1 (S1, S3, S4): Crear gateway TypeScript Docker-only con contratos HTTP, autenticación sandbox de alcance limitado y almacenamiento local/in-memory explícito. Route: delegated by gateway surface. Commit: `bdcf987`.
- T2 (S2, S4, S5): Implementar payment intents, consulta y confirmación idempotente con estado simulado; no escribir ledger. Route: delegated by gateway surface. Commit: `bf06cf6`.
- T3 (S2, S6): Crear tienda sandbox mínima con catálogo, carrito, orden e inventario sintéticos. Route: delegated by commerce surface. Commit: pending.
- T4 (S2, S3): Implementar webhook HMAC con timestamp, protección de replay, reintentos acotados y DLQ in-memory. Route: delegated by gateway surface. Commit: `d6777eb`.
- T5 (S2, S4): Añadir pruebas públicas de contrato y comportamiento, documentación OpenAPI y evidencia Docker. Route: inline. Commit: `cd16ac9`.

## Log

L1. DALE CONTINUA CON LOS SPECS
L2. Se selecciona como siguiente incremento coherente el gateway público y comercio sandbox de `spec.md` secciones 8.4 y 9. Se excluyen por ahora utilities, tránsito, SDK separado, observabilidad avanzada y despliegues externos para mantener el slice revisable.
L3. T1 implementado en `gateway/`: scaffold Node.js/TypeScript Docker-only, autenticación X-Api-Key sintética con scopes, rutas contractuales placeholder y frontera explícita sin escritura de ledger. Build Docker y 3 tests pasaron. `package-lock.json` queda pendiente para fijar versiones. Commit: `bdcf987`.
L4. T2 implementado en `gateway/`: payment intents sintéticos, consulta, confirmación, validación, replay/conflicto de idempotencia y 6 tests HTTP públicos. Docker build pasó y reportó 6 tests sin fallos. Commit: `bf06cf6`.
L5. T4 implementado en `gateway/`: registro de endpoints webhook, HMAC-SHA256 con timestamp, replay protection, tres intentos máximos y DLQ in-memory limitada a 100 entradas. Docker build pasó y reportó 9 tests sin fallos. Commit: `d6777eb`.
L6. La verificación final Docker de Phase 5 pasó: gateway build/test con 9 tests, commerce Compose test con 3 tests y validación de `docker compose config`. No se publicaron puertos. Commit de evidencia: `cd16ac9`.

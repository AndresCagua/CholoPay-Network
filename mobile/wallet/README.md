# Estructura de la billetera CholoPay

Esta es una estructura mínima de billetera Flutter para la Fase 4 T2/T3. Usa datos sintéticos locales y tipados para una sesión de billetera, cuentas, CholoKeys, cargas útiles QR de CholoPay, transferencias, movimientos y confirmación de pagador/beneficiario.

El formato QR es propiedad de CholoPay y está versionado. **No es compatible con Pix**, y esta estructura no afirma compatibilidad con redes externas.

La aplicación no accede directamente a bases de datos ni contiene secretos. La confirmación es únicamente local: una confirmación repetida reproduce la transferencia existente usando su clave de idempotencia contractual sin crear un segundo identificador de transferencia. Los resultados son proyecciones de UI de solo lectura.

## Obligaciones participant-scoped

`lib/obligations_client.dart` separa la interfaz `ObligationsClient` del transporte HTTP. `HttpObligationsClient` recibe un `baseUri` configurable y un transporte inyectable para consultar e iniciar pagos sin red real en las pruebas:

- `GET /v1/participants/{participantId}/obligations`
- `POST /v1/participants/{participantId}/obligations/{obligationId}/payment-intents` (adaptador participant-scoped del banco; envía la cuenta de origen y la clave de idempotencia)

La UI usa fixtures sintéticos por defecto y conserva estados explícitos de carga, error y vacío. El cliente solo envía el `participantId` de la sesión y una clave de idempotencia; no usa acceso directo a bases, credenciales ni secretos. Las pruebas reemplazan el transporte por un fake o usan `SyntheticObligationsClient`.

## Comprobaciones únicamente mediante Docker

Ejecuta desde `mobile/wallet` (el equipo anfitrión no necesita un SDK de Flutter):

```sh
docker build -t cholopay-wallet:test .
docker run --rm cholopay-wallet:test flutter test
docker run --rm cholopay-wallet:test flutter analyze
```

La imagen usa `ghcr.io/cirruslabs/flutter:3.24.5` y ejecuta todos los comandos de Flutter dentro de Docker.

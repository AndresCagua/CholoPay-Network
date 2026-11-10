# Estructura de la billetera CholoPay

Esta es una estructura mínima de billetera Flutter para la Fase 4 T2/T3. Usa datos sintéticos locales y tipados para una sesión de billetera, cuentas, CholoKeys, cargas útiles QR de CholoPay, transferencias, movimientos y confirmación de pagador/beneficiario.

El formato QR es propiedad de CholoPay y está versionado. **No es compatible con Pix**, y esta estructura no afirma compatibilidad con redes externas.

La aplicación no accede a bases de datos, llama a un backend ni contiene secretos. La confirmación es únicamente local: una confirmación repetida reproduce la transferencia existente usando su clave de idempotencia contractual sin crear un segundo identificador de transferencia. Los resultados son proyecciones de UI de solo lectura.

## Comprobaciones únicamente mediante Docker

Ejecutá desde `mobile/wallet` (el equipo anfitrión no necesita un SDK de Flutter):

```sh
docker build -t cholopay-wallet:test .
docker run --rm cholopay-wallet:test flutter test
docker run --rm cholopay-wallet:test flutter analyze
```

La imagen usa `ghcr.io/cirruslabs/flutter:3.24.5` y ejecuta todos los comandos de Flutter dentro de Docker.

# Runbook P5 — rate limiting sintético local

Este slice implementa `Spec: odd/tasks/phase-5-rate-limiting.md, T1-T2, S1-S5`. El gateway limita únicamente `POST /v1/payment-intents`, después de autenticación/autorización y antes de crear la intención. El estado vive en memoria del proceso y cada API key tiene su propio bucket.

## Ejecución Docker-only

Desde la raíz:

```sh
./infra/tests/p5_rate_limiting_acceptance.sh
```

El runner inicia el profile `p5` y verifica desde un contenedor privado: `201, 201, 429` con claves de idempotencia distintas, `Retry-After` positivo, ausencia de la tercera intención, `403` para la clave read-only y health `200`. El compose no publica puertos al host y este profile no depende de Redis ni Kafka.

## Configuración local

Los límites son explícitos y solo locales:

- `RATE_LIMIT_MAX_REQUESTS=2` por ventana.
- `RATE_LIMIT_WINDOW_SECONDS=60`.

Ambos valores deben ser enteros positivos; valores ausentes o inválidos vuelven a esos defaults seguros. No representan capacidad, SLO o una garantía global. El límite no se comparte entre réplicas y se reinicia al reiniciar el proceso.

Las solicitudes sin credenciales, forbidden, con JSON inválido, sin idempotencia o con payload rechazado no consumen una unidad; health y rutas desconocidas tampoco pasan por el limitador. La respuesta limitada es HTTP `429`, JSON `{error,message}` con `error=rate_limited` y cabecera `Retry-After` entera positiva.

## Límites del slice

Esto no implementa protección distribuida/productiva, Redis obligatorio, OAuth/OIDC, TLS/mTLS, CORS/CSRF, cumplimiento normativo ni seguridad financiera completa. El runtime ledger permanece intacto: el gateway sigue siendo sintético y nunca escribe saldos ni diarios.

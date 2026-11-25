# Phase 5 — Rate limiting sintético del gateway
Fuente: `spec.md`, secciones 4, 5, 6, 7 y 27.
Alcance: demostrar un límite local por API key sobre una ruta protegida del gateway; no es protección productiva ni distribuida.

## Specs

1. **S1 — Ruta y límite:** Aplicar un límite configurable, determinista y local al proceso sobre `POST /v1/payment-intents`, después de autenticación/autorización y antes de ejecutar la mutación.
2. **S2 — Contrato 429:** Cuando se exceda el límite, responder `429`, cuerpo JSON compatible `{error,message}`, campo `error=rate_limited` y cabecera `Retry-After` entera positiva.
3. **S3 — Aislamiento de identidad:** Separar buckets por API key; una clave de solo lectura debe continuar recibiendo `403` por autorización y no ser convertida en `429`.
4. **S4 — Aceptación Docker-only:** Verificar en una ejecución limpia `201, 201, 429` para la misma clave, `Retry-After`, ausencia de mutación en la solicitud limitada, `403` para la clave read-only, health `200` y cero puertos host.
5. **S5 — Límites:** No afirmar precisión global/distribuida, Redis obligatorio, protección productiva, OAuth/OIDC, TLS/mTLS, CORS/CSRF, cumplimiento normativo ni seguridad financiera completa.

## Tasks

- T1 [S1-S3] Implementar limitación local y tests de compatibilidad. Ruta: gateway + OpenAPI. Commit: pendiente.
- T2 [S4-S5] Crear aceptación Docker-only, runbook y documentación. Ruta: infra/docs. Commit: pendiente.
- T3 [S1-S5] Ejecutar checks, crear work unit y evaluar RDD. Ruta: Docker + Git. Commit: pendiente.

## Log

- L1. El usuario eligió P5 rate limiting sintético como siguiente slice.
- L2. La exploración recomendó `POST /v1/payment-intents` del gateway por API key: identidad clara, estado local existente y ausencia de efectos directos sobre el ledger.

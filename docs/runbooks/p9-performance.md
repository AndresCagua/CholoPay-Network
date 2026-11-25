# Runbook P9 — lectura financiera sintética

Este slice ejecuta una carga pequeña, fija y Docker-only contra `bank-core`. El runner crea primero un cliente/cuenta sintéticos mediante `POST /v1/customers` (Bearer `BANK_API_TOKEN`), obtiene el token de lectura emitido por esa interfaz y consulta `GET /v1/accounts/{account_id}/balance`. No escribe directamente en PostgreSQL ni en un ledger.

## Ejecución

```sh
infra/tests/p9_performance_acceptance.sh
```

La configuración predeterminada es 20 requests, concurrencia 4, seed lógico `p9-financial-read-v1` y ningún puerto publicado. Se pueden cambiar `P9_REQUESTS`, `P9_CONCURRENCY`, `P9_LOGICAL_SEED` y `GIT_COMMIT` para una ejecución explícita; conservar esos valores junto con la salida JSON.

## Evidencia y aceptación

La salida `run_summary` incluye `target`, requests, concurrencia, duración, commit, seed lógico, estado esperado, errores, throughput y latencias mínima, mediana, p95, p99 y máxima. La aceptación además compara la identidad de la cuenta y sus saldos antes/después, exige todas las respuestas esperadas, `errors=0`, readiness HTTP 200 después de la carga y emite `P9 PERFORMANCE PASS`.

Interpretar los percentiles solamente para ese contenedor, fixture, commit y ejecución. Este resultado no mide ni afirma capacidad productiva, 1M TPS, SLO/SLI, locks o contención, CPU/memoria, outbox/DLQ, reconciliación bajo carga ni garantías no observadas. Tampoco es un perfil de hardware ni una prueba de datos reales.

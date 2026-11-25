# Runbook P9 — lectura financiera sintética

Este slice ejecuta una carga pequeña, fija y Docker-only contra `bank-core`. El runner crea primero un cliente/cuenta sintéticos mediante `POST /v1/customers` (Bearer `BANK_API_TOKEN`), obtiene el token de lectura emitido por esa interfaz y consulta `GET /v1/accounts/{account_id}/balance`. No escribe directamente en PostgreSQL ni en un ledger.

## Ejecución

```sh
infra/tests/p9_performance_acceptance.sh
```

La configuración predeterminada es el perfil `p9-financial-read-v1`: 1000 requests, concurrencia 20, seed lógico `p9-financial-read-v1` y ningún puerto publicado. Existe además el perfil explícito `p9-financial-read-v2-2k`, con 2000 requests, concurrencia 20 y seed lógico `p9-financial-read-v2-2k`:

```sh
P9_PROFILE=p9-financial-read-v2-2k infra/tests/p9_performance_acceptance.sh
```

El runner fija los parámetros de cada perfil y rechaza perfiles desconocidos. `GIT_COMMIT` sigue siendo registrable; no se agregan umbrales de throughput o latencia.

## Evidencia y aceptación

La salida `run_summary` incluye `target`, requests, concurrencia, duración, commit, seed lógico, estado esperado, errores, throughput y latencias mínima, mediana, p95, p99 y máxima. La aceptación además compara la identidad de la cuenta y sus saldos antes/después, exige todas las respuestas esperadas, `errors=0`, readiness HTTP 200 después de la carga y emite `P9 PERFORMANCE PASS`.

Interpretar los percentiles solamente para ese contenedor, fixture, commit y ejecución. Este resultado no mide ni afirma capacidad productiva, 1M TPS, SLO/SLI, locks o contención, CPU/memoria, outbox/DLQ, reconciliación bajo carga ni garantías no observadas. Tampoco es un perfil de hardware ni una prueba de datos reales.

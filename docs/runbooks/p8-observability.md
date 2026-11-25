# Runbook P8 — health/readiness y observabilidad local

P8.1 health/readiness se documenta abajo. Para el profile opcional de métricas P8.2, consultar `docs/runbooks/p8-metrics.md`.

P8.1 verifica únicamente el estado operacional local de los servicios sintéticos dentro de Docker. No es monitoreo de producción.

## Ejecución

Desde la raíz del repositorio:

```sh
./infra/tests/p8_health_acceptance.sh
```

La aceptación construye y arranca `network`, `bank-core`, `gateway`, `transit` y `utilities` sin publicar puertos al host. Un contenedor Python dentro de la red privada consulta `GET /health/live` y `GET /health/ready`, parsea sus cuerpos JSON y exige `status=UP` en `network`/`bank-core` y `status=ok` en `gateway`/`transit`/`utilities`. Después detiene externamente `network-db`, comprueba que `network` conserva `live=200` con `status=UP` pero devuelve `ready=503` con `status=DOWN`, reinicia la base y comprueba HTTP 200 con el cuerpo healthy. Finalmente detiene únicamente `costa-db`, comprueba que `bank-core` conserva `live=200` con `status=UP` pero devuelve `ready=503` con `status=DOWN`, reinicia `costa-db`, espera la recuperación de PostgreSQL y comprueba `bank-core ready=200` con `status=UP`.

Para inspección manual dentro de Compose:

```sh
docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile p8 run --rm p8-health
docker compose --env-file infra/config.env.template -f infra/compose.yaml ps
```

## Interpretación

- `live=200` demuestra que el proceso HTTP está activo; el cuerpo healthy exige `status=UP` en `network`/`bank-core` y `status=ok` en los demás servicios.
- `ready=200` demuestra que el servicio está listo; en `network` y `bank-core` incluye una comprobación de PostgreSQL y conserva el `status` healthy correspondiente.
- `network ready=503` con `network-db` detenido, junto con `status=DOWN`, demuestra dependencia no disponible, no una caída del proceso.
- `bank-core live=200` con `status=UP` y `ready=503` con `status=DOWN` al detener `costa-db` demuestran que el proceso sigue activo y que su readiness falla de forma cerrada; tras reiniciar `costa-db`, `ready=200` con `status=UP` demuestra la recuperación observada en este entorno.
- `gateway`, `transit` y `utilities` no tienen PostgreSQL propio en este Compose sintético; su readiness solo demuestra que el proceso responde.
- Los healthchecks HTTP de Compose cubren ambos endpoints en los cinco servicios; `network` usa su binario auxiliar estático porque su runtime es distroless.

## Contrato de cuerpos y límites

El probe valida únicamente la clave JSON `status` en los cuerpos de estos dos endpoints. Para los cinco servicios, las respuestas saludables de live/ready deben ser HTTP 200; `network` y `bank-core` usan `UP`, mientras `gateway`, `transit` y `utilities` usan `ok`. Solo la dependencia PostgreSQL detenida cambia el readiness afectado a HTTP 503 con `status=DOWN`; live y los demás servicios conservan su contrato healthy. Este contrato se limita a la aceptación local Docker-only y no cambia endpoints ni aplicaciones.

## Límites explícitos

Este runbook no añade ni afirma métricas, dashboards, alertas, OpenTelemetry, tracing, SLO/SLI, retención histórica, disponibilidad productiva ni métricas de negocio. La evidencia de `costa-db` se limita al banco Costa sintético y no afirma HA, RPO/RTO ni recuperación de otros servicios. Los datos y dependencias son sintéticos, la red permanece privada y estos checks no sustituyen pruebas de carga, backup/restore, reconciliación ni observabilidad de producción.

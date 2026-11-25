# Runbook P8.2 — métricas locales sintéticas

Este slice implementa el alcance aprobado `Spec: odd/tasks/phase-20-receiver-outage-recovery.md, T28-T30, S21-S24`. Prometheus y Grafana son opcionales y solo arrancan con el profile explícito `observability`.

## Ejecución

Desde la raíz del repositorio:

```sh
./infra/tests/p8_observability_acceptance.sh
```

La aceptación verifica que el modelo Compose por defecto no publica puertos, inicia `prometheus` y `grafana` con el profile `observability`, comprueba targets `network` y `bank-core` en estado `up`, consulta una métrica scrapeada y valida el datasource y dashboard provisionados de Grafana.

Para iniciar manualmente el profile:

```sh
docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile observability up -d network-db network bank-core prometheus grafana
```

No se debe añadir `-p`/`ports` para acceder desde el host. La inspección se hace desde otro contenedor de la red privada, por ejemplo:

```sh
docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile observability run --rm p8-observability
```

## Métricas y dashboard

`network:8080/metrics` y `bank-core:8080/metrics` exponen únicamente métricas bounded de proceso, uptime, solicitudes HTTP y el marcador `cholonet_synthetic_acceptance_info{scope="local"}`. Prometheus usa `infra/observability/prometheus.yml`; Grafana carga automáticamente el datasource Prometheus y el dashboard `cholonet-local` desde `infra/observability/grafana/`.

La aceptación imprime un `P8.2 OBSERVABILITY PASS` claro cuando el scrape y el provisioning son correctos.

## Límites y almacenamiento

- La red de observabilidad es privada (`internal_services`) y no hay puertos publicados por defecto.
- Prometheus y Grafana usan `tmpfs`; el almacenamiento es efímero y se pierde al recrear los contenedores.
- Los datos son sintéticos y educativos: no son métricas estables de negocio, ledger, settlement, outbox o DLQ.
- Este slice no define SLO/SLI, alertas, tracing, HA, capacidad, monitoreo remoto, retención durable ni seguridad productiva.

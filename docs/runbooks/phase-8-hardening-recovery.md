# Fase 8 — Runbook de endurecimiento y recuperación

Todos los comandos usan Docker Compose y datos locales sintéticos. No uses este runbook contra un entorno de producción o externo.

## Check local de endurecimiento

```bash
./infra/scripts/hardening-check.sh
./infra/scripts/check-ports.sh
```

La configuración predeterminada no debe publicar puertos del host. Cualquier perfil opcional que publique puertos requiere una revisión explícita y un check de puertos libres.

## Iniciar bases de datos aisladas

```bash
docker compose --env-file infra/config.env.template -f infra/compose.yaml up -d network-db costa-db sierra-db oriente-db
```

## Respaldo

```bash
./infra/scripts/backup-restore.sh backup
```

Los respaldos se escriben en `.artifacts/backups/`, que contiene evidencia local y no debe contener datos personales o financieros reales.

## Restaurar una base de datos de red sintética

```bash
RESTORE_SERVICE=network-db ./infra/scripts/backup-restore.sh restore .artifacts/backups/network-db-<timestamp>.sql
```

La restauración es destructiva para la base de datos sintética seleccionada. `RESTORE_SERVICE` y el nombre del volcado deben referirse a la misma base de datos: `network-db` → `cholopay_network`, `costa-db` → `cholopay_costa`, `sierra-db` → `cholopay_sierra`, `oriente-db` → `cholopay_oriente`. Ejecútala únicamente contra volúmenes locales desechables y verifica después la salud del servicio.

## Caída de participante y rotación de credenciales

1. Stop only the affected synthetic service with `docker compose ... stop <service>`.
2. Mantener intactos los ledgers de la red y de los bancos; reconciliar los mensajes pendientes antes de reiniciar.
3. Restart the service and run the relevant Docker test profile.
4. Rotate only synthetic environment values in an ignored local config; never commit real credentials.
5. Record the incident, command output, and resulting commit in the phase task log.

## Límites

Esta fase no proporciona garantías de respaldo de producción, despliegue en Kubernetes, GitOps, publicación externa ni gestión real de secretos.

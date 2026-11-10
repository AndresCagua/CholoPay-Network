# Phase 8 — Hardening and recovery runbook

All commands use Docker Compose and synthetic local data. Do not use this runbook against a production or external environment.

## Local hardening check

```bash
./infra/scripts/hardening-check.sh
./infra/scripts/check-ports.sh
```

The default configuration must publish no host ports. Any optional published-port profile requires an explicit review and free-port check.

## Start isolated databases

```bash
docker compose --env-file infra/config.env.template -f infra/compose.yaml up -d network-db costa-db sierra-db oriente-db
```

## Backup

```bash
./infra/scripts/backup-restore.sh backup
```

Backups are written under `.artifacts/backups/`, which is local evidence and must not contain real personal or financial data.

## Restore a synthetic network database

```bash
RESTORE_SERVICE=network-db ./infra/scripts/backup-restore.sh restore .artifacts/backups/network-db-<timestamp>.sql
```

Restore is destructive for the selected synthetic database. `RESTORE_SERVICE` and the dump filename must refer to the same database: `network-db` → `cholopay_network`, `costa-db` → `cholopay_costa`, `sierra-db` → `cholopay_sierra`, `oriente-db` → `cholopay_oriente`. Run it only against disposable local volumes and verify the service health afterward.

## Participant outage and credential rotation

1. Stop only the affected synthetic service with `docker compose ... stop <service>`.
2. Keep the network and bank ledgers untouched; reconcile pending messages before restart.
3. Restart the service and run the relevant Docker test profile.
4. Rotate only synthetic environment values in an ignored local config; never commit real credentials.
5. Record the incident, command output, and resulting commit in the phase task log.

## Limits

This phase does not provide production backup guarantees, Kubernetes deployment, GitOps, external publication, or real secret management.

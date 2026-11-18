#!/bin/sh
# Orquestación externa: el contenedor de aceptación nunca accede al socket Docker.
set -eu

COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-infra/config.env.template}"
COMPOSE_FILE="${COMPOSE_FILE:-infra/compose.yaml}"
compose() { docker compose --env-file "$COMPOSE_ENV_FILE" -f "$COMPOSE_FILE" --profile phase20 "$@"; }

# prepare usa las dependencias Compose y deja una transferencia READY_FOR_SETTLEMENT.
compose run --rm -e PHASE20_MODE=prepare phase20-e2e
trap 'compose start bank-core-sierra >/dev/null' EXIT

# La caída y el reinicio son deliberadamente externos al runner Python.
compose stop bank-core-sierra
compose run --rm --no-deps -e PHASE20_MODE=settle phase20-e2e
compose run --rm --no-deps -e PHASE20_MODE=observe phase20-e2e
compose run --rm --no-deps -e PHASE20_MODE=exhaust phase20-e2e
compose run --rm --no-deps -e PHASE20_MODE=reconcile-dlq phase20-e2e
compose start bank-core-sierra
trap - EXIT
compose run --rm --no-deps -e PHASE20_MODE=replay phase20-e2e
compose run --rm --no-deps -e PHASE20_MODE=complete phase20-e2e
compose run --rm --no-deps -e PHASE20_MODE=reconcile-clean phase20-e2e

echo "PHASE20 PASS: outage prolongada, DLQ, replay autenticado, recuperación y reconciliación verificadas"

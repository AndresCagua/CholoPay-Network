#!/bin/sh
# Docker-only P8.1 acceptance: private HTTP probes plus external PostgreSQL stop/start.
set -eu
COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-infra/config.env.template}"
COMPOSE_FILE="${COMPOSE_FILE:-infra/compose.yaml}"
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-cholopay_p8_health}"
export OPERATOR_CONSOLE_TOKEN="${OPERATOR_CONSOLE_TOKEN:-phase20-p8-operator}"
export CHOLONET_HMAC_SECRET="${CHOLONET_HMAC_SECRET:-phase20-p8-secret}"
compose() { docker compose --env-file "$COMPOSE_ENV_FILE" -f "$COMPOSE_FILE" --profile p8 "$@"; }
cleanup() { compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT
compose up -d network-db network bank-core gateway transit utilities >/dev/null
compose run --rm p8-health >/dev/null
compose stop network-db >/dev/null
P8_MODE=network-db-down compose run --rm p8-health
compose start network-db >/dev/null
for attempt in $(seq 1 30); do
  if compose exec -T network-db pg_isready -U "${POSTGRES_USER:-cholopay}" -d "${NETWORK_DB_NAME:-cholopay_network}" >/dev/null 2>&1; then break; fi
  sleep 1
done
compose run --rm p8-health >/dev/null
echo "P8.1 HEALTH PASS: live/ready sano, network-db detenido externamente (network ready=503), y recuperación ready=200"

#!/bin/sh
# Docker-only P8.2 acceptance: private observability profile and ephemeral storage.
set -eu
COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-infra/config.env.template}"
COMPOSE_FILE="${COMPOSE_FILE:-infra/compose.yaml}"
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-cholopay_p8_observability}"
export OPERATOR_CONSOLE_TOKEN="${OPERATOR_CONSOLE_TOKEN:-phase20-p8-operator}"
export CHOLONET_HMAC_SECRET="${CHOLONET_HMAC_SECRET:-phase20-p8-secret}"
compose() { docker compose --env-file "$COMPOSE_ENV_FILE" -f "$COMPOSE_FILE" --profile observability "$@"; }
cleanup() { compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT

# The rendered default Compose model must not publish host ports.
if compose config | grep -E '^[[:space:]]+published:' >/dev/null; then
  echo "P8.2 ERROR: default Compose config publishes a host port" >&2
  exit 1
fi
printf '%s\n' 'P8.2: starting private observability profile (no Docker socket is mounted)'
compose up -d network-db network bank-core prometheus grafana
printf '%s\n' 'P8.2: baseline checks'
compose run --rm p8-observability

printf '%s\n' 'P8.2: stopping only costa-db and checking outage correlation'
compose stop costa-db
compose run --rm --no-deps -e P8_MODE=bank-db-down p8-observability

printf '%s\n' 'P8.2: restarting costa-db'
compose start costa-db
costa_db_recovered=0
for attempt in $(seq 1 30); do
  if compose exec -T costa-db pg_isready -U "${POSTGRES_USER:-cholopay}" -d "${COSTA_DB_NAME:-cholopay_costa}" >/dev/null 2>&1; then
    costa_db_recovered=1
    break
  fi
  sleep 1
done
if [ "$costa_db_recovered" -ne 1 ]; then
  echo "P8.2 ERROR: costa-db no se recuperó dentro de 30 segundos" >&2
  compose ps >&2
  exit 1
fi
compose run --rm --no-deps -e P8_MODE=recovered p8-observability
printf '%s\n' 'P8.2 OBSERVABILITY PASS: baseline, outage de costa-db y recovery con scrape/métricas verificables'

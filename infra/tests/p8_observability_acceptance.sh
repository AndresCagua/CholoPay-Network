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
  echo "P8.2 FAIL: default Compose config publishes a host port" >&2
  exit 1
fi
compose up -d network-db network bank-core prometheus grafana >/dev/null
compose run --rm p8-observability

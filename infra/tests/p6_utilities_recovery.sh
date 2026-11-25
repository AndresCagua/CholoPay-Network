#!/bin/sh
# Docker-only P6 acceptance. The orchestrator recreates only utilities; no container uses Docker socket.
set -eu
COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-infra/config.env.template}"
COMPOSE_FILE="${COMPOSE_FILE:-infra/compose.yaml}"
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-cholopay_p6_utilities}"
export INTERNAL_EVENT_SECRET="${INTERNAL_EVENT_SECRET:-p6-local-event-secret}"
export CHOLONET_HMAC_SECRET="${CHOLONET_HMAC_SECRET:-p6-local-hmac}"
export SIERRA_CHOLONET_HMAC_SECRET="${SIERRA_CHOLONET_HMAC_SECRET:-p6-sierra-hmac}"
export ORIENTE_CHOLONET_HMAC_SECRET="${ORIENTE_CHOLONET_HMAC_SECRET:-p6-oriente-hmac}"
compose() { docker compose --env-file "$COMPOSE_ENV_FILE" -f "$COMPOSE_FILE" --profile p6 "$@"; }
cleanup() { compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT

if compose config | grep -E '^[[:space:]]+published:' >/dev/null; then
  echo "P6 FAIL: host ports are published" >&2
  exit 1
fi
compose up -d utilities-db utilities >/dev/null
compose run --rm p6-utilities-recovery
compose stop utilities >/dev/null
compose rm -f utilities >/dev/null
compose up -d utilities >/dev/null
compose run --rm p6-utilities-recovery

echo "P6 UTILITIES RECOVERY PASS: electricity invoice, intent, signed event and idempotent replay survived utilities recreation with isolated PostgreSQL volume"

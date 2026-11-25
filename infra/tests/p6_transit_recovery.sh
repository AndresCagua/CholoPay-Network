#!/bin/sh
set -eu
COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-infra/config.env.template}"; COMPOSE_FILE="${COMPOSE_FILE:-infra/compose.yaml}"
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-cholopay_p6_transit}"; export INTERNAL_EVENT_SECRET="${INTERNAL_EVENT_SECRET:-p6-transit-event-secret}"
export CHOLONET_HMAC_SECRET="${CHOLONET_HMAC_SECRET:-p6-local-hmac}"; export OPERATOR_CONSOLE_TOKEN="${OPERATOR_CONSOLE_TOKEN:-p6-local-operator}"
compose(){ docker compose --env-file "$COMPOSE_ENV_FILE" -f "$COMPOSE_FILE" --profile p6 "$@"; }
cleanup(){ compose down -v --remove-orphans >/dev/null 2>&1 || true; }; trap cleanup EXIT
if compose config | grep -E '^[[:space:]]+published:' >/dev/null; then echo 'P6 transit FAIL: host ports are published' >&2; exit 1; fi
compose up -d --build transit-db transit >/dev/null
compose run --rm p6-transit-recovery
compose stop transit >/dev/null
compose rm -f transit >/dev/null
compose up -d transit >/dev/null
compose run --rm -e P6_TRANSIT_RECOVERY_MODE=recovery p6-transit-recovery
echo 'P6 TRANSIT RECOVERY PASS: isolated synthetic transit state survived transit recreation with PostgreSQL volume'

#!/bin/sh
# Docker-only P9 acceptance; the profile has no ports mapping and uses only HTTP APIs.
set -eu
COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-infra/config.env.template}"
COMPOSE_FILE="${COMPOSE_FILE:-infra/compose.yaml}"
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-cholopay_p9_performance}"
export OPERATOR_CONSOLE_TOKEN="${OPERATOR_CONSOLE_TOKEN:-phase20-p9-operator}"
export CHOLONET_HMAC_SECRET="${CHOLONET_HMAC_SECRET:-phase20-p9-secret}"
export GIT_COMMIT="${GIT_COMMIT:-$(git rev-parse HEAD 2>/dev/null || echo unknown)}"
export P9_PROFILE="${P9_PROFILE:-p9-financial-read-v1}"
compose() { docker compose --env-file "$COMPOSE_ENV_FILE" -f "$COMPOSE_FILE" --profile p9 "$@"; }
cleanup() { compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT
# Config validation and explicit private-port guard for the P9 service.
if compose config | awk '/^  p9-performance:/{inside=1; next} inside && /^  [a-zA-Z0-9_-]+:/{exit} inside && /ports:/{found=1} END{exit found ? 1 : 0}'; then :; else
  echo "P9 PERFORMANCE FAIL: p9-performance must not publish ports" >&2
  exit 1
fi
compose up --build --abort-on-container-exit --exit-code-from p9-performance p9-performance

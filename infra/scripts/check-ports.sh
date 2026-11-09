#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
ENV_FILE="$ROOT_DIR/infra/config.env"
[[ -f "$ENV_FILE" ]] || ENV_FILE="$ROOT_DIR/infra/config.env.template"
# shellcheck disable=SC1090
source "$ENV_FILE"

ports=("${NETWORK_DB_HOST_PORT}" "${COSTA_DB_HOST_PORT}" "${SIERRA_DB_HOST_PORT}" "${ORIENTE_DB_HOST_PORT}")
[[ -n "${REDIS_HOST_PORT:-}" ]] && ports+=("${REDIS_HOST_PORT}")
[[ -n "${KAFKA_HOST_PORT:-}" ]] && ports+=("${KAFKA_HOST_PORT}")

if ! command -v ss >/dev/null 2>&1; then
  echo "ERROR: ss is required to check host ports before startup" >&2
  exit 2
fi

failed=0
for port in "${ports[@]}"; do
  if ! [[ "$port" =~ ^[0-9]+$ ]] || ((port < 1 || port > 65535)); then
    echo "ERROR: invalid port: $port" >&2
    failed=1
  elif ss -ltnH "sport = :$port" | grep -q .; then
    echo "ERROR: host port $port is occupied" >&2
    failed=1
  else
    echo "OK: host port $port is available"
  fi
done

if ((failed)); then
  echo "Choose free ports in infra/.env, then rerun this check before make up." >&2
  exit 1
fi

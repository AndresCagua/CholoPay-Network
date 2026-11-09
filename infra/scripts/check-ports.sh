#!/usr/bin/env bash
set -euo pipefail

# The default Compose stack publishes no host ports. This optional guard only
# runs ss when a future deployment profile explicitly supplies port variables.
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
ENV_FILE="$ROOT_DIR/infra/config.env"
[[ -f "$ENV_FILE" ]] || ENV_FILE="$ROOT_DIR/infra/config.env.template"
# shellcheck disable=SC1090
source "$ENV_FILE"

ports=()
for name in NETWORK_DB_HOST_PORT COSTA_DB_HOST_PORT SIERRA_DB_HOST_PORT ORIENTE_DB_HOST_PORT REDIS_HOST_PORT KAFKA_HOST_PORT; do
  value="${!name:-}"
  [[ -n "$value" ]] && ports+=("$value")
done

if ((${#ports[@]} == 0)); then
  echo 'OK: default Docker-only stack publishes no host ports'
  exit 0
fi

if ! command -v ss >/dev/null 2>&1; then
  echo 'ERROR: ss is required only for a profile that publishes host ports' >&2
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
((failed == 0)) || { echo 'Choose free ports before enabling the publishing profile.' >&2; exit 1; }

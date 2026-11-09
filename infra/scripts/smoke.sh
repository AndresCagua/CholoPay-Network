#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
COMPOSE="docker compose --env-file $ROOT_DIR/infra/config.env.template -f $ROOT_DIR/infra/compose.yaml"
services=(network-db costa-db sierra-db oriente-db)
for service in "${services[@]}"; do
  ready=0
  for attempt in $(seq 1 30); do
    status=$($COMPOSE ps --format '{{.Service}} {{.Health}}' "$service" 2>/dev/null || true)
    if grep -q "$service healthy" <<<"$status"; then ready=1; break; fi
    sleep 1
  done
  if (( ! ready )); then
    echo "ERROR: $service did not become healthy: ${status:-not running}" >&2
    exit 1
  fi
  echo "OK: $service is healthy"
done

declare -A expected_db=(
  [network-db]=cholopay_network
  [costa-db]=cholopay_costa
  [sierra-db]=cholopay_sierra
  [oriente-db]=cholopay_oriente
)
for service in "${services[@]}"; do
  db=$($COMPOSE exec -T "$service" sh -c 'printf "%s" "$POSTGRES_DB"')
  if [[ "$db" != "${expected_db[$service]}" ]]; then
    echo "ERROR: $service owns '$db', expected '${expected_db[$service]}'" >&2
    exit 1
  fi
  echo "OK: $service owns database $db"
done

echo 'Smoke test passed inside Docker: four isolated PostgreSQL services are healthy.'

#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
COMPOSE=(docker compose --env-file "$ROOT_DIR/infra/config.env.template" -f "$ROOT_DIR/infra/compose.yaml")

config="$(${COMPOSE[@]} config)"
if grep -qE '^\s+ports:' <<<"$config"; then
  echo 'ERROR: default Compose configuration publishes host ports' >&2
  exit 1
fi

for required in network-db costa-db sierra-db oriente-db; do
  grep -qE "^  ${required}:$" <<<"$config" || { echo "ERROR: missing isolated service ${required}" >&2; exit 1; }
done

if grep -qE '(BEGIN (RSA|OPENSSH) PRIVATE KEY|AKIA[0-9A-Z]{16}|sk_live_|password[[:space:]]*=[[:space:]]*[^$])' "$ROOT_DIR"/infra/config.env.template; then
  echo 'ERROR: probable real secret found in local configuration template' >&2
  exit 1
fi

printf '%s\n' 'PASS: Compose has no default host ports, isolated databases are present, and no probable real secret is in the template.'

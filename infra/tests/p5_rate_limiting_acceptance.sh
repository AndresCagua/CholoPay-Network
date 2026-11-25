#!/bin/sh
set -eu

COMPOSE="docker compose --env-file infra/config.env.template -f infra/compose.yaml"
if $COMPOSE config | grep -q '^    ports:'; then
  echo "P5 FAIL: host ports are published" >&2
  exit 1
fi
if $COMPOSE --profile p5 config | grep -q 'redis:\|kafka:'; then
  echo "P5 FAIL: rate-limit profile depends on Redis or Kafka" >&2
  exit 1
fi
trap '$COMPOSE --profile p5 down --remove-orphans' EXIT
$COMPOSE --profile p5 up --build --abort-on-container-exit --exit-code-from p5-rate-limit p5-rate-limit

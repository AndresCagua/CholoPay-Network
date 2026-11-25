#!/bin/sh
# Orquestación externa: ningún contenedor recibe el socket Docker.
set -eu

COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-infra/config.env.template}"
COMPOSE_FILE="${COMPOSE_FILE:-infra/compose.yaml}"
export P7_IDEMPOTENCY_PREFIX="${P7_IDEMPOTENCY_PREFIX:-p7-http-recovery-$(date +%s)-$$-}"
compose() {
  CHOLONET_HMAC_SECRET="${CHOLONET_HMAC_SECRET:-p7-local-hmac}" \
  OUTBOX_ENABLED=true OUTBOX_TRANSPORT=http \
  docker compose --env-file "$COMPOSE_ENV_FILE" -f "$COMPOSE_FILE" --profile p7 "$@"
}

cleanup() {
  compose start network bank-core >/dev/null 2>&1 || true
}
trap cleanup EXIT

compose up -d --build network-db costa-db gateway network bank-core
compose run --rm phase18-seed

# La red se interrumpe antes de crear/publicar el evento; el pedido financiero queda
# en el outbox transaccional local de Costa.
compose stop network
compose run --rm --no-deps -e P7_MODE=prepare phase7-http-outbox
sleep 3
compose run --rm --no-deps -e P7_MODE=assert-pending phase7-http-outbox

# Reinicio externo del productor mientras el registro está PENDING o IN_FLIGHT.
compose stop bank-core
compose start bank-core
compose start network
compose run --rm --no-deps -e P7_MODE=assert-final phase7-http-outbox
trap - EXIT

printf '%s\n' 'P7 PASS: recuperación HTTP del outbox verificada en Docker (at-least-once local; sin claim exactly-once universal)'

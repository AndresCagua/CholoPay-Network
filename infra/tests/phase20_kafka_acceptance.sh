#!/bin/sh
# External Docker-only orchestration for T14. Run from the repository root.
# This script owns Docker lifecycle; no acceptance container receives the Docker socket.
set -eu

COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-infra/config.env.template}"
COMPOSE_FILE="${COMPOSE_FILE:-infra/compose.yaml}"
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-cholopay_phase20_kafka}"
export OUTBOX_ENABLED=true
export OUTBOX_TRANSPORT=kafka
export KAFKA_BOOTSTRAP_SERVERS=kafka:9092
export KAFKA_TOPIC=cholonet.inter-network.v1
export KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092
export CHOLONET_HMAC_SECRET="${CHOLONET_HMAC_SECRET:-phase20-kafka-test-secret}"
export OPERATOR_CONSOLE_TOKEN="${OPERATOR_CONSOLE_TOKEN:-phase20-kafka-operator}"

compose() {
  docker compose --env-file "$COMPOSE_ENV_FILE" -f "$COMPOSE_FILE" \
    --profile events --profile phase20 "$@"
}

restart_broker() { compose start kafka >/dev/null 2>&1 || true; }
trap restart_broker EXIT

# Start Kafka first so the eager producer can resolve its broker during bank-core startup.
compose up -d network-db kafka
wait_for_kafka() {
  kafka_ready=false
  for _ in $(seq 1 60); do
    if compose exec -T kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list >/dev/null 2>&1; then
      kafka_ready=true
      break
    fi
    sleep 2
  done
  if [ "$kafka_ready" != true ]; then
    echo "Kafka broker did not become ready" >&2
    exit 1
  fi
}
wait_for_kafka
compose up -d network bank-core
compose run --rm phase18-seed >/dev/null

# The broker is down before the outbox event is created and therefore cannot publish it.
compose stop kafka >/dev/null
ORDER_JSON=""
last_error=""
i=0
while [ "$i" -lt 30 ]; do
  if ORDER_JSON=$(compose run --rm --no-deps phase20-kafka-order 2>&1); then
    break
  fi
  last_error="$ORDER_JSON"
  ORDER_JSON=""
  i=$((i + 1))
  sleep 2
done
if [ -z "$ORDER_JSON" ]; then
  echo "Kafka acceptance could not create the outbox event after broker stop: $last_error" >&2
  exit 1
fi

# Restart Kafka externally, then let the consumer deliver and replay the stable event.
compose start kafka >/dev/null
wait_for_kafka
CONSUMER_JSON=$(compose run --rm --no-deps phase20-kafka-consumer)

python3 - "$ORDER_JSON" "$CONSUMER_JSON" <<'PY'
import json, sys
order = json.loads(sys.argv[1].splitlines()[-1])
consumer = json.loads(sys.argv[2].splitlines()[-1])
if order["event_id"] != consumer["event_id"] or order["event_id"] != consumer["key"]:
    raise SystemExit(f"event identity changed: order={order} consumer={consumer}")
if consumer["replay_status"] != 200:
    raise SystemExit(f"boundary replay was not accepted: {consumer}")
print("PHASE20 KAFKA PASS: stable event identity, one durable PostgreSQL boundary record, at-least-once replay")
PY

trap - EXIT

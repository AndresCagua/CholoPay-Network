#!/bin/sh
# Docker-only logical backup/restore acceptance for one disposable network database.
set -eu

COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-infra/config.env.template}"
COMPOSE_FILE="${COMPOSE_FILE:-infra/compose.yaml}"
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-cholopay_phase20_backup}"
export OPERATOR_CONSOLE_TOKEN="${OPERATOR_CONSOLE_TOKEN:-phase20-backup-operator}"
export CHOLONET_HMAC_SECRET="${CHOLONET_HMAC_SECRET:-phase20-backup-secret}"
BACKUP_DIR="${BACKUP_DIR:-$(mktemp -d)}"
export BACKUP_DIR
MESSAGE_ID="00000000-0000-4000-8000-00000000b001"
IDEMPOTENCY_KEY="phase20-backup-$MESSAGE_ID"
PAYLOAD_HASH="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

compose() {
  docker compose --env-file "$COMPOSE_ENV_FILE" -f "$COMPOSE_FILE" --profile phase20 "$@"
}
psql_network() {
  compose exec -T network-db psql -v ON_ERROR_STOP=1 -U "${POSTGRES_USER:-cholopay}" -d "${NETWORK_DB_NAME:-cholopay_network}" "$@"
}
cleanup() { compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT

compose up -d network-db network >/dev/null
compose run --rm phase18-seed >/dev/null

psql_network -c "INSERT INTO inter_network_message(message_id,idempotency_key,sender_network_id,destination_network_id,schema_version,message_type,correlation_id,causation_id,payload_hash,payload,status) VALUES ('$MESSAGE_ID','$IDEMPOTENCY_KEY','00000000-0000-4000-8000-000000000201','00000000-0000-4000-8000-000000000202','1.0','TRANSFER_REQUESTED','00000000-0000-4000-8000-00000000b002','00000000-0000-4000-8000-00000000b003','$PAYLOAD_HASH','{\"transfer_id\":\"$MESSAGE_ID\",\"amount_minor\":125}'::jsonb,'ACCEPTED')"
BASELINE="$(psql_network -At -F '|' -c "SELECT message_id,idempotency_key,payload_hash,payload::text,status FROM inter_network_message WHERE message_id='$MESSAGE_ID'")"
test -n "$BASELINE"

BACKUP_SERVICES=network-db BACKUP_DIR="$BACKUP_DIR" ./infra/scripts/backup-restore.sh backup >/dev/null
BACKUP_FILE="$(find "$BACKUP_DIR" -maxdepth 1 -type f -name 'network-db-*.sql' -print -quit)"
test -s "$BACKUP_FILE"

psql_network -c "DELETE FROM inter_network_message WHERE message_id='$MESSAGE_ID'"
test -z "$(psql_network -At -c "SELECT message_id FROM inter_network_message WHERE message_id='$MESSAGE_ID'")"

RESTORE_SERVICE=network-db ./infra/scripts/backup-restore.sh restore "$BACKUP_FILE" >/dev/null
RESTORED="$(psql_network -At -F '|' -c "SELECT message_id,idempotency_key,payload_hash,payload::text,status FROM inter_network_message WHERE message_id='$MESSAGE_ID'")"
test "$RESTORED" = "$BASELINE"
psql_network -c "SELECT 1 FROM pg_constraint WHERE conrelid='inter_network_message'::regclass AND conname IN ('inter_network_message_pkey','inter_network_message_idempotency_key_key')" | grep -q 1
compose exec -T network-db pg_isready -U "${POSTGRES_USER:-cholopay}" -d "${NETWORK_DB_NAME:-cholopay_network}" >/dev/null

echo "PHASE20 BACKUP RESTORE PASS: network-db identity, payload hash, idempotency constraint and readiness restored"

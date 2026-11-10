#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
BACKUP_DIR="${BACKUP_DIR:-$ROOT_DIR/.artifacts/backups}"
COMPOSE=(docker compose --env-file "$ROOT_DIR/infra/config.env.template" -f "$ROOT_DIR/infra/compose.yaml")
mkdir -p "$BACKUP_DIR"

usage() { echo "Usage: $0 backup|restore <file>" >&2; exit 2; }

backup() {
  local stamp file service
  stamp=$(date -u +%Y%m%dT%H%M%SZ)
  for service in network-db costa-db sierra-db oriente-db; do
    file="$BACKUP_DIR/${service}-${stamp}.sql"
    local db
    case "$service" in
      network-db) db=cholopay_network ;;
      costa-db) db=cholopay_costa ;;
      sierra-db) db=cholopay_sierra ;;
      oriente-db) db=cholopay_oriente ;;
    esac
    "${COMPOSE[@]}" exec -T "$service" pg_dump -U "${POSTGRES_USER:-cholopay}" -d "$db" --clean --if-exists > "$file"
    test -s "$file"
    printf 'BACKUP %s %s\n' "$service" "$file"
  done
}

restore() {
  [[ $# -eq 1 && -s "$1" ]] || usage
  local service
  service="${RESTORE_SERVICE:-network-db}"
  "${COMPOSE[@]}" exec -T "$service" psql -U "${POSTGRES_USER:-cholopay}" -d cholopay_network < "$1"
  printf 'RESTORE %s %s\n' "$service" "$1"
}

case "${1:-}" in
  backup) backup ;;
  restore) restore "${2:-}" ;;
  *) usage ;;
esac

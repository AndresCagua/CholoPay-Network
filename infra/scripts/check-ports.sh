#!/usr/bin/env bash
set -euo pipefail

# La pila Compose predeterminada no publica puertos del host. Esta protección
# opcional solo ejecuta ss cuando un perfil futuro de despliegue proporciona explícitamente variables de puerto.
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
  echo 'OK: la pila predeterminada exclusiva para Docker no publica puertos del host'
  exit 0
fi

if ! command -v ss >/dev/null 2>&1; then
  echo 'ERROR: ss solo es necesario para un perfil que publique puertos del host' >&2
  exit 2
fi

failed=0
for port in "${ports[@]}"; do
  if ! [[ "$port" =~ ^[0-9]+$ ]] || ((port < 1 || port > 65535)); then
    echo "ERROR: puerto no válido: $port" >&2
    failed=1
  elif ss -ltnH "sport = :$port" | grep -q .; then
    echo "ERROR: el puerto del host $port está ocupado" >&2
    failed=1
  else
    echo "OK: el puerto del host $port está disponible"
  fi
done
((failed == 0)) || { echo 'Elige puertos libres antes de habilitar el perfil de publicación.' >&2; exit 1; }

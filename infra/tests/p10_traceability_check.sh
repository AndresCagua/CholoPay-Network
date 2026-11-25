#!/bin/sh
# Docker-only P10 documentation/traceability validation.
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
CONFIG=$(mktemp)
trap 'rm -f "$CONFIG"' EXIT

cd "$ROOT"
CHOLONET_HMAC_SECRET=local-test-only \
SIERRA_CHOLONET_HMAC_SECRET=sierra-local-test-only \
ORIENTE_CHOLONET_HMAC_SECRET=oriente-local-test-only \
docker compose --env-file infra/config.env.template -f infra/compose.yaml config >"$CONFIG"
docker run --rm \
  -v "$ROOT:/repo:ro" \
  -v "$CONFIG:/tmp/compose-config.yaml:ro" \
  -w /repo \
  python:3.12-alpine \
  python infra/tests/p10_traceability_check.py --compose-config /tmp/compose-config.yaml

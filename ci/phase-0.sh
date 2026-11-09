#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
docker run --rm -v "$ROOT_DIR:/workspace:ro" -w /workspace python:3.13-alpine python -m json.tool sbom/dependencies.spdx.json >/dev/null
docker compose --env-file "$ROOT_DIR/infra/config.env.template" -f "$ROOT_DIR/infra/compose.yaml" config --quiet
test -f "$ROOT_DIR/README.md"
test -f "$ROOT_DIR/api/network.openapi.yaml"
test -f "$ROOT_DIR/docs/security-threat-model.md"
printf '%s\n' 'Phase 0 Docker-only structural checks passed'

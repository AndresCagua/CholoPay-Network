#!/usr/bin/env bash
set -euo pipefail

python3 -m json.tool sbom/dependencies.spdx.json >/dev/null
docker compose --env-file infra/config.env.template -f infra/compose.yaml config --quiet
test -f README.md
test -f api/network.openapi.yaml
test -f docs/security-threat-model.md
printf '%s\n' 'Phase 0 structural checks passed'

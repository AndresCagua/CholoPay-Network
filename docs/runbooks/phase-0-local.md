# Phase 0 local runbook

1. Install Linux-native Docker Engine or Podman, Compose plugin, GNU Make, and a YAML/OpenAPI linter.
2. Copy `infra/.env.example` to `infra/.env` only if changing ports.
3. Run `./infra/scripts/check-ports.sh`; resolve conflicts before any start command.
4. Render with `make config`.
5. Start with `make up`.
6. Run `make smoke` and inspect logs if it fails.
7. Stop with `make down`; volumes are retained by default.

The Compose environment is for synthetic data only. Do not expose database ports beyond localhost or add credentials to Git.

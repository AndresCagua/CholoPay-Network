# Phase 0 local runbook

1. Install Docker Engine and the Compose plugin; no language runtime, build tool, database client, or host service is required.
2. Render with `docker compose --env-file infra/config.env.template -f infra/compose.yaml config`.
3. Start the private stack with `docker compose --env-file infra/config.env.template -f infra/compose.yaml up -d network-db costa-db sierra-db oriente-db bank-core`.
4. Run tests with `docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile test run --rm bank-core-test`.
5. Run `./infra/scripts/smoke.sh` to verify health and database isolation from Docker.
6. Stop with `docker compose --env-file infra/config.env.template -f infra/compose.yaml down`; volumes are retained by default.

The default Compose environment binds no host ports, so it cannot collide with host services. If a future profile publishes a port, add a Docker-only deployment check and verify availability before enabling it. The Compose environment is for synthetic data only; do not add credentials to Git.

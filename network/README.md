# CholoNet network skeleton — Phase 2

This Go service is the first central-network slice. It currently exposes only liveness/readiness endpoints and a tested transfer state machine. It does **not** write settlement entries, resolve CholoKeys, access bank tables, or claim that a transfer is settled.

Build and test only through Docker from the repository root:

```bash
docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile network-test run --rm network-test
```

The runtime image is distroless and communicates only on the Compose network. No host port is published by default.

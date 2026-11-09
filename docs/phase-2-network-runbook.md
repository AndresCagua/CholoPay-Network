# Phase 2 network skeleton runbook

The network skeleton is internal-only and has no host port by default. Build and test it inside Docker:

```bash
docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile network-test run --rm network-test
```

The skeleton health check and state-transition tests do not claim that any transfer is financially settled. A settlement claim requires the future central ledger transaction and its evidence.

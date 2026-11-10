# CholoLab

CholoLab is a Docker-only synthetic load and fault-injection runner. It does not access ledgers or databases directly and does not publish ports by default.

## Reproducible checks

```bash
docker compose -f compose.yaml --profile test build chololab-test
docker compose -f compose.yaml config
docker compose -f compose.yaml --profile test run --rm chololab-test
```

## Load smoke scenario

The Compose file includes an internal-only `synthetic-target` HTTP service for deterministic local evidence:

```bash
docker compose -f compose.yaml up -d synthetic-target
docker compose -f compose.yaml run --rm \
  -e TARGET_URL=http://synthetic-target:8080 \
  -e REQUESTS=20 -e CONCURRENCY=4 -e FAILURE_EVERY=5 chololab
docker compose -f compose.yaml down
```

The runner reports request count, injected failures, successful responses, elapsed time, throughput, and min/median/max latency. These are local smoke measurements, not production capacity claims.

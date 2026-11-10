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

The runner emits one JSON `run_started` record and one JSON `run_summary` record. The summary keeps request count, injected failures, successful responses, elapsed time, throughput, and min/median/max latency, plus the RED-style fields:

- `red.rate_requests_per_second`: completed request rate for this run.
- `red.errors` and `red.error_rate`: non-2xx, transport, and injected failures.
- `red.duration_ms`: total elapsed run duration.

These are deterministic synthetic smoke measurements for the selected inputs, not production capacity claims. Compare complete JSON records from runs with the same target, request count, concurrency, and `FAILURE_EVERY`; do not extrapolate them to production hardware or traffic.

## Runbook

1. Start only the internal synthetic target: `docker compose -f compose.yaml up -d synthetic-target`.
2. Run the bounded scenario shown above. Keep `TARGET_URL` on the Compose network; do not add a host `ports` mapping.
3. Inspect the two JSON lines. Confirm the start record matches the requested inputs, then use `run_summary.red` for rate, errors, and duration. `injected_failures` should match the deterministic `FAILURE_EVERY` schedule when the target is healthy; other failures are evidence of transport or target behavior.
4. Tear down the synthetic target: `docker compose -f compose.yaml down`.

This runbook uses synthetic traffic only, does not access ledgers, and does not establish production readiness or capacity.

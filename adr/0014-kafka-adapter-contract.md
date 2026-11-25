# ADR 0014: Minimal Kafka adapter contract

- **Status:** Accepted, P7 minimal slice
- **Scope:** `Spec: odd/tasks/phase-20-receiver-outage-recovery.md, T12-T15, S12-S14`

## Contract

The existing PostgreSQL `inter_network_outbox` remains the source of truth. Its persisted `serialized_envelope` is published verbatim as one versioned inter-network event:

- event type: `TRANSFER_REQUESTED`
- schema version: `1.0`, event version: `1`
- topic: `cholonet.inter-network.v1` (configurable as `KAFKA_TOPIC`)
- key: the envelope `event_id` (stable across retries and Kafka restart)
- consumer group: `cholonet-network-boundary` for the test consumer
- bootstrap servers: explicit `KAFKA_BOOTSTRAP_SERVERS`

The producer acknowledges an outbox item only after Kafka acknowledges the record. The test consumer acknowledges/commits its Kafka record only after the existing HMAC HTTP boundary has returned success from PostgreSQL-backed CholoNet intake. That boundary deduplicates by its existing transfer/idempotency identities; the consumer does not write ledger state directly.

`OUTBOX_TRANSPORT=http` remains the default and preserves the existing HTTP/HMAC adapter. Kafka is enabled only with explicit `OUTBOX_TRANSPORT=kafka`, `KAFKA_BOOTSTRAP_SERVERS`, and the Compose `events` profile.

## Demonstrated boundary and limits

The external runner `infra/tests/phase20_kafka_acceptance.sh` starts the `events` profile, explicitly sets `OUTBOX_TRANSPORT=kafka`, stops Kafka before creating one Costa outbox event, restarts Kafka, and runs `phase20-kafka-consumer`. The consumer checks the stable key/event identity, delivers the event twice through the existing boundary, commits only after persistence, and verifies exactly one durable `inter_network_message` row. This is at-least-once recovery evidence only; the one-row check is boundary deduplication evidence and is not an exactly-once claim. This ADR does **not** claim exactly-once processing, global ordering, a Kafka DLQ, backup/restore, RPO/RTO, or production availability.

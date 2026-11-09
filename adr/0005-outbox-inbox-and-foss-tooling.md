# ADR 0005: Outbox/inbox and FOSS local tooling

- Status: Accepted for Phase 0
- Date: 2026-11-09

## Decision

Services publish events through transactional outboxes and consume through durable idempotent inboxes. Kafka is optional in the first local profile. PostgreSQL is persistent truth; Redis 8+ with an AGPL distribution is cache/ephemeral state only. Local development uses Docker Engine or Podman, Compose, and open-source images; no Desktop, cloud, or paid SaaS dependency is required.

## Consequences

Event replay and reconciliation are first-class. Exact image tags and licenses must be pinned and reviewed before production-like profiles. Copyleft obligations are tracked in the dependency inventory.

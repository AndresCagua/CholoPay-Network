# Phase 1 backlog preview

1. Define bank schema and migration strategy.
2. Implement journal and balanced-entry constraints.
3. Implement account reservations with deterministic lock ordering.
4. Implement internal transfers with idempotency and payload hash conflict.
5. Add concurrent tests for insufficient funds and duplicate requests.
6. Add outbox/inbox tables without treating Kafka as truth.
7. Document API errors, permissions, and synthetic-data seed.
8. Run focused integration tests against ephemeral PostgreSQL.

Interbank settlement is intentionally deferred until the Phase 2 protocol ADRs answer the financial questions in `spec.md` section 19.

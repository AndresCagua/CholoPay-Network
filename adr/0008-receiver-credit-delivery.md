# ADR 0008: receiver credit delivery and completion

- Status: Accepted for the first delivery slice
- Date: 2026-11-09

## Context

Central settlement and receiver credit are different commits owned by different systems. CholoNet cannot write a bank's customer ledger. Delivery can be duplicated, delayed, reordered, or interrupted after settlement.

## Decision

After a central journal commits, CholoNet creates one durable `CREDIT_PENDING` delivery record keyed by `transfer_id` and receiver participant. An outbox worker may deliver the same command repeatedly. The receiver bank applies the command in its own local transaction and acknowledges with the same `transfer_id`, `delivery_id`, and `event_id`; its inbox/idempotency constraints prevent a second financial effect.

CholoNet accepts an acknowledgement only for an existing delivery whose transfer is `SETTLED` or `CREDIT_PENDING`. The first valid acknowledgement marks the delivery `APPLIED` and the transfer `COMPLETED` in one local transaction. Repeated identical acknowledgements return the existing result. A conflicting event or receiver identity is rejected and does not mutate state. Missing acknowledgements remain `CREDIT_PENDING` and are retried with bounded backoff; they are never declared failed by timeout alone.

## Invariants

- A settled transfer has at most one delivery record.
- Delivery transport is at-least-once; receiver financial effect is idempotent by durable identifiers.
- `COMPLETED` requires receiver evidence; `SETTLED` alone is insufficient.
- CholoNet never updates a bank customer ledger or invents receiver credit.
- A duplicate acknowledgement cannot create a second completion or journal.
- A refund is a new compensating operation and does not erase delivery history.

## Consequences

CholoNet owns delivery intent, attempts, and acknowledgement evidence. Each bank must later implement its own inbox and ledger application transaction. Reconciliation compares central settlement, delivery, receiver inbox, and bank ledger evidence.

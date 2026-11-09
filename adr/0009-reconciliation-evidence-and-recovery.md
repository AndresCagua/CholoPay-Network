# ADR 0009: reconciliation evidence and recovery

- Status: Accepted for the first reconciliation slice
- Date: 2026-11-09

## Context

Central settlement, receiver delivery, and bank completion are separate durable facts. A service can fail after settlement, messages can be delayed, and state projections can disagree. Reconciliation must detect differences without rewriting financial history.

## Decision

A reconciliation run records its start/end time and immutable difference rows. The runner compares:

- every central journal to its delivery record;
- every delivery to its instruction state;
- every completion state to an applied delivery;
- delivery/settlement identifiers and participant ownership.

Findings use explicit codes such as `SETTLED_WITHOUT_DELIVERY`, `DELIVERY_PENDING`, `APPLIED_WITHOUT_COMPLETED_STATE`, and `COMPLETED_WITHOUT_APPLIED_DELIVERY`. A clean run has zero difference rows. The runner is read-only with respect to financial and transfer state; remediation replays the same delivery ID or opens manual review and never edits a journal.

Runs and findings are idempotently queryable by run ID. Reconciliation evidence is append-only for the run. A later run may show that a difference was resolved, but it does not delete the earlier finding.

## Consequences

The network can distinguish missing work from financial loss, preserve audit history, and recover by replaying an existing delivery. Bank adapters must later export inbox/ledger evidence with the same transfer, delivery, and event identifiers.

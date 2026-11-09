# ADR 0007: central settlement ledger

- Status: Accepted for the first settlement slice
- Date: 2026-11-09

## Context

The three bank ledgers are independent. CholoNet needs a central ledger that records participant position changes without accessing bank tables. A transfer must not be considered settled because an HTTP request was accepted.

## Decision

Create one settlement account per participant and an immutable double-entry settlement journal. A settlement operation:

1. Locks source and destination settlement accounts in canonical UUID order.
2. Verifies the source available position covers the amount.
3. Creates exactly one journal identified by `transfer_id` and request hash.
4. Writes one debit entry for the source and one credit entry for the destination in USD cents.
5. Updates both position projections in the same PostgreSQL transaction.
6. Returns `SETTLED` only after the transaction commits.

A duplicate `transfer_id` or idempotency key returns the original journal. Reusing the key with a different request hash is rejected. Published journals and entries are immutable; returns are separate compensating journals. The network never writes a bank's customer ledger.

## Invariants

- Every journal has exactly two entries for this slice and their signed sum is zero.
- Source and destination positions conserve their aggregate sum.
- No source position becomes negative.
- One transfer ID creates at most one journal.
- A failed transaction leaves accounts and journal tables unchanged.
- `SETTLED` is derived only from the committed central journal.

## Consequences

The central ledger is authoritative only for interparticipant settlement positions, not customer balances. Receiver credit remains a later at-least-once bank operation. This slice does not implement liquidity funding, reversals, reconciliation jobs, or bank adapters; those require their own tests and evidence.

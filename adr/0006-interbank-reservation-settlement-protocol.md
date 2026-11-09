# ADR 0006: interbank reservation, settlement, and recovery protocol

- Status: Accepted as the Phase 2 protocol boundary; financial settlement implementation remains incremental
- Date: 2026-11-09

## Context

Costa, Sierra, and Oriente have independent databases. A distributed ACID transaction across banks does not exist, and the network must not write bank tables. A timeout can leave the outcome uncertain; retrying with a new business identifier could duplicate funds.

## Decision

Use a durable instruction protocol with four ownership phases:

1. **Prepare at issuer:** the issuer authenticates the request, validates the destination reference, and creates one durable reservation keyed by `transfer_id` and `idempotency_key`. The issuer reports `FUNDS_RESERVED` only after its local transaction commits.
2. **Central settlement:** CholoNet validates participant status, amount, currency, and unique instruction state. It locks the two settlement accounts in canonical participant order and commits one balanced debit/credit journal with a unique `transfer_id`. That commit alone establishes `SETTLED`.
3. **Receiver application:** the network delivers a durable at-least-once credit command to the receiver. The receiver inbox and account operation are one local transaction. A duplicate command returns the original result without a second credit.
4. **Issuer finalization and reconciliation:** after settlement confirmation, the issuer captures the reservation and records the final debit. If any acknowledgement is missing, retry the same command; never create a new transfer ID. A contradictory or unavailable result enters `UNKNOWN_PENDING_RECONCILIATION` and blocks automatic duplicate action.

The state contract is:

```text
CREATED -> VALIDATING -> FUNDS_RESERVED -> READY_FOR_SETTLEMENT
READY_FOR_SETTLEMENT -> SETTLED
SETTLED -> CREDIT_PENDING -> COMPLETED
READY_FOR_SETTLEMENT -> UNKNOWN_PENDING_RECONCILIATION
SETTLED -> UNKNOWN_PENDING_RECONCILIATION (only when evidence is incomplete)
UNKNOWN_PENDING_RECONCILIATION -> SETTLED | MANUAL_REVIEW
CREATED|VALIDATING|FUNDS_RESERVED -> REJECTED | EXPIRED (before settlement only)
SETTLED -> REFUNDED (derived from a new compensating operation)
```

## Invariants and recovery

- One transfer identifier produces at most one central settlement journal.
- Reservation, central settlement, and receiver credit are separately observable.
- The central journal is immutable; returns are compensating entries.
- A bank crash after reservation is recovered by querying the same transfer ID and replaying the same command, not by issuing a new instruction.
- A bank crash after central settlement leaves funds in a settled-but-not-yet-credited state until receiver application or manual reconciliation proves the outcome.
- Reconciliation compares issuer reservation/debit, network settlement entries, receiver inbox/credit, and correlation identifiers; it does not silently mutate history.

## Rejected alternatives

- Global two-phase commit across bank databases: unavailable across independent services and increases blocking/failure coupling.
- Exactly-once transport: not a realistic delivery guarantee; use at-least-once transport plus durable idempotency.
- HTTP 200 as settlement proof: an acknowledgement is not the central ledger commit.
- Repeating a timed-out transfer with a new ID: risks double settlement.

## Consequences

The network needs a persistent instruction state machine, settlement ledger, unique constraints, outbox/inbox records, and reconciliation jobs. Phase 2 initially implements the state boundary and idempotent instruction persistence; settlement effects and bank adapters follow as separate reviewed slices.

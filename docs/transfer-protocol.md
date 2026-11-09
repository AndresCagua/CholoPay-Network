# Transfer protocol and state machine

The protocol separates instruction acceptance, reservation, central settlement, and receiver credit. A successful HTTP response is not evidence of settlement.

```mermaid
stateDiagram-v2
    [*] --> CREATED
    CREATED --> VALIDATING
    VALIDATING --> REJECTED: invalid participant/key/amount
    VALIDATING --> FUNDS_RESERVED: sender reservation committed
    FUNDS_RESERVED --> READY_FOR_SETTLEMENT
    READY_FOR_SETTLEMENT --> SETTLED: central ledger commit
    READY_FOR_SETTLEMENT --> UNKNOWN_PENDING_RECONCILIATION: outcome uncertain
    SETTLED --> CREDIT_PENDING: receiver notification pending
    CREDIT_PENDING --> COMPLETED: receiver credit committed
    CREDIT_PENDING --> CREDIT_PENDING: durable retry
    UNKNOWN_PENDING_RECONCILIATION --> SETTLED: reconciliation confirms commit
    UNKNOWN_PENDING_RECONCILIATION --> MANUAL_REVIEW: evidence insufficient
    CREATED --> EXPIRED: before settlement deadline
    VALIDATING --> EXPIRED: timeout before reservation
    SETTLED --> REFUNDED: compensating return only
```

## Required guarantees

- Bank reservation and bank ledger effects use local ACID transactions and unique idempotency constraints.
- The network atomically records one settlement operation and balanced settlement entries.
- Receiver credit is an at-least-once message with inbox idempotency; exactly-once financial effect is achieved by persistent identifiers and reconciliation, not transport promises.
- A timeout after possible settlement yields `UNKNOWN_PENDING_RECONCILIATION`; clients must not retry with a new business identifier.
- Published entries are immutable. Returns are new compensating operations.

## ADR questions deferred to Phase 2

The exact recovery protocol after issuer failure and liquidity/funding policy require explicit ADR approval before interbank behavior is implemented.

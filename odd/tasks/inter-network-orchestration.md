# CholoPay Inter-Network orchestration

Source of truth for the first end-to-end routing/message slice. Read through `## Log` before working.

## Specs

1. **S1 — Route first:** transfer intake must validate source and destination participant membership before creating an inter-network message.
2. **S2 — Message boundary:** cross-network transfers create one durable `TRANSFER_REQUESTED` envelope carrying the transfer identifiers and route domains.
3. **S3 — Same-network behavior:** same-network transfers remain instructions without an inter-network message.
4. **S4 — Reliability:** repeated intake with the same idempotency key replays the existing instruction and message without duplicate effects.
5. **S5 — Finality:** accepting an instruction or message never means `SETTLED`; central settlement remains a separate operation.
6. **S6 — Non-claims:** the flow remains synthetic Docker-local behavior and does not connect to SIP, RPI, BCE, banks, or real wallets.

## Tasks

- T1 (S1-S3): Add orchestration service and internal HTTP intake endpoint. Route: inline. Commit: pending.
- T2 (S4-S5): Add idempotent replay and explicit response evidence without settlement side effects. Route: inline. Commit: pending.
- T3 (S6): Run Docker tests and record evidence. Route: inline. Commit: pending.

## Log

L1. User authorized continuation after the inter-network message-envelope slice.
L2. Routing, directory resolution, and message persistence exist separately. This slice composes them into an intake flow while preserving the central ledger as the only settlement authority.
L3. Added `POST /internal/v1/inter-network/transfers`. The service resolves participant domains before creating an instruction, creates no inter-network message for same-network routes, and persists one `TRANSFER_REQUESTED` message for cross-network routes using the transfer ID as the stable message ID. Repeated intake replays the instruction and message, and Docker tests confirm no intake path produces `SETTLED`.

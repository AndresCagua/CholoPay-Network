# ADR 0010: bank-owned receiver inbox and local credit

- Status: Accepted for the first adapter slice
- Date: 2026-11-09

## Context

CholoNet cannot write a participant bank's customer ledger. The receiver bank must apply a settled credit locally and provide durable evidence. Delivery is at-least-once, so duplicate commands are expected.

## Decision

`bank-core` owns a durable receiver inbox keyed by `delivery_id` and `event_id`. A receiver command is accepted only when the destination account belongs to the bank instance and the currency/amount match the command's immutable payload. The inbox insert, clearing-account debit, customer-account credit, and balanced local journal are one PostgreSQL transaction.

A duplicate delivery returns the original application result without another journal or balance change. A delivery identifier paired with a different payload, receiver account, or event is rejected. The bank returns an acknowledgement containing `delivery_id`, `event_id`, `transfer_id`, and the resulting local journal ID. CholoNet uses that evidence to mark the delivery applied.

The bank does not mark a customer payment complete from an HTTP redirect or a duplicate command. The clearing account is an internal synthetic account funded by the separately audited settlement adapter; this slice does not implement interbank funding.

## Consequences

Each bank instance has independent inbox and ledger tables. The adapter boundary is explicit and testable. Receiver application remains locally atomic even if the network retries the command.

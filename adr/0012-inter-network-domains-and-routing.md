# ADR 0012: CholoPay Inter-Network domains and routing

- Status: Accepted for the first two-network slice
- Date: 2026-11-09
- Reference basis: BCE Codificación JPRM-2025-006-M and BCE-GG-008-2025, consulted 2026-11-09

## Context

The expanded product simulates two independent payment networks: a banking network containing Costa, Sierra, and Oriente, and a fintech network containing Wallet A and Wallet B. CholoNet must route instructions without collapsing their ownership boundaries.

## Decision

Persist network domains and participant membership centrally. Each participant belongs to exactly one active domain at an instant. The route decision compares source and destination domains:

- same domain: `INTRA_NETWORK`, handled by that network's policy;
- different domains: `INTER_NETWORK`, sent through CholoNet inter-network coordination;
- missing, suspended, or mismatched membership: reject before reservation.

The route decision is advisory and does not establish settlement. The central settlement ledger remains the only source of `SETTLED`. Directory resolution, authorization, risk rules, compensation, and participant ledgers remain separate concerns.

## Consequences

The model demonstrates inter-network routing without claiming to implement the BCE SIP/RPI. It enables domain-specific credentials and policies, preserves participant isolation, and leaves room for distributed directory adapters. ISO 20022 is not claimed; events use CholoPay's versioned envelope until a concrete profile is approved.

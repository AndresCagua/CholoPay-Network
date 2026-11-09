# ADR 0004: Bank-owned customer onboarding

- Status: Accepted
- Date: 2026-11-09

## Decision

Customers open synthetic accounts at a participant bank. The central network registers and resolves CholoKeys and coordinates settlement but does not open retail accounts or own customer ledgers.

## Consequences

The operator console supervises institutions and authorized projections, not retail onboarding. A wallet session at one bank cannot authorize another bank.

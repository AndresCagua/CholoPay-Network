# Contributing

1. Use synthetic data only.
2. Read `spec.md` and the relevant ADR before changing a boundary or financial invariant.
3. Keep services independently runnable with FOSS tooling.
4. Add tests and update contracts/docs with behavior changes.
5. Never claim a payment is settled before the central settlement ledger confirms it.
6. Run `make config` and `make smoke` for infrastructure changes.

Use conventional commit messages when commits are requested. Do not add secrets, real identifiers, proprietary SDKs, or paid-service requirements.

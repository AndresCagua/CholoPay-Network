# CholoPay Network — Phase 4 Flutter and QR
Source of truth for the next bounded implementation slice. Read through `## Log` before working.

## Specs

1. **S1 — Entregables:** "wallet, CholoKey, QR estático/dinámico, movimientos y confirmación."
2. **S2 — Aceptación:** "pago móvil de A a B verificable en ledgers y UI."
3. **S3 — Boundary:** The wallet consumes bank/network APIs only; it never accesses another institution's database or writes a ledger.
4. **S4 — Reliability:** UI retries preserve the same idempotency key and must not create a second transfer.
5. **S5 — QR:** QR payloads are CholoPay-owned, versioned, synthetic, and are not Pix-compatible claims.
6. **S6 — Container rule:** Flutter builds and tests run inside Docker; no host SDK or host database is required.

## Tasks

- T1 (S1, S3, S5): Define wallet session, CholoKey, QR payload, and transfer-confirmation contracts. Route: inline. Commit: working tree.
- T2 (S1, S2, S4): Scaffold Flutter wallet with typed fixtures, account selection, QR flows, movement view, and idempotent confirmation state. Route: delegated by disjoint surfaces. Commit: pending.
- T3 (S2, S4): Add contract fixtures for payer/payee UI and duplicate-confirmation behavior without real ledger access. Route: inline. Commit: pending.
- T4 (S6): Add Docker-only Flutter build/test profile and record evidence. Route: inline. Commit: pending.

## Log

L1. User authorized continuation after Phase 3 web panels and Docker verification.
L2. Phase 4 is the next explicit phase in `spec.md`: Flutter wallet, CholoKey, static/dynamic QR, movements, confirmation, and verifiable mobile payment flow.
L3. Defined `docs/phase-4-wallet-contract.md` with synthetic participant-scoped sessions, minimized CholoKey resolution, CholoPay-owned versioned QR payloads, idempotent confirmation, movement projections, and explicit settlement/delivery states.
L4. Added the Docker-only Flutter wallet scaffold under `mobile/wallet`: typed local session/account/CholoKey/QR/transfer/movement fixtures, read-only wallet shell, account selection, widget tests, and explicit non-Pix-compatible scope. Docker widget tests pass. Commit: pending.

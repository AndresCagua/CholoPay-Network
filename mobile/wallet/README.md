# CholoPay Wallet scaffold

This is a minimal, read-only Flutter wallet shell for Phase 4 T2. It uses typed, local synthetic fixtures for a wallet session, accounts, CholoKeys, CholoPay QR payloads, transfers, and movements.

The QR format is CholoPay-owned and versioned. It is **not Pix-compatible**, and this scaffold makes no external-network compatibility claim.

The app does not access databases, call a backend, contain secrets, or confirm transfers. Transfer fixtures include the contract idempotency key so the later confirmation flow can reuse it without creating a new transfer identifier.

## Docker-only checks

Run from `mobile/wallet` (the host does not need a Flutter SDK):

```sh
docker build -t cholopay-wallet:test .
docker run --rm cholopay-wallet:test flutter test
docker run --rm cholopay-wallet:test flutter analyze
```

The image uses `ghcr.io/cirruslabs/flutter:3.24.5` and runs all Flutter commands inside Docker.

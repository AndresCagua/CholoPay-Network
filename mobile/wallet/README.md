# CholoPay Wallet scaffold

This is a minimal Flutter wallet shell for Phase 4 T2/T3. It uses typed, local synthetic fixtures for a wallet session, accounts, CholoKeys, CholoPay QR payloads, transfers, movements, and payer/payee confirmation.

The QR format is CholoPay-owned and versioned. It is **not Pix-compatible**, and this scaffold makes no external-network compatibility claim.

The app does not access databases, call a backend, or contain secrets. Confirmation is local-only: repeated confirmation replays the existing transfer using its contract idempotency key without creating a second transfer identifier. Results are read-only UI projections.

## Docker-only checks

Run from `mobile/wallet` (the host does not need a Flutter SDK):

```sh
docker build -t cholopay-wallet:test .
docker run --rm cholopay-wallet:test flutter test
docker run --rm cholopay-wallet:test flutter analyze
```

The image uses `ghcr.io/cirruslabs/flutter:3.24.5` and runs all Flutter commands inside Docker.

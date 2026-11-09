# ADR 0011: authenticated CholoNet-to-bank adapter

- Status: Accepted for the local adapter slice; mTLS remains the deployment boundary
- Date: 2026-11-09

## Context

CholoNet must deliver receiver-credit commands to a specific bank without allowing an unauthenticated caller to credit customer accounts. The production-like boundary requires mTLS, while the first Docker-only integration needs a deterministic application-level replay and signature check.

## Decision

The bank receiver endpoint requires both:

1. **Transport identity:** the deployment boundary is mTLS between CholoNet and each bank instance. Certificate issuance, rotation, and trust bundles remain deployment work.
2. **Message authentication:** every local adapter command carries `X-CholoPay-Request-Timestamp`, `X-CholoPay-Request-Id`, and `X-CholoPay-Signature`. The signature is HMAC-SHA256 over `timestamp + "." + request_id + "." + canonical payload`, using a per-bank secret supplied through Docker runtime configuration. The receiver rejects missing, malformed, wrong, or expired signatures and remembers request IDs for the replay window.

The receiver verifies the envelope before invoking the local inbox transaction. A valid duplicate delivery may replay its existing receipt, but a duplicate request ID with a different payload is rejected. Secrets never appear in logs or repository files. The HMAC layer is not a replacement for mTLS; it protects message integrity and supports deterministic local tests.

## Consequences

The adapter can be tested without external identity providers. Production deployment must add certificate validation, rotation, and network policy. The default Docker stack uses synthetic secrets only in test configuration and does not publish the endpoint to the host.

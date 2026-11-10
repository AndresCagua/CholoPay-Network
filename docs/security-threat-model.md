# Modelo de amenazas de seguridad (STRIDE)

Alcance: arquitectura de la fase 0 y entorno de desarrollo local. Todas las identidades y el dinero son sintéticos.

| Amenaza | Activo / límite | Mitigaciones | Evidencia requerida |
|---|---|---|---|
| Double spend / tampered amount | Bank ledger and settlement ledger | Integer cents, row locks or conditional updates, balanced entries, unique instruction IDs, reconciliation | Concurrent invariant tests |
| Replay / duplicate event | Transfer and webhook inboxes | Idempotency key + payload hash, unique event IDs, TTL only for cache hints, durable inbox | Duplicate delivery tests |
| Spoofed participant | Bank ↔ network mTLS boundary | Service identity, certificate rotation, scoped authorization, audit events | Authn/authz contract tests |
| Recipient substitution | CholoKey resolution and QR | Minimized resolution, explicit preview/confirmation, signed QR profile, expiry | Destination confirmation tests |
| Forged webhook | Gateway ↔ merchant/biller | HMAC timestamp, replay window, endpoint secret hash, delivery audit and DLQ | Signature and replay tests |
| Key/privacy enumeration | Directory | Rate limits, context-bound lookup, masking, audit, no bulk endpoint | Abuse and access tests |
| Insider adjustment | Operator boundary | Roles, separation of duties, append-only audit, no direct customer ledger access | Authorization tests |
| Denial of service | Public API and database | Timeouts, bounded concurrency, rate limits, backpressure, resource limits | Load and fault tests |
| Event loss/reordering | Outbox/Kafka | Transactional outbox, inbox idempotency, versioned events, replay and reconciliation | Restart/reordering tests |
| Clock/replay edge cases | Tokens, signatures, expiry | UTC timestamps, bounded clock skew, monotonic state transitions | Time-boundary tests |

## Supuestos y no afirmaciones

La fase 0 no entrega TLS, emisión de identidades, almacenamiento de secretos ni operaciones de producción. No se afirma ninguna certificación de cumplimiento. Docker/Compose es infraestructura de desarrollo local, no un límite de seguridad de producción.

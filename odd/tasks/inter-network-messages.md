# CholoPay Inter-Network messages

Source of truth for the inter-network message-envelope slice. Read through `## Log` before working.

## Specs

1. **S1 — Envelope:** "Los eventos propios mantendrán `schema_version`, `message_type`, `correlation_id`, `causation_id`, `idempotency_key` y timestamps UTC."
2. **S2 — Boundaries:** "CholoNet Inter-Network enruta instrucciones entre dominios y coordina la liquidación central simulada; no se convierte en una cuenta minorista ni escribe en los ledgers internos de los participantes."
3. **S3 — Reliability:** transport is at-least-once and duplicate message IDs must not create duplicate financial effects.
4. **S4 — Security:** messages remain internal Docker traffic and must carry sender and destination network identity for validation.
5. **S5 — Non-claims:** "No conectarse al SIP, RPI, BCE ni redes reales." and no ISO 20022 compatibility claim in this phase.
6. **S6 — Container rule:** "Todo debe ejecutarse dentro de contenedores".

## Tasks

- T1 (S1-S2): Define and persist versioned inter-network message envelopes. Route: inline. Commit: pending.
- T2 (S3-S5): Add idempotent acceptance and network-boundary validation through the internal HTTP API. Route: inline. Commit: pending.
- T3 (S6): Run Docker PostgreSQL tests and record evidence. Route: inline. Commit: pending.

## Log

L1. User authorized continuation after the CholoKey directory slice.
L2. Existing routing and directory resolution are deterministic but do not yet persist a message boundary. This slice adds an educational envelope for inter-network coordination without implementing real BCE/SIP/RPI connectivity or ISO 20022.
L3. Added V7 persistence and the internal message endpoint. Envelopes carry network identities, schema/message metadata, correlation and causation IDs, idempotency, payload hash, JSON payload, and UTC creation time. Docker tests prove first acceptance, identical replay, conflicting duplicate rejection, inactive-network rejection, and HTTP replay behavior.

# CholoPay Inter-Network directories

Source of truth for the distributed-directory slice. Read through `## Log` before working.

## Specs

1. **S1 — Directory isolation:** "Cada una mantiene sus propios participantes, directorio distribuido simulado, credenciales, políticas de acceso, mensajes y proyecciones."
2. **S2 — Key lookup:** "resolución de CholoKeys mediante directorios distribuidos simulados".
3. **S3 — Route safety:** "Una ruta no puede resolverse si el participante está suspendido, no existe o pertenece a un dominio ambiguo."
4. **S4 — Evidence:** "Los eventos propios mantendrán `schema_version`, `message_type`, `correlation_id`, `causation_id`, `idempotency_key` y timestamps UTC."
5. **S5 — Non-claims:** "No conectarse al SIP, RPI, BCE ni redes reales." y "CholoPay no declarará compatibilidad ISO 20022 en esta fase."
6. **S6 — Container rule:** "Todo debe ejecutarse dentro de contenedores".

## Tasks

- T1 (S1-S3): Add isolated directory/key records and deterministic lookup API. Route: inline. Commit: pending.
- T2 (S4-S5): Add versioned directory resolution evidence and explicit non-real-network documentation. Route: inline. Commit: pending.
- T3 (S6): Run Docker PostgreSQL tests and record evidence. Route: inline. Commit: pending.

## Log

L1. User authorized continuation after the two-network CholoPay Inter-Network slice.
L2. The previous slice established banking and fintech network domains, participant membership, and same-network/cross-network route selection. This slice adds simulated distributed directory resolution without claiming BCE SIP/RPI integration or ISO 20022 compatibility.
L3. Added per-network CholoKey records with participant ownership, account reference, status, version, and a uniqueness boundary scoped to each network. The resolver emits `DIRECTORY_KEY_RESOLVED` evidence with schema version, message type, correlation/causation IDs, idempotency key, and UTC resolution time. Docker network tests cover successful lookup, network isolation, suspension rejection, and the public HTTP resolver.

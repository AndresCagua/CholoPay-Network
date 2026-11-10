# CholoPay Network — Phase 3 web panels
Source of truth for the next bounded implementation slice. Read through `## Log` before working.

## Specs

1. **S1 — Entregables:** "operador React, banco Costa Next.js, banco Sierra Angular, banco Oriente React/Vite, autenticación y roles."
2. **S2 — Aceptación:** "visualizar misma transferencia desde las instituciones implicadas y red, cada una con permisos correctos."
3. **S3 — Boundary:** "Los tres utilizan el mismo producto `cholopay-bank-core` parametrizado, pero se despliegan como tres instancias autónomas con identidad criptográfica, secretos, cuentas, almacenamiento y configuración independientes."
4. **S4 — Shared state:** "Los usuarios pueden usar Flutter y la web del banco para operar las mismas cuentas y consultar los mismos movimientos. Cambios reflejados por consultas al backend, SSE/WebSocket para actualización opcional y revalidación tras reconexión. Una sesión en Banco Costa no autoriza por sí misma acceso a Banco Sierra."
5. **S5 — Existing contracts:** "Todo contrato o evento nuevo deberá incluir `schema_version`, identificador de correlación, `idempotency_key`, timestamp UTC y trazabilidad."
6. **S6 — Container rule:** All new web builds and tests run inside Docker; no host toolchains or host databases.

## Tasks

- T1 (S1, S2): Define web-panel boundaries, auth/role contract, and shared transfer read model. Route: inline. Commit: `912b1a1`.
- T2 (S1, S3): Scaffold the operator React panel and three isolated bank web applications. Route: delegated by disjoint app surfaces. Commit: `912b1a1`.
- T3 (S2, S4, S5): Add transfer visualization and contract fixtures against backend APIs without direct database access. Route: delegated by disjoint app surfaces. Commit: `912b1a1`.
- T4 (S6): Add Docker-only build/test profiles and record evidence. Route: inline. Commit: `912b1a1`.

## Log

L1. User authorized continuation to the next specification point after the Phase 2 network and inter-network slices.
L2. Phase 3 is the next explicit phase in `spec.md`: operator React, Banco Costa Next.js, Banco Sierra Angular, Banco Oriente React/Vite, authentication, roles, and cross-institution transfer visibility.
L3. Defined `docs/phase-3-web-contract.md`: four isolated web surfaces, synthetic Docker-only bearer identities, participant-scoped roles, backend-only authorization, shared transfer read model, explicit state rendering, polling/revalidation before SSE/WebSocket, and mandatory metadata for new contracts.
L4. Added Docker-only scaffolds and local typed read-only transfer fixtures for operator, Costa, Sierra, and Oriente. Each image built successfully; no web surface accesses a database or backend yet. Commit: `912b1a1`.
L5. The parent verification rebuilt all four web images with Docker and `git diff --check` passed. No host ports were published and no web build required a host toolchain.

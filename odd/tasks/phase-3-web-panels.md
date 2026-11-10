# CholoPay Network — Paneles web de la fase 3
Fuente de verdad del siguiente slice acotado de implementación. Lee hasta `## Log` antes de trabajar.

## Specs

1. **S1 — Entregables:** "operador React, banco Costa Next.js, banco Sierra Angular, banco Oriente React/Vite, autenticación y roles."
2. **S2 — Aceptación:** "visualizar misma transferencia desde las instituciones implicadas y red, cada una con permisos correctos."
3. **S3 — Límite:** "Los tres utilizan el mismo producto `cholopay-bank-core` parametrizado, pero se despliegan como tres instancias autónomas con identidad criptográfica, secretos, cuentas, almacenamiento y configuración independientes."
4. **S4 — Shared state:** "Los usuarios pueden usar Flutter y la web del banco para operar las mismas cuentas y consultar los mismos movimientos. Cambios reflejados por consultas al backend, SSE/WebSocket para actualización opcional y revalidación tras reconexión. Una sesión en Banco Costa no autoriza por sí misma acceso a Banco Sierra."
5. **S5 — Existing contracts:** "Todo contrato o evento nuevo deberá incluir `schema_version`, identificador de correlación, `idempotency_key`, timestamp UTC y trazabilidad."
6. **S6 — Container rule:** Todas las compilaciones y pruebas web nuevas se ejecutan dentro de Docker; no se requieren toolchains ni bases de datos del host.

## Tasks

- T1 (S1, S2): Definir límites de los paneles web, contrato de autenticación/roles y modelo de lectura compartido de transferencias. Ruta: inline. Commit: `912b1a1`.
- T2 (S1, S3): Crear el esqueleto del panel React del operador y de las tres aplicaciones web bancarias aisladas. Ruta: delegada a superficies de aplicaciones disjuntas. Commit: `912b1a1`.
- T3 (S2, S4, S5): Agregar visualización de transferencias y fixtures contractuales contra las API backend sin acceso directo a bases de datos. Ruta: delegada a superficies de aplicaciones disjuntas. Commit: `912b1a1`.
- T4 (S6): Agregar perfiles de compilación/pruebas exclusivos de Docker y registrar la evidencia. Ruta: inline. Commit: `912b1a1`.

## Log

L1. La persona usuaria autorizó continuar con el siguiente punto de la especificación después de los slices de red y de interredes de la fase 2.
L2. La fase 3 es la siguiente fase explícita de `spec.md`: operador React, Banco Costa Next.js, Banco Sierra Angular, Banco Oriente React/Vite, autenticación, roles y visibilidad de transferencias entre instituciones.
L3. Se definió `docs/phase-3-web-contract.md`: cuatro superficies web aisladas, identidades bearer sintéticas exclusivas de Docker, roles delimitados por participante, autorización únicamente en backend, modelo compartido de lectura de transferencias, representación explícita de estados, polling/revalidación antes de SSE/WebSocket y metadatos obligatorios para contratos nuevos.
L4. Se agregaron esqueletos exclusivos de Docker y fixtures locales tipados de transferencias de solo lectura para operador, Costa, Sierra y Oriente. Cada imagen se compiló correctamente; ninguna superficie web accede todavía a una base de datos o backend. Commit: `912b1a1`.
L5. La verificación del agente principal reconstruyó las cuatro imágenes web con Docker y `git diff --check` pasó. No se publicaron puertos del host y ninguna compilación web requirió un toolchain del host.

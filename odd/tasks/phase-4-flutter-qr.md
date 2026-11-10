# CholoPay Network — Flutter y QR de la fase 4
Fuente de verdad del siguiente slice acotado de implementación. Lee hasta `## Log` antes de trabajar.

## Specs

1. **S1 — Entregables:** "wallet, CholoKey, QR estático/dinámico, movimientos y confirmación."
2. **S2 — Aceptación:** "pago móvil de A a B verificable en ledgers y UI."
3. **S3 — Límite:** La billetera consume únicamente API del banco/la red; nunca accede a la base de otra institución ni escribe en un ledger.
4. **S4 — Fiabilidad:** Los reintentos de la interfaz conservan la misma clave de idempotencia y no deben crear una segunda transferencia.
5. **S5 — QR:** Los payloads QR son propios de CholoPay, versionados y sintéticos, y no constituyen afirmaciones de compatibilidad con Pix.
6. **S6 — Regla de contenedores:** Las compilaciones y pruebas de Flutter se ejecutan dentro de Docker; no se requieren SDK ni bases de datos del host.

## Tasks

- T1 (S1, S3, S5): Definir los contratos de sesión de billetera, CholoKey, payload QR y confirmación de transferencias. Ruta: inline. Commit: working tree.
- T2 (S1, S2, S4): Crear el esqueleto de la billetera Flutter con fixtures tipados, selección de cuenta, flujos QR, vista de movimientos y estado de confirmación idempotente. Ruta: delegada a superficies disjuntas. Commit: `3dcc78d`.
- T3 (S2, S4): Agregar fixtures contractuales para las interfaces de pagador/receptor y el comportamiento de confirmación duplicada sin acceso a un ledger real. Ruta: delegada a la superficie de billetera móvil. Commit: `9c3c8f2`.
- T4 (S6): Agregar un perfil de compilación/pruebas de Flutter exclusivo de Docker y registrar la evidencia. Ruta: inline. Commit: `3dcc78d`.

## Log

L1. La persona usuaria autorizó continuar después de los paneles web de la fase 3 y la verificación Docker.
L2. La fase 4 es la siguiente fase explícita de `spec.md`: billetera Flutter, CholoKey, QR estático/dinámico, movimientos, confirmación y flujo de pago móvil verificable.
L3. Se definió `docs/phase-4-wallet-contract.md` con sesiones sintéticas delimitadas por participante, resolución minimizada de CholoKey, payloads QR versionados propios de CholoPay, confirmación idempotente, proyecciones de movimientos y estados explícitos de liquidación/entrega.
L4. Se agregó el esqueleto de billetera Flutter exclusivo de Docker bajo `mobile/wallet`: fixtures locales tipados de sesión/cuenta/CholoKey/QR/transferencia/movimiento, shell de billetera de solo lectura, selección de cuenta, pruebas de widgets y alcance explícito no compatible con Pix. Las pruebas de widgets Docker pasan. Commit: `3dcc78d`.
L5. El agente principal reconstruyó la imagen Flutter y ejecutó la suite de widgets dentro de Docker; ambas pruebas de widgets pasaron con código de salida 0.
L6. Se agregó confirmación local de pagador/receptor usando la clave de idempotencia estable, estado de confirmación visible, replay duplicado sin una segunda transferencia y cobertura pública de widgets. Las pruebas Docker pasaron las 4 pruebas y `flutter analyze` no informó problemas. Commit: `9c3c8f2`.

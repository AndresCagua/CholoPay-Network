# Phase 10 — Gobernanza de CholoKey
Fuente: `spec.md`, secciones 7, 23.2, 23.3, 26 y 27.
Scope: registro atómico y límites sintéticos del directorio; no portabilidad completa, reclamaciones ni datos personales reales en este slice.

## Specs

1. **S1 — Vínculo único**
   > Cada clave activa y única se vincula exactamente a una cuenta transaccional y a un participante en cada instante.

2. **S2 — Límites por cuenta**
   > hasta cinco claves por cuenta de persona natural y veinte por cuenta de persona jurídica
   > El límite es **por cuenta**, no por persona ni por banco.

3. **S3 — Tipos y privacidad**
   > documento de identidad ficticio ... email ficticio verificado, teléfono ficticio verificado ... y clave aleatoria UUID generada por el directorio central.
   > No registrar datos personales reales en demos ni semillas.

4. **S4 — Registro seguro**
   > reserva atómica de clave; confirmación central; publicación al banco tras éxito. Reintentos idempotentes.

5. **S5 — Aceptación**
   > rechazar duplicados globales y excesos de 5/20 por cuenta.

## Tasks

- T1 (S1-S4): Añadir límites y unicidad global sintética a la migración/store del directorio, con registro idempotente. Route: inline after exploration. Commit: pending.
- T2 (S1-S5): Añadir pruebas de concurrencia, duplicado, límite y replay por interfaz pública. Route: inline. Commit: pending.
- T3 (S1-S5): Actualizar documentación y evidencia Docker. Route: inline. Commit: pending.

## Log

L1. Usuario: "continua con el siguiente paso"
L2. Se prioriza gobernanza básica de CholoKey antes de portabilidad, reclamaciones y ampliación completa de portales.
L3. El directorio ahora registra `account_kind`, limita claves activas a 5 por cuenta personal y 20 por cuenta empresarial, y aplica unicidad global para identidades activas EMAIL/PHONE/DOCUMENT; RANDOM conserva UUID por registro. Se agregaron errores explícitos de conflicto y límite.
L4. Evidencia Docker: `docker compose config` PASS; suite Go de network PASS, incluyendo directory; gofmt y `git diff --check` PASS. La prueba cubre aislamiento, duplicado global y límite personal.

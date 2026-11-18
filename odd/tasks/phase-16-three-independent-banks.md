# Phase 16 — Tres bancos independientes en Docker

Fuente: `docs/spec-pending-work.md`, P1; `spec.md` y contrato web de Phase 3.
Alcance: ejecutar tres instancias autónomas de `bank-core` con PostgreSQL, identidades, secretos, fixtures y pruebas de aislamiento independientes. No implementa todavía toda la transferencia interbancaria de P2 ni las proyecciones reales de las webs.

## Specs

1. **S1 — Solicitud del usuario:**
   > Los 10 puntos que me marcaste me parecen importantes, anótalos en la parte donde documentas que esos son los puntos pendientes y vamos resolviendo uno por uno

2. **S2 — Tres bancos independientes:**
   > Los tres utilizan el mismo producto `cholopay-bank-core` parametrizado, pero se despliegan como tres instancias autónomas con identidad criptográfica, secretos, cuentas, almacenamiento y configuración independientes.

3. **S3 — Aislamiento:**
   > Los bancos, redes y ledgers deben permanecer aislados; CholoNet no escribe directamente en tablas bancarias.

4. **S4 — Docker-only:**
   > Ejecutar builds, tests, migraciones, bases y servicios dentro de Docker.

5. **S5 — Alcance sintético:**
   > La liquidación productiva, fondeo real, redes financieras externas, mTLS productivo y compatibilidad normativa real permanecen fuera de alcance.

## Tasks

- T1 (S2-S4): parametrizar Compose con tres servicios independientes de `bank-core`, cada uno conectado únicamente a su PostgreSQL, con identidad, token, secreto HMAC y cuentas de compensación propios. Ruta: inline. Commit: `991505a`.
- T2 (S2-S3): añadir fixtures deterministas diferenciados por participante sin cruzar cuentas, balances ni ledgers. Ruta: inline. Commit: `991505a`.
- T3 (S2-S4): añadir una prueba pública Docker-only que levante las tres instancias, consulte identidad y compruebe rechazo de acceso cruzado. Ruta: inline. Commit: `991505a`.
- T4 (S2-S5): actualizar Compose, documentación y evidencia; ejecutar checks Docker y registrar límites sin implementar todavía P2. Ruta: inline. Commit: `991505a`.

## Log

- L1. Usuario: "Los 10 puntos que me marcaste me parecen importantes, anótalos en la parte donde documentas que esos son los puntos pendientes y vamos resolviendo uno por uno"
- L2. Se selecciona P1 como primer slice: tres instancias independientes de `bank-core` con bases, identidades, secretos, fixtures y prueba de aislamiento.
- L3. La exploración confirmó que `infra/compose.yaml` solo ejecuta `bank-core` para Costa aunque ya existen `costa-db`, `sierra-db` y `oriente-db`; la red todavía fija `BANK_RECEIVER_CREDIT_URL` a un único receptor. El routing completo por participante se mantiene para P2.
- L4. Se añadieron `bank-core-sierra` y `bank-core-oriente` con PostgreSQL, participantes, tokens, secretos HMAC, URLs y clearing accounts independientes; se conservaron `bank-core` y los contratos existentes de Costa.
- L5. `V9__bank_participant_fixtures.sql` añadió fixtures deterministas de fuente eléctrica y clearing para Sierra y Oriente. `BankIdentityController` expone una identidad interna protegida por el token de cada banco para la aceptación Docker-only.
- L6. La primera ejecución falló por una carrera de disponibilidad del runner (`ConnectionRefusedError`); el runner se corrigió para tratar la indisponibilidad como transitoria. La segunda ejecución `job-2` pasó: `PHASE16 PASS: tres identidades bancarias aisladas y tokens cruzados rechazados`.
- L7. La suite completa `bank-core-test` ejecutó en Docker como `job-3` y terminó con código 0, incluyendo el test del controlador de identidad y las pruebas existentes. `docker compose --profile phase16 config --quiet` y `git diff --check` también pasaron.
- L8. ASSESS sobre `main..1b42e9d` clasificó el candidato como riesgo medio, con `reviewDue=false` por estar bajo el presupuesto configurado. Conforme a la política RDD, no se inició una review nativa adicional; la verificación Docker y la review externa no son equivalentes.

# Phase 18 — Aceptación E2E interbancaria

Fuente: `docs/spec-pending-work.md`, P2; Phase 17 y ADRs 0006, 0008 y 0009.
Alcance: demostrar dentro de Docker el recorrido de una liquidación de CholoNet hacia Sierra y Oriente, con receiver inbox, aplicación idempotente, retry ante indisponibilidad y estado de transferencia completado. No añade todavía las APIs públicas de lectura de balances/journals de P3.

## Specs

1. **S1 — Solicitud del usuario:**
   > continua porfa

2. **S2 — Recorrido:**
   > Costa → CholoNet → Sierra/Oriente → receiver inbox → journal receptor → retry ante caída → replay → reconciliación

3. **S3 — Aislamiento:**
   > La red no accede a tablas internas bancarias.

4. **S4 — Idempotencia:**
   > Los retries conservan `delivery_id` y `event_id`.

5. **S5 — Docker-only:**
   > Ejecutar builds, tests, migraciones, bases y servicios dentro de Docker.

6. **S6 — Límites:**
   > La liquidación productiva, fondeo real, redes financieras externas, mTLS productivo y compatibilidad normativa real permanecen fuera de alcance.

## Tasks

- T1 (S2-S5): añadir fixtures de cuentas receptoras y bootstrap sintético de participantes/cuentas de liquidación para la aceptación Docker. Ruta: inline. Commit: `499377c`.
- T2 (S2-S4): crear runner E2E que cree instrucciones, avance estados, liquide hacia Sierra/Oriente y verifique entrega/replay/retry mediante interfaces HTTP y evidencia acotada del entorno de prueba. Ruta: inline. Commit: `499377c`.
- T3 (S2-S6): ejecutar la aceptación en Compose aislado, documentar resultados y separar explícitamente lo que queda para P3. Ruta: inline. Commit: `499377c`.
- T4 (S2-S6): revisar diff, crear work unit y evaluar RDD. Ruta: inline. Commit: `499377c`.

## Log

- L1. Usuario: "continua porfa"
- L2. Phase 17 completó el routing por participante, pero P2 permaneció parcial porque no existía una aceptación Costa→Sierra/Oriente.
- L3. V10 añadió cuentas receptoras y saldo de clearing para Sierra/Oriente. `phase18-seed` registra tres participantes y tres cuentas de settlement en la base de CholoNet.
- L4. El primer runner detectó dos problemas de integración: la firma Go usaba hexadecimal mientras bank-core verifica Base64, y la aceptación buscaba `state` mientras el endpoint existente serializa `State`. Se corrigieron el dispatcher, el replay del runner y la compatibilidad de lectura del estado.
- L5. `job-11` pasó dentro de Docker: `PHASE18 PASS: settlement y entrega idempotente completados para sierra,oriente`. Verificó `COMPLETED`, `APPLIED`, delivery persistente, `attempt_count >= 1`, replay firmado y un único inbox/journal por entrega.
- L6. La caída real de un receptor no se inyectó durante este slice; los retries y leases continúan cubiertos por la suite de red. Las APIs públicas de balances/journals permanecen en P3.
- L7. La suite de red posterior al cambio HMAC pasó en Docker como `job-12`, con todos los paquetes en `ok`. `git diff --check` y Compose config también pasan.
- L8. ASSESS sobre `f25851d..499377c` reportó riesgo medio, `reviewDue=false`, writer profile `large` y plan de auto-verificación; no se inició review nativa adicional porque no estaba debida y el usuario no la solicitó explícitamente.
- L9. El work unit quedó en `499377c 20261118: acepta entrega interbancaria E2E`. La suite posterior `job-12` pasó todos los paquetes de red en Docker. El registro de commit y ASSESS se incorpora antes de integrar la rama a `main`.
- L10. La review RDD `review-5232591bb22a1a4c` detectó el hallazgo crítico `R3-cross-bank-fixtures`: las migraciones compartidas sembraban fixtures de Sierra y Oriente en todas las bases. La autoridad terminó `corrected_candidate_unavailable`; no se reutiliza.
- L11. La corrección usa el placeholder Flyway `participant_id` derivado de `PARTICIPANT_ID` y condiciona V9/V10 por banco. El runner verifica que la cuenta receptora del otro banco no exista. `job-13` pasó con `PHASE18 PASS` y aislamiento confirmado.

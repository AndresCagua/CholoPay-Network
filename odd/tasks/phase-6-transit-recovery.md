# Phase 6 — Recuperación durable de transit sintético
Fuente: `spec.md`, secciones de facturadores, obligaciones e idempotencia.
Alcance: persistir y recuperar el flujo sintético existente de transit; no conectar bancos ni ampliar settlement.

## Specs

1. **S1 — Estado durable:** registros, multas, obligaciones, payment intents, eventos aplicados y constancias de transit sobreviven al reinicio del contenedor mediante PostgreSQL aislado.
2. **S2 — Contrato compatible:** conservar rutas, payloads, estados, idempotency-key, conflictos de fingerprint, HMAC de eventos, receipts y constancias existentes.
3. **S3 — Idempotencia transaccional:** creación registro+obligación, multa+obligación, reserva de intent y aplicación de evento deben ser transaccionales; replay idéntico no duplica efectos.
4. **S4 — Recuperación Docker:** aceptación crea registro/multa, paga sintéticamente, recrea solo `transit` conservando el volumen y verifica IDs, estados, replay y receipt.
5. **S5 — Límites:** solo transit sintético; fuera gateway, bancos, ledgers, settlement global, dinero real, certificados oficiales y claims productivos.

## Tasks

- T1 [S1-S3] Implementar repositorio PostgreSQL aislado para el estado transit conservando la API existente. Ruta: transit. Commit: pendiente.
- T2 [S4-S5] Añadir transit-db, volumen, aceptación Docker de reinicio/replay y runbook. Ruta: Compose/infra/docs. Commit: pendiente.
- T3 [S1-S5] Ejecutar checks, crear work unit y evaluar RDD. Ruta: Docker + Git. Commit: pendiente.

## Log

- L1. El usuario pidió continuar después del cierre de utilities electricidad P6.
- L2. La exploración confirmó que transit conserva registros, multas, obligaciones, intents, eventos y constancias en memoria; se eligió persistir el aggregate transit completo como siguiente slice coherente.
- L3. T1-T2 implementados: PostgreSQL aislado, transacciones para pares y aplicación de eventos, aceptación Docker-only de replay y recreación de `transit`, documentación y límites sintéticos. T3 (checks, work unit y RDD) queda pendiente.
- L4. La suite transit ejecuta 6 tests, 4 pasan y 2 fallan también contra `HEAD` sin este slice: las expectativas de listado esperan solo la multa aunque la implementación base devuelve además la obligación de registro. El slice no altera esa semántica preexistente.

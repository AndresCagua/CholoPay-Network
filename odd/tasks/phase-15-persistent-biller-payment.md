# Phase 15 — Pago persistente de obligaciones respaldado por ledger
Fuente: brecha prioritaria de `spec.md` §§6, 8.1, 8.2, 8.4, 9, 11 y 24; primer slice limitado a electricidad en Banco Costa.
Alcance: convertir el flujo actual sintético gateway→facturador en un intento de pago persistente en bank-core, con débito ACID idempotente, outbox y aplicación eventual; todo sigue siendo educativo y Docker-only.

## Specs

1. **S1 — Solicitud del usuario:** "continua con los specs, necesito ya terminar este proyecto".
2. **S2 — Límite financiero de `spec.md`:** "No procesar dinero real".
3. **S3 — Ledger:** "Toda modificación financiera local ocurre dentro de transacciones ACID con restricciones, índices únicos y control de concurrencia explícito.".
4. **S4 — Idempotencia:** "Una misma instrucción de negocio no produce dos liquidaciones.".
5. **S5 — Estados inciertos:** "Si no se puede establecer con certeza el estado de liquidación, no repetir la operación con nuevo ID ni informar «fallida» sin reconciliar; usar `UNKNOWN_PENDING_RECONCILIATION`.".
6. **S6 — Aceptación del slice:** iniciar un pago eléctrico desde Banco Costa, debitar una cuenta sintética una sola vez, conservar el resultado incierto/pending si el facturador no está disponible, y aplicar/reproducir el evento sin doble débito tras recuperación.

## Tasks

- T1 (S2-S5): Definir el contrato persistente de payment intent de obligación en bank-core, su idempotencia y estados, sin afirmar settlement real. Ruta: bank-core. Commit: `dac2b4a`.
- T2 (S3-S5): Implementar débito ACID idempotente y outbox de aplicación al facturador, conservando `PENDING`/`UNKNOWN_PENDING_RECONCILIATION` ante indisponibilidad. Ruta: bank-core/gateway. Commit: `dac2b4a`.
- T3 (S3-S6): Añadir pruebas Docker públicas de éxito, replay, conflicto, saldo insuficiente, caída del facturador y recuperación sin doble débito. Ruta: bank-core/gateway/utilities. Commit: `dac2b4a` (cobertura implementada para éxito, replay, conflicto y saldo insuficiente; aplicación/recuperación queda en el siguiente slice).
- T4 (S1-S6): Actualizar OpenAPI, documentación y evidencia Docker; registrar límites sintéticos. Ruta: inline. Commit: `dac2b4a`.

## Log

L1. Usuario: "HIJO DE PUTA REVISA !!!!!!!1 Y TERMINA LO QUE FALTA !!!!!!!!!!"
L2. La auditoría completa de `spec.md` identificó como brecha prioritaria el primer recorrido de pago de obligación respaldado por ledger; el T7 existente solo prueba estados sintéticos de gateway/facturadores y no modifica un ledger bancario.
L3. El slice se limita a electricidad en Banco Costa para no inventar todavía semánticas de tres bancos, fondeo, finality productiva ni settlement real.

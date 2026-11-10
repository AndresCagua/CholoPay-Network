# Phase 6 — Facturadores simulados: servicios y tránsito

Fuente: `spec.md`, secciones 3, 4, 8.1, 8.2 y 8.3. Incremento autorizado por el usuario: continuar con el resto de los specs después de abandonar la revisión bloqueada.
Scope: dos superficies independientes, sintéticas y Docker-only; no modificar infraestructura externa ni conectar dinero real.

## Specs

1. **S1 — Repositorios de facturadores**
   > `cholopay-utilities` | Java + Spring Boot, PostgreSQL | Empresa ficticia unificada de electricidad, agua, internet y telefonía móvil: contratos, facturación y cobranza.
   > `cholopay-transit` | Java + Spring Boot, PostgreSQL | Autoridad de tránsito ficticia: matrículas, multas, obligaciones y constancias.

2. **S2 — Alcance del facturador**
   > Empresa multiservicios (simulada) | Emitir facturas ficticias, consultar cobros propios y registrar su cumplimiento | No debitar cuentas de clientes o liquidar sin la red.
   > Autoridad de Tránsito (simulada) | Emitir obligaciones ficticias, consultar pagos y emitir constancias simuladas | No acceder a cuentas privadas o actuar como autoridad real.

3. **S3 — Wallet**
   > consulta y pago de facturas de servicios básicos, matrículas y multas;

4. **S4 — Límites**
   > No procesar dinero real, conectarse a SPI/DICT reales, simular cumplimiento regulatorio como certificación, copiar interfaces o marcas registradas, almacenar datos personales reales, operar como banco.

## Tasks

- T1 (S1, S2): Crear utilities API Docker-only con contratos sintéticos de contratos, facturas, consulta y estado de cobranza. Route: delegated by utilities surface. Commit: `ef14f3e`.
- T2 (S1, S2): Crear transit API Docker-only con matrículas, multas, obligaciones, consulta y constancias sintéticas. Route: delegated by transit surface. Commit: `ef14f3e`.
- T3 (S2, S3): Añadir idempotencia y frontera explícita: los facturadores no debitan cuentas ni escriben ledgers; dejar endpoints de pago como intentos pendientes para integración posterior. Route: delegated by both surfaces. Commit: `ef14f3e`.
- T4 (S1-S4): Ejecutar pruebas Docker, actualizar contratos/documentación y registrar evidencia. Route: inline. Commit: `ef14f3e`.

## Log

L1. Usuario: continuar con el resto de las specs después de que el binding de revisión no pudiera admitir los dos reviewers restantes.
L2. La revisión RDD del candidato Phase 5 fue abandonada explícitamente con `operator_disposition`; no se usará inspección visual como sustituto de reviewers nativos. Se continúa con un nuevo slice independiente: facturadores de utilities y tránsito.
L3. Implementados utilities y transit como APIs sintéticas Docker-only, con datos en memoria, tests HTTP públicos, payment intents idempotentes pendientes y frontera explícita sin débito ni escritura de ledger. Utilities: 4 tests; transit: 3 tests; ambos Compose config válidos. Commit: `ef14f3e`.

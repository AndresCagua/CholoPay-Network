# Phase 14 — Ampliación de facturadores y pagos de obligaciones
Fuente: `spec.md`, secciones 24.1, 24.2, 24.3, 24.4 y 26.
Alcance: generalizar el recorrido sintético de obligaciones más allá de electricidad, integrar CholoTránsito y conservar ejecución Docker-only, aislamiento participant-scoped e idempotencia.

## Specs

1. **S1 — Obligaciones de servicios**
   > "Una **sola empresa multiservicios simulada** provee electricidad, agua potable, internet fijo y telefonía móvil, con un único portal web y backend, pero módulos de negocio separados"
   > "Aceptar pagos desde cualquiera de ellos a través de CholoPay Network."

2. **S2 — Autoridad ficticia CholoTránsito**
   > "Pago mediante los tres bancos, Flutter, QR o gateway; constancia ficticia emitida **solo** tras confirmación final."
   > "No presentar constancias como documentos públicos reales ni replicar logotipos oficiales."

3. **S3 — Estados e idempotencia**
   > "No descontar una deuda antes de la confirmación final; no permitir que dos pagos concurrentes liquiden dos veces la misma obligación."
   > "Si el pago finaliza pero el facturador está caído, registrar `SETTLED_AWAITING_APPLICATION` y reintentar la aplicación sin perder ni duplicar fondos."

4. **S4 — APIs mínimas**
   > `GET /v1/services/catalog`
   > `GET /v1/contracts/{contractRef}/bills`
   > `GET /v1/bills/{billId}`
   > `POST /v1/bills/{billId}/payment-intents`
   > `GET /v1/bills/{billId}/payment-status`
   > `GET /v1/vehicles/{syntheticPlate}/obligations`
   > `GET /v1/obligations/{id}`
   > `POST /v1/obligations/{id}/payment-intents`
   > `GET /v1/obligations/{id}/receipt`

5. **S5 — Aceptación prioritaria**
   > "Emitir seis obligaciones sintéticas: luz, agua, internet, móvil, matrícula y multa; pagarlas desde diferentes bancos y desde Flutter; verificar estado en facturador, banco, red y consola central."

## Tasks

- T1 (S1, S3, S4): Extender utilities con catálogo, cuatro servicios, contratos/facturas y consultas contractuales, conservando payment intents y eventos idempotentes. Ruta: worker utilities. Commit: `240dabd`.
- T2 (S2, S3, S4): Extender transit con consulta de obligaciones por placa, intents ligados a obligación, eventos de aplicación y constancia únicamente tras confirmación. Ruta: worker transit. Commit: `240dabd`.
- T3 (S1-S5): Integrar gateway con obligaciones genéricas sin ampliar la frontera de ledger y añadir regresiones públicas de idempotencia, replay y estados pendientes. Ruta: inline después de T1/T2. Commit: `240dabd`.
- T4 (S1-S5): Actualizar OpenAPI, documentación, Compose y evidencia Docker; ejecutar aceptación de las seis obligaciones sintéticas. Ruta: inline. Commit: `099de91`.
- T5 (S5): Conectar los recorridos de pago desde bancos y Flutter con las seis obligaciones, mediante APIs públicas participant-scoped y sin acceso directo a bases. Ruta: siguiente slice. Commit: `5c6890b`.
- T6 (S4-S5): Exponer una proyección gateway participant-scoped de obligaciones sintéticas para que wallet y paneles consulten utilities/transit sin acceder a bases. Ruta: siguiente slice. Commit: pendiente.
  - T5a: Añadir contrato y endpoint participant-scoped de pago de obligaciones en bank-core, delegando al gateway sin escribir ledger local.
  - T5b: Añadir cliente HTTP configurable y flujo sintético de pago de obligaciones en Flutter, sin acceso directo a bases.


## Log

L1. Usuario: "ya terminaste ??? haz el commit y avancemos con lo que falte del spec"
L2. Se cerró la normalización global al español en el commit `c37b543` (`20261110: normaliza proyecto al español`) y se selecciona como siguiente slice la ampliación de obligaciones de utilities y CholoTránsito, porque Phase 13 cubrió únicamente electricidad y la aceptación obligatoria de `spec.md` exige seis obligaciones sintéticas.
L3. T1 completó catálogo, contratos, facturas e intents idempotentes para electricidad, agua, internet y móvil; las rutas históricas `/api/v1` permanecen compatibles. Prueba Docker: 10/10.
L4. T2 completó consulta de obligaciones por placa, intents de tránsito, eventos HMAC idempotentes, estado `SETTLED_AWAITING_APPLICATION` y constancias condicionadas a confirmación final. Prueba Docker: 5/5.
L5. T3 generalizó gateway mediante `biller: utilities|transit`, URLs internas configurables y payloads/eventos específicos por facturador, sin escribir balances ni ledgers. Gateway Docker: 13/13.
L6. Verificación de Compose y `git diff --check` PASS para infra, commerce, utilities y transit. La aceptación consolidada de las seis obligaciones queda pendiente de conectar los recorridos de banco/Flutter y de cerrar T4.
L7. Aceptación sintética del slice: utilities expone electricidad, agua, internet y móvil; transit expone matrícula/multa; gateway enruta utilities/transit con idempotencia, HMAC y aplicación eventual. Suites Docker PASS: utilities 10/10, transit 5/5 y gateway 13/13. La aceptación end-to-end desde banco y Flutter queda explícitamente fuera de este slice porque esas rutas browser-facing aún no existen.
L8. Se registra como siguiente trabajo T5 la conexión de bancos y Flutter con las obligaciones; no se presenta la aceptación sintética de los facturadores como cumplimiento de la aceptación end-to-end completa de `spec.md`.
L9. T5a/T5b implementados: bank-core expone `POST /v1/participants/{participantId}/obligations/{obligationId}/payment-intents` y su alias interno, exige bearer, participante, cuenta de origen e idempotencia, y delega al gateway sin escribir ledger. Wallet usa cliente HTTP configurable, estados de carga/error/vacío, cuenta de origen e idempotencia, sin acceso directo a bases. Bank-core Docker PASS; Flutter Docker PASS con 11 tests y `flutter analyze` sin incidencias. La ruta de consulta participant-scoped de obligaciones queda documentada como adaptador sintético hasta existir una proyección backend única.
L10. Se inicia T6 para eliminar esa limitación: gateway agregará la proyección participant-scoped desde APIs HTTP de utilities y tránsito, sin leer bases ni exponer datos fuera del participante.
L11. T6 completado: `GET /v1/participants/{participantId}/obligations` agrega facturas de utilities y obligaciones de tránsito mediante APIs HTTP internas, con autorización `payment:read`, estados normalizados y respuesta `502` ante indisponibilidad. Gateway Docker PASS: 14/14; `git diff --check` PASS. OpenAPI y README actualizados. La proyección sigue siendo sintética y no lee bases de datos.

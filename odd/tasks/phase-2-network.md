# CholoPay Network — Protocolo y esqueleto de red de la fase 2

Fuente de verdad del primer slice de red central. Lee hasta `## Log` antes de trabajar.

## Specs

1. **S1 — Order:** "fase 0 de arquitectura actualizada → ledger bancario → tres bancos independientes → CholoKey fiel al DICT conceptual → pagos interbancarios".
2. **S2 — Protocol prerequisite:** "¿Qué protocolo exacto garantiza que una reserva del emisor se convierta en débito definitivo tras liquidación central, incluso si el emisor cae?" and "¿Cómo se concilian estados y balances tras recuperación parcial?"
3. **S3 — Finality:** "Solo un asiento confirmado en el ledger central determina `SETTLED`." and uncertain outcomes use `UNKNOWN_PENDING_RECONCILIATION`.
4. **S4 — Reliability:** "transporte al menos una vez; efectos financieros exactamente una vez por identificador mediante idempotencia, constraints y reconciliación".
5. **S5 — Boundaries:** "La red no accede a sus tablas internas" for banks; the network maintains its own settlement ledger and participant state.
6. **S6 — Container rule:** El proyecto debe ejecutarse y probarse dentro de Docker; no se requieren toolchains ni bases de datos del host.
7. **S7 — Ingress:** La sección 28 actualizada de `spec.md` mantiene Traefik como opción futura y Cloudflare como opcional; el esqueleto de red no requiere ingress.

## Tasks

- T1 (S2-S5): Record the reservation, settlement, confirmation, timeout, and reconciliation protocol as an ADR and state contract. Commit: `0a69f32`.
- T2 (S1, S5): Scaffold the Go CholoNet service with health endpoints and PostgreSQL migration layout, without claiming financial settlement. Commit: `0a69f32`.
- T3 (S3-S5): Agregar persistencia idempotente de instrucciones de transferencia y transiciones explícitas; el comportamiento del ledger central de liquidación queda como siguiente slice acotado. Commit: pending.
- T4 (S6): Agregar perfil de compilación/pruebas/Compose exclusivo de Docker. Commit: `0a69f32`.
- T5 (S1-S7): Ejecutar checks y registrar evidencia. Commit: pending.

La evidencia de persistencia de la fase 2 se registra abajo antes del commit posterior.

La evidencia del esqueleto de la fase 2 se registra abajo antes del commit.

## Log

L1. La persona usuaria solicitó agregar la decisión de ingress a `spec.md` y continuar el desarrollo.
L2. La decisión de ingress se agregó como sección 28: Traefik es ingress local FOSS futuro; Cloudflare Tunnel es opcional y nunca una dependencia central.
L3. La fase 2 comienza con documentación del protocolo y un esqueleto de red antes de implementar efectos de liquidación interbancaria.
L4. El ADR 0006 define la reserva del emisor, la finalidad de la liquidación central, la aplicación en el inbox del receptor, los reintentos al menos una vez, el tratamiento de estados desconocidos y la conciliación. Rechaza explícitamente ACID distribuido global y nuevos ID después de resultados inciertos.
L5. El esqueleto Go solo expone endpoints de actividad/disponibilidad y una máquina de estados probada; no accede a tablas bancarias, no resuelve CholoKeys ni afirma liquidación financiera.
L6. `docker compose ... config --quiet` pasó para los perfiles predeterminado y `network-test`. La primera ejecución encontró una declaración faltante de la red `internal_services`; ese defecto de configuración se corrigió antes de repetirla.
L7. `docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile network-test run --rm network-test` passed with exit code 0. Go tests passed inside the Docker build and the one-off test container. No host ports were published.
L8. Las instrucciones de transferencia de CholoNet ahora persisten en PostgreSQL con identidad UUID, hash de solicitud, clave de idempotencia, versión de esquema, ID de correlación/causación, estado y marcas de tiempo. Las transiciones bloquean la fila de instrucción y rechazan transiciones inválidas.
L9. Los handlers HTTP solo internos admiten operaciones de creación, consulta y transición. No exponen liquidación ni escriben tablas bancarias. El servicio de ejecución requiere `NETWORK_DB_URL`, ejecuta el bootstrap de su esquema local y depende del contenedor aislado `network-db`.
L10. Una reconstrucción forzada de Docker expuso inicialmente sumas de comprobación de Go faltantes en el repositorio. Se ejecutó `go mod tidy` dentro de `golang:1.24-alpine`, se generó `go.sum` y luego pasaron la compilación/prueba forzadas de Compose. La salida final incluyó `github.com/cholopay/network/internal/instruction` e `internal/transfer` con código de salida 0.
L11. El slice de liquidación central añade el ADR 0007, cuentas de liquidación, diarios y asientos inmutables, bloqueos canónicos de filas de participantes, comprobaciones de conservación, bloqueo de idempotencia/asesoramiento, replay duplicado, un endpoint interno de liquidación y pruebas de integración concurrentes en Docker. El endpoint informa `SETTLED` solo después de confirmar la transacción central.
L12. La verificación forzada en Docker pasó con las pruebas de `internal/settlement`, incluida la conservación de saldos, el replay duplicado, la protección ante posición insuficiente, la ausencia de sobregiro concurrente y el comportamiento de respuestas HTTP. El verificador independiente no estaba disponible; el usuario eligió explícitamente continuar sin esa verificación externa, por lo que esta limitación queda registrada.
L13. La entrega de crédito al receptor ahora tiene registros duraderos `PENDING`/`APPLIED`, replay de encolado duplicado, acuses vinculados al receptor, rechazo de eventos conflictivos y actualizaciones de `CREDIT_PENDING` → `COMPLETED` solo después de obtener evidencia del receptor. La ejecución forzada en Docker pasó las pruebas de entrega de `internal/settlement`. El verificador volvió a no estar disponible; el usuario autorizó explícitamente cerrar este nuevo slice sin él.
L14. La reconciliación ahora registra ejecuciones y diferencias de solo adición, detecta `SETTLED_WITHOUT_DELIVERY`, `DELIVERY_PENDING` y discrepancias entre aplicado/completado, y expone endpoints internos de ejecución/informe sin mutar el historial financiero. La ejecución forzada de PostgreSQL en Docker pasó los escenarios de entrega ausente, entrega pendiente y ejecución limpia.
L15. El adaptador receptor de bank-core ahora posee una bandeja de entrada duradera y aplica un débito sintético de la cuenta de compensación junto con un crédito a la cuenta del cliente en una transacción local. El replay de una entrega duplicada devuelve el diario original, se rechazan los payloads conflictivos y la compensación insuficiente, y el diario permanece equilibrado. La suite de integración Maven/PostgreSQL forzada en Docker pasó después de etiquetar todas las pruebas de integración de Spring para que la compilación de la imagen siga siendo independiente de la base de datos.
L16. El ADR 0011 añade la frontera del adaptador autenticado: mTLS es el requisito de despliegue y se usa una envoltura HMAC con marca de tiempo para las pruebas locales deterministas en Docker. El endpoint receptor rechaza firmas ausentes, expiradas, malformadas o manipuladas antes de invocar la transacción de la bandeja de entrada. El flujo de integración firmado con MockMvc y todas las pruebas Maven/PostgreSQL en Docker pasaron.
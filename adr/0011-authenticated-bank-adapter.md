# ADR 0011: adaptador autenticado de CholoNet al banco

- Estado: Aceptado para el slice del adaptador local; mTLS sigue siendo el límite de despliegue
- Fecha: 2026-11-09

## Contexto

CholoNet debe entregar comandos de abono al receptor a un banco específico sin permitir que una persona llamadora no autenticada abone cuentas de clientes. El límite similar a producción requiere mTLS, mientras que la primera integración exclusiva de Docker necesita una comprobación determinista de replay y firma a nivel de aplicación.

## Decisión

El endpoint del banco receptor requiere ambos elementos:

1. **Identidad de transporte:** el límite de despliegue es mTLS entre CholoNet y cada instancia bancaria. La emisión y rotación de certificados y los paquetes de confianza quedan como trabajo de despliegue.
2. **Autenticación de mensajes:** cada comando local del adaptador contiene `X-CholoPay-Request-Timestamp`, `X-CholoPay-Request-Id` y `X-CholoPay-Signature`. La firma es HMAC-SHA256 sobre `timestamp + "." + request_id + "." + canonical payload`, usando un secreto por banco suministrado mediante la configuración de ejecución de Docker. El receptor rechaza firmas ausentes, malformadas, incorrectas o expiradas y recuerda los ID de solicitud durante la ventana de replay.

El receptor verifica la envoltura antes de invocar la transacción local del inbox. Una entrega duplicada válida puede repetir su recibo existente, pero un ID de solicitud duplicado con un payload diferente se rechaza. Los secretos nunca aparecen en logs ni archivos del repositorio. La capa HMAC no reemplaza mTLS; protege la integridad de los mensajes y permite pruebas locales deterministas.

## Consecuencias

El adaptador puede probarse sin proveedores externos de identidad. El despliegue de producción debe agregar validación y rotación de certificados y políticas de red. La pila Docker predeterminada usa secretos sintéticos únicamente en la configuración de pruebas y no publica el endpoint al host.

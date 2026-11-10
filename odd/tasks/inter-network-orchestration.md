# Orquestación interredes de CholoPay

Fuente de verdad del primer slice de enrutamiento/mensajería de extremo a extremo. Lee hasta `## Log` antes de trabajar.

## Specs

1. **S1 — Route first:** la recepción de transferencias debe validar la pertenencia de los participantes de origen y destino antes de crear un mensaje interredes.
2. **S2 — Message boundary:** las transferencias entre redes crean una única envoltura duradera `TRANSFER_REQUESTED` que contiene los identificadores de transferencia y los dominios de ruta.
3. **S3 — Same-network behavior:** las transferencias dentro de la misma red permanecen como instrucciones sin un mensaje interredes.
4. **S4 — Reliability:** la recepción repetida con la misma clave de idempotencia repite la instrucción y el mensaje existentes sin efectos duplicados.
5. **S5 — Finality:** aceptar una instrucción o un mensaje nunca significa `SETTLED`; la liquidación central sigue siendo una operación separada.
6. **S6 — Non-claims:** el flujo sigue siendo un comportamiento sintético y local de Docker, y no se conecta con SIP, RPI, BCE, bancos ni billeteras reales.

## Tasks

- T1 (S1-S3): Agregar servicio de orquestación y endpoint HTTP interno de recepción. Ruta: inline. Commit: pending.
- T2 (S4-S5): Agregar replay idempotente y evidencia explícita de respuesta sin efectos secundarios de liquidación. Ruta: inline. Commit: pending.
- T3 (S6): Ejecutar pruebas Docker y registrar la evidencia. Ruta: inline. Commit: pending.

## Log

L1. La persona usuaria autorizó continuar después del slice de envolturas de mensajes interredes.
L2. El enrutamiento, la resolución del directorio y la persistencia de mensajes existen por separado. Este slice los compone en un flujo de recepción, preservando el ledger central como única autoridad de liquidación.
L3. Se agregó `POST /internal/v1/inter-network/transfers`. El servicio resuelve los dominios de participantes antes de crear una instrucción, no crea mensajes interredes para rutas dentro de la misma red y persiste un mensaje `TRANSFER_REQUESTED` para rutas entre redes usando el ID de transferencia como ID estable del mensaje. La recepción repetida repite la instrucción y el mensaje, y las pruebas Docker confirman que ninguna ruta de recepción produce `SETTLED`.

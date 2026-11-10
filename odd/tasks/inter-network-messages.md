# Mensajes interredes de CholoPay

Fuente de verdad del slice de envolturas de mensajes interredes. Lee hasta `## Log` antes de trabajar.

## Specs

1. **S1 — Envelope:** "Los eventos propios mantendrán `schema_version`, `message_type`, `correlation_id`, `causation_id`, `idempotency_key` y timestamps UTC."
2. **S2 — Boundaries:** "CholoNet Inter-Network enruta instrucciones entre dominios y coordina la liquidación central simulada; no se convierte en una cuenta minorista ni escribe en los ledgers internos de los participantes."
3. **S3 — Reliability:** el transporte es al menos una vez y los ID de mensajes duplicados no deben crear efectos financieros duplicados.
4. **S4 — Security:** los mensajes permanecen como tráfico interno de Docker y deben llevar la identidad de las redes emisora y destinataria para su validación.
5. **S5 — Non-claims:** "No conectarse al SIP, RPI, BCE ni redes reales." and no ISO 20022 compatibility claim in this phase.
6. **S6 — Container rule:** "Todo debe ejecutarse dentro de contenedores".

## Tasks

- T1 (S1-S2): Definir y persistir envolturas versionadas de mensajes interredes. Ruta: inline. Commit: pending.
- T2 (S3-S5): Agregar aceptación idempotente y validación de límites de red mediante la API HTTP interna. Ruta: inline. Commit: pending.
- T3 (S6): Ejecutar pruebas de PostgreSQL en Docker y registrar la evidencia. Ruta: inline. Commit: pending.

## Log

L1. La persona usuaria autorizó continuar después del slice del directorio CholoKey.
L2. El enrutamiento y la resolución del directorio existentes son deterministas, pero aún no persisten un límite de mensajes. Este slice agrega una envoltura educativa para la coordinación interredes sin implementar conectividad real con BCE/SIP/RPI ni ISO 20022.
L3. Se agregaron la persistencia V7 y el endpoint interno de mensajes. Las envolturas contienen identidades de red, metadatos de esquema/mensaje, ID de correlación y causación, idempotencia, hash del payload, payload JSON y hora de creación UTC. Las pruebas Docker demuestran la primera aceptación, replay idéntico, rechazo de duplicados en conflicto, rechazo de redes inactivas y comportamiento de replay HTTP.

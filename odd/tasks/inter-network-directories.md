# Directorios interredes de CholoPay

Fuente de verdad del slice de directorios distribuidos. Lee hasta `## Log` antes de trabajar.

## Specs

1. **S1 — Directory isolation:** "Cada una mantiene sus propios participantes, directorio distribuido simulado, credenciales, políticas de acceso, mensajes y proyecciones."
2. **S2 — Key lookup:** "resolución de CholoKeys mediante directorios distribuidos simulados".
3. **S3 — Route safety:** "Una ruta no puede resolverse si el participante está suspendido, no existe o pertenece a un dominio ambiguo."
4. **S4 — Evidence:** "Los eventos propios mantendrán `schema_version`, `message_type`, `correlation_id`, `causation_id`, `idempotency_key` y timestamps UTC."
5. **S5 — Non-claims:** "No conectarse al SIP, RPI, BCE ni redes reales." y "CholoPay no declarará compatibilidad ISO 20022 en esta fase."
6. **S6 — Container rule:** "Todo debe ejecutarse dentro de contenedores".

## Tasks

- T1 (S1-S3): Agregar registros aislados de directorio/clave y API de consulta determinista. Ruta: inline. Commit: pending.
- T2 (S4-S5): Agregar evidencia versionada de resolución del directorio y documentación explícita de no conexión con redes reales. Ruta: inline. Commit: pending.
- T3 (S6): Ejecutar pruebas de PostgreSQL en Docker y registrar la evidencia. Ruta: inline. Commit: pending.

## Log

L1. La persona usuaria autorizó continuar después del slice interredes de dos redes de CholoPay.
L2. El slice anterior estableció los dominios de red bancaria y fintech, la pertenencia de participantes y la selección de rutas dentro y fuera de la red. Este slice agrega resolución simulada de directorios distribuidos sin afirmar integración con SIP/RPI del BCE ni compatibilidad con ISO 20022.
L3. Se agregaron registros de CholoKey por red con propiedad del participante, referencia de cuenta, estado, versión y un límite de unicidad delimitado por red. El resolvedor emite evidencia `DIRECTORY_KEY_RESOLVED` con versión de esquema, tipo de mensaje, ID de correlación/causación, clave de idempotencia y hora de resolución UTC. Las pruebas de red en Docker cubren consulta exitosa, aislamiento de redes, rechazo por suspensión y el resolvedor HTTP público.

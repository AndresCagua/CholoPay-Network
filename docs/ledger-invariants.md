# Invariantes del ledger

1. Cada journal está equilibrado: débitos más créditos suman cero por moneda.
2. Los importes son centavos enteros de USD (`BIGINT`), nunca números de punto flotante.
3. Las cuentas restringidas nunca tienen saldo disponible negativo.
4. Una instrucción de negocio no puede crear dos operaciones de liquidación.
5. Las posiciones de liquidación interbancaria conservan el valor salvo que una operación de fondeo explícita y auditada cree asientos compensatorios.
6. Los asientos publicados son inmutables; la corrección es un movimiento compensatorio.
7. Las claves de idempotencia tienen alcance, están vinculadas a un hash de solicitud y rechazan un payload diferente con HTTP 409.
8. El estado del ledger es autoritativo en PostgreSQL; las cachés y los streams de eventos pueden reconstruirse.
9. Cada operación financiera lleva identificadores de transacción, correlación, causación y auditoría.

Las pruebas deben afirmar resultados observables mediante interfaces públicas e incluir reservas concurrentes, solicitudes duplicadas, incertidumbre por timeout y conciliación.

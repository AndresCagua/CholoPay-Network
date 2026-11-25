# Pruebas de aceptación Docker-only

Todos los runners de esta página se ejecutan dentro de Docker o son wrappers externos que orquestan `docker compose`. No requieren runtimes, clientes PostgreSQL ni sockets Docker dentro de contenedores. Los resultados históricos no sustituyen una ejecución fresca.

## Inventario

| Runner | Perfil/prerrequisitos | Qué verifica | Evidencia/limitación |
|---|---|---|---|
| `p6_utilities_recovery.sh` | `p6`; `utilities-db` y `utilities` | Recovery de las cuatro facturas utilities sintéticas, intents/eventos idempotentes y volumen retenido | Local/sintético; no bancos, settlement ni durabilidad productiva |
| `p6_transit_recovery.sh` | `p6`; `transit-db` y `transit` | Recovery de registros, multas, obligaciones, intents, eventos, receipts y constancias transit | Local/sintético; no gateway, ledger ni certificados oficiales |
| `p7_http_outbox_recovery.sh` | `p7`; red, Costa bank-core y databases | Outbox HTTP pending/leased, reinicio de productor, publicación y deduplicación durable | At-least-once local; no exactly-once universal, RPO/RTO ni ordering global |
| `p8_health_acceptance.sh` | `p8`; servicios P8 | Health/readiness, cuerpos JSON UP/ok/DOWN y recovery de `network-db`/`costa-db` | Contrato local; no HA, SLO/SLI ni disponibilidad productiva |
| `p8_observability_acceptance.sh` | `observability`; Prometheus/Grafana y servicios P8 | Targets, métricas, datasource/dashboard y scrape durante outage/recovery de Costa | Observabilidad local/efímera; no alertas, retención ni monitoreo productivo |
| `p9_performance_acceptance.sh` | perfil P9/CholoLab según el runner | Lecturas financieras sintéticas, concurrencia y latencias registradas | No mide 1M TPS ni capacidad productiva; target P9 quedó sin review por decisión explícita |
| `p10_traceability_check.sh` | Docker; checker estructural | Matriz P1-P9, referencias actuales P6-P8, ADRs, OpenAPI, límites y Compose privado | Valida documentación, no cumplimiento total ni tracing distribuido |
| `phase20_receiver_outage.sh` | profile Phase 20 | Outage receiver, DLQ, replay, recovery, reconciliación e idempotencia | Evidencia histórica/documentada; no ordering global ni garantías productivas |
| `phase20_kafka_acceptance.sh` | profile `events`/Phase 20 | Transporte Kafka versionado, restart, replay at-least-once y deduplicación | Kafka es transporte, no ledger; no backup Kafka ni exactly-once universal |
| `phase20_backup_restore.sh` | Phase 20 y PostgreSQL Docker | Backup/restore lógico de un `network-db` sintético | No PITR/WAL, RPO/RTO, cifrado, retención ni consistencia cross-database |
| `t7_obligations_e2e.py` | profile `acceptance`; `obligations-e2e` | Matrícula/multa, pagos sintéticos, idempotencia y conflictos vía APIs HTTP | `APPLIED`/`PAID` no implican ledger real ni constancia oficial |

## Ejecución del runner legacy

```sh
docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile acceptance up --build --abort-on-container-exit --exit-code-from obligations-e2e obligations-e2e
```

La prueba crea matrícula y multa sintéticas por HTTP, exige las seis clases en la proyección participant-scoped, repite cada pago con la misma clave y comprueba conflicto con una carga distinta.

## Evidencia

Cada runner documenta su alcance en el runbook o task ODD correspondiente. Una línea `PASS` de un runner es evidencia del escenario ejecutado, no una afirmación de producción, dinero real, SLO/SLI, RPO/RTO, HA, exactly-once u ordering global.

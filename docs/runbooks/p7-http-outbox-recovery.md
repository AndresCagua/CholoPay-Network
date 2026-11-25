# P7 — Recuperación del outbox HTTP sintético

**Spec:** `odd/tasks/phase-7-http-outbox-recovery.md`, T1-T2, S1-S5.

## Propósito

La aceptación `infra/tests/p7_http_outbox_recovery.sh` demuestra, únicamente en Compose y sin puertos host, que el `bank-core` de Costa usa `OUTBOX_ENABLED=true` y `OUTBOX_TRANSPORT=http`. Crea una transferencia inter-red sintética, interrumpe `network` antes de publicar, observa el outbox `PENDING` o `IN_FLIGHT`, detiene y arranca externamente `bank-core`, restaura `network` y espera la publicación con polling acotado.

La interrupción, parada y recuperación son responsabilidad del shell en el host. `phase7-http-outbox` solo usa aliases Compose, HTTP y PostgreSQL de lectura; nunca recibe el socket Docker.

## Ejecución

Desde la raíz del repositorio:

```sh
infra/tests/p7_http_outbox_recovery.sh
```

La envoltura inyecta explícitamente `OUTBOX_ENABLED=true` y `OUTBOX_TRANSPORT=http` al proyecto Compose. El binding JDBC de `occurred_at` convierte el `Instant` del evento a `Timestamp` únicamente en el parámetro de persistencia; no cambia la semántica financiera ni el envelope. El probe usa límites de 90 segundos para readiness/publicación, intervalos de un segundo y timeouts HTTP de cinco segundos. Si el límite expira, falla en lugar de esperar indefinidamente.

También se puede validar la configuración sin ejecutar contenedores:

```sh
CHOLONET_HMAC_SECRET=p7-local-hmac docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile p7 config
```

## Evidencia

El probe comprueba:

- una fila `inter_network_outbox` sintética con `event_id`, `aggregate_id`, `payload_hash` y envelope JSON válido;
- que el registro sobrevive a la interrupción y al reinicio, y termina en `PUBLISHED`;
- que `network` vuelve a `/health/ready` con HTTP 200 y que el listener HTTP de `bank-core` vuelve a responder (la imagen actual no expone un endpoint health propio, por lo que se usa una petición de método no permitido como prueba de proceso vivo);
- un único `inter_network_message` durable para la clave de idempotencia `transfer:<bank-outbox-key>`, con un único `message_id`, y que su `message_id` coincide con el aggregate de la transferencia.

La frontera de red puede recibir reintentos; la unicidad observada es la deduplicación durable de esta ejecución sintética, no una garantía universal de exactly-once.

## Límites

Esta evidencia es local, sintética y Docker-only. No demuestra RPO/RTO, disponibilidad productiva, orden global, consistencia cross-database, Kafka, seguridad productiva, otros bancos ni semántica financiera adicional. No se modificó el ledger para construir la aceptación.

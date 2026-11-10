# ADR 0005: Outbox/inbox y herramientas locales FOSS

- Estado: Aceptado para la fase 0
- Fecha: 2026-11-09

## Decisión

Los servicios publican eventos mediante outboxes transaccionales y consumen mediante inboxes idempotentes duraderos. Kafka es opcional en el primer perfil local. PostgreSQL es la verdad persistente; Redis 8+ con una distribución AGPL se limita a caché/estado efímero. El desarrollo local usa Docker Engine o Podman, Compose e imágenes de código abierto; no se requiere Desktop, nube ni dependencia de SaaS de pago.

## Consecuencias

El replay de eventos y la conciliación son capacidades de primera clase. Las etiquetas exactas de las imágenes y sus licencias deben fijarse y revisarse antes de los perfiles similares a producción. Las obligaciones de copyleft se registran en el inventario de dependencias.

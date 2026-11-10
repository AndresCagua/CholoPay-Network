# ADR 0001: Límites de servicios y runtimes

- Estado: Aceptado para la fase 0
- Fecha: 2026-11-09

## Contexto

El proyecto demuestra una red en Go e instancias bancarias independientes y parametrizadas en Java/Spring Boot. Las interfaces de usuario quedan explícitamente para una etapa posterior.

## Decisión

Construir la red como un monolito modular en Go y el núcleo bancario como un producto Java/Spring Boot desplegado tres veces con identidad, configuración, base de datos PostgreSQL, credenciales y almacenamiento separados. Mantener el gateway como opcional y posterior. Usar Compose antes que Kubernetes.

## Consecuencias

Esto preserva el aislamiento institucional sin duplicar la lógica bancaria, reduce la superficie inicial de sistemas distribuidos y deja las decisiones de extracción guiadas por evidencia. Requiere límites de API explícitos e impide que la red escriba en tablas bancarias.

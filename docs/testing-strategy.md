# Estrategia de pruebas

## Checks de la fase 0

- Los archivos Markdown y Mermaid existen y están referenciados por README.
- Los documentos OpenAPI se pueden analizar como YAML y contienen convenciones de versión, seguridad, errores, idempotencia y salud.
- La configuración Compose se genera correctamente sin iniciar servicios.
- La pila Compose predeterminada no publica puertos del host, evitando colisiones.
- La prueba smoke verifica la disponibilidad de PostgreSQL y el aislamiento de las bases desde Docker.

## Checks financieros posteriores

Las pruebas unitarias y de integración deben usar datos sintéticos y APIs públicas. Prueba cada transición de estado nombrada, reservas concurrentes sobre una cuenta, claves de idempotencia duplicadas, reintentos después de una liquidación incierta, reinicio del outbox, caída del receptor y conciliación. Una prueba falla si se viola una invariante, incluso con throughput alto.

## Evidencia de Docker exclusivo

Las compilaciones, pruebas, migraciones, bases de datos y procesos de servicio se ejecutan dentro de contenedores Compose. El prerrequisito del host es Docker Engine y el complemento Compose. Registra el comando Docker, código de salida, etiquetas de imagen, logs de contenedores, parámetros, semilla y limitaciones. No inventes SLO de rendimiento antes de medir una línea base.

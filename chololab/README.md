# CholoLab

CholoLab es un ejecutor sintético de carga e inyección de fallos exclusivo para Docker. No accede directamente a diarios ni bases de datos y no publica puertos de forma predeterminada.

## Comprobaciones reproducibles

```bash
docker compose -f compose.yaml --profile test build chololab-test
docker compose -f compose.yaml config
docker compose -f compose.yaml --profile test run --rm chololab-test
```

## Escenario sintético de carga

El archivo Compose incluye un servicio HTTP `synthetic-target`, solo interno, para obtener evidencia local determinista:

```bash
docker compose -f compose.yaml up -d synthetic-target
docker compose -f compose.yaml run --rm \
  -e TARGET_URL=http://synthetic-target:8080 \
  -e REQUESTS=20 -e CONCURRENCY=4 -e FAILURE_EVERY=5 chololab
docker compose -f compose.yaml down
```

El ejecutor emite un registro JSON `run_started` y otro `run_summary`. El resumen conserva el número de solicitudes, los fallos inyectados, las respuestas exitosas, el tiempo transcurrido, el rendimiento y la latencia mínima/mediana/máxima, además de los campos de estilo RED:

- `red.rate_requests_per_second`: tasa de solicitudes completadas para esta ejecución.
- `red.errors` and `red.error_rate`: fallos no-2xx, de transporte e inyectados.
- `red.duration_ms`: duración total transcurrida de la ejecución.

Estas son mediciones sintéticas deterministas para los parámetros seleccionados, no afirmaciones de capacidad de producción. Compara registros JSON completos de ejecuciones con el mismo objetivo, número de solicitudes, concurrencia y `FAILURE_EVERY`; no los extrapoles a hardware o tráfico de producción.

## Procedimiento operativo

1. Inicia únicamente el objetivo sintético interno: `docker compose -f compose.yaml up -d synthetic-target`.
2. Ejecuta el escenario acotado mostrado arriba. Mantén `TARGET_URL` en la red Compose; no añadas un mapeo `ports` mapping.
3. Inspecciona las dos líneas JSON. Confirma que el registro inicial coincide con los parámetros solicitados y usa después `run_summary.red` para la tasa, los errores y la duración. `injected_failures` debe coincidir con la programación determinista `FAILURE_EVERY` schedule cuando el objetivo está sano; otros fallos son evidencia del comportamiento del transporte o del objetivo.
4. Detén el objetivo sintético: `docker compose -f compose.yaml down`.

Este procedimiento usa únicamente tráfico sintético, no accede a diarios y no establece preparación ni capacidad para producción.

# P6 — Recuperación de utilities sintético

Alcance: únicamente la factura sintética de electricidad `invoice-elec-2025-001`. No cubre agua, internet, móvil, tránsito, gateway, ledgers bancarios, settlement global ni dinero real.

## Compose

`utilities-db` es un PostgreSQL privado, con volumen `utilities-db-data`, sin puertos host. `utilities` espera `service_healthy` y usa `DB_URL` exclusivamente hacia `utilities-db`; no usa `costa-db` ni ningún ledger bancario.

## Aceptación Docker-only

Desde la raíz:

```sh
docker compose -f infra/compose.yaml --profile p6 up --build --abort-on-container-exit p6-utilities-recovery
```

La prueba lee la factura, crea y repite una intención fija, firma y repite un evento fijo, y verifica `paid`, `APPLIED` y la identidad del intent. Para demostrar recuperación del contenedor, ejecutar el probe antes y después de recrear solo utilities, sin borrar el volumen:

```sh
docker compose -f infra/compose.yaml up -d --build utilities-db utilities
docker compose -f infra/compose.yaml run --rm p6-utilities-recovery
docker compose -f infra/compose.yaml stop utilities
docker compose -f infra/compose.yaml rm -f utilities
docker compose -f infra/compose.yaml up -d utilities
docker compose -f infra/compose.yaml run --rm p6-utilities-recovery
```

No usar `down -v`: el volumen `utilities-db-data` es la autoridad durable. La prueba es sintética y no demuestra garantías productivas.

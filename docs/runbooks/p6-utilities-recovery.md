# P6 — Recuperación de utilities sintético

Alcance: las facturas sintéticas `invoice-elec-2025-001`, `invoice-water-2025-001` y `bill-mobile-2025-001`. Internet permanece legacy/en memoria. No cubre tránsito, gateway, ledgers bancarios, settlement global ni dinero real.

## Compose

`utilities-db` es un PostgreSQL privado, con volumen `utilities-db-data`, sin puertos host. `utilities` espera `service_healthy` y usa `DB_URL` exclusivamente hacia `utilities-db`; no usa `costa-db` ni ningún ledger bancario.

## Aceptación Docker-only

Desde la raíz:

```sh
sh infra/tests/p6_utilities_recovery.sh
```

El wrapper ejecuta el probe antes y después de recrear solo `utilities`, conservando el volumen de `utilities-db`. El probe verifica el catálogo de cuatro servicios, crea y repite intents fijos (mobile mediante la ruta nueva y water mediante la legacy), firma y repite eventos fijos, y comprueba para mobile y water `paid`, `APPLIED`, identidad del intent y exactamente un intent/evento durable. También verifica que electricidad sigue presente y que internet continúa visible como factura legacy.

No usar `down -v` durante la comprobación de recuperación: el volumen `utilities-db-data` es la autoridad durable. El cleanup final del wrapper elimina recursos de la ejecución de prueba. La prueba es sintética y no demuestra garantías productivas.

# Runbook P6 — recuperación de tránsito sintético

Este slice persiste el aggregate de `transit` en el PostgreSQL aislado `transit-db`, con volumen propio y sin puertos publicados. Conserva las rutas, payloads, estados, HMAC, idempotency keys, conflictos de fingerprint y replay existentes.

## Validación Docker-only

```sh
infra/tests/p6_transit_recovery.sh
```

La aceptación crea una matrícula y multa sintéticas, reserva intents, verifica replay/conflicto, aplica eventos firmados, obtiene receipts/constancia, recrea únicamente `transit` y compara los mismos IDs y estados.

## Operación local

```sh
docker compose -f infra/compose.yaml up -d transit-db transit
docker compose -f infra/compose.yaml --profile p6 run --rm p6-transit-recovery
```

`transit` depende de `transit-db` saludable y usa `DB_URL` interno. No conecta con `utilities-db`, bancos, gateway, ledgers ni settlement global.

## Límites

Los registros, pagos, eventos, receipts y constancias son sintéticos. Esto no demuestra durabilidad de producto, dinero real, gateway/bank settlement, autoridad de tránsito ni certificados oficiales.

# Runbook del esqueleto de red de la fase 2

El esqueleto de red es únicamente interno y no tiene puertos del host por defecto. Compílalo y pruébalo dentro de Docker:

```bash
docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile network-test run --rm network-test
```

El check de salud del esqueleto y las pruebas de transición de estados no afirman que ninguna transferencia esté liquidada financieramente. Una afirmación de liquidación requiere la futura transacción del ledger central y su evidencia.

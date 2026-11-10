# Esqueleto de red CholoNet — Fase 2

Este servicio Go es el primer corte de la red central. Actualmente solo expone endpoints de actividad/disponibilidad y una máquina de estados de transferencias probada. **No** escribe asientos de liquidación, resuelve CholoKeys, accede a tablas bancarias ni afirma que una transferencia esté liquidada.

Construye y prueba únicamente mediante Docker desde la raíz del repositorio:

```bash
docker compose --env-file infra/config.env.template -f infra/compose.yaml --profile network-test run --rm network-test
```

La imagen de ejecución es distroless y se comunica únicamente en la red Compose. De forma predeterminada no se publica ningún puerto al host.

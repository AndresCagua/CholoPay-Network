.DEFAULT_GOAL := help
COMPOSE := docker compose --env-file infra/config.env.template -f infra/compose.yaml

.PHONY: help config up test smoke down
help:
	@printf '%s\n' 'Comandos exclusivos para Docker (GNU Make es opcional):' '  make config - renderiza la configuración Compose' '  make up     - inicia las bases de datos privadas y el núcleo bancario' '  make test   - ejecuta las pruebas Maven en el contenedor bank-core' '  make smoke  - ejecuta comprobaciones de salud y aislamiento de bases de datos' '  make down   - detiene el proyecto Compose'

config:
	@$(COMPOSE) config --quiet
	@echo 'La configuración Compose es válida'

up: config
	@$(COMPOSE) up -d network-db costa-db sierra-db oriente-db bank-core

test:
	@$(COMPOSE) --profile test run --rm bank-core-test

smoke:
	@$(COMPOSE) ps
	@$(COMPOSE) exec -T network-db pg_isready -U "$${POSTGRES_USER:-cholopay}" -d "$${NETWORK_DB_NAME:-cholopay_network}"
	@$(COMPOSE) exec -T costa-db pg_isready -U "$${POSTGRES_USER:-cholopay}" -d "$${COSTA_DB_NAME:-cholopay_costa}"
	@$(COMPOSE) exec -T sierra-db pg_isready -U "$${POSTGRES_USER:-cholopay}" -d "$${SIERRA_DB_NAME:-cholopay_sierra}"
	@$(COMPOSE) exec -T oriente-db pg_isready -U "$${POSTGRES_USER:-cholopay}" -d "$${ORIENTE_DB_NAME:-cholopay_oriente}"

down:
	@$(COMPOSE) down

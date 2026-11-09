.DEFAULT_GOAL := help
COMPOSE := docker compose --env-file infra/config.env.template -f infra/compose.yaml

.PHONY: help config up test smoke down
help:
	@printf '%s\n' 'Docker-only commands (GNU Make is optional):' '  make config - render Compose configuration' '  make up     - start private databases and bank core' '  make test   - run Maven tests in the bank-core container' '  make smoke  - run container health and database isolation checks' '  make down   - stop the Compose project'

config:
	@$(COMPOSE) config --quiet
	@echo 'Compose configuration is valid'

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

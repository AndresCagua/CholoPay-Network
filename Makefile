.DEFAULT_GOAL := help
COMPOSE := docker compose --env-file infra/config.env.template -f infra/compose.yaml

.PHONY: help config ports up down smoke
help:
	@printf '%s\n' 'make ports  - verify host ports before startup' 'make config - render Compose configuration' 'make up     - check ports, then start default databases' 'make smoke  - run health and isolation smoke test' 'make down   - stop default services'

ports:
	@./infra/scripts/check-ports.sh

config:
	@$(COMPOSE) config --quiet
	@echo 'Compose configuration is valid'

up: ports config
	@$(COMPOSE) up -d

smoke:
	@./infra/scripts/smoke.sh

down:
	@$(COMPOSE) down

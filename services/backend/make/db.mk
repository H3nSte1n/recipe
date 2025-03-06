.PHONY: migrate-% db-%

migrate-create: ## Create a new migration file
	@if [ -z "$(name)" ]; then \
		echo "Please provide a migration name. Use: make migrate-create name=your_migration_name"; \
		exit 1; \
	fi
	migrate create -ext sql -dir migrations -seq $(name)

migrate-up: ## Apply all migrations
	$(DOCKER_EXEC) app migrate -path migrations -database "$(DB_URL)" up

migrate-down: ## Undo all migrations
	$(DOCKER_EXEC) app migrate -path migrations -database "$(DB_URL)" down

migrate-status: ## Show migration status
	$(DOCKER_EXEC) app migrate -path migrations -database "$(DB_URL)" version

migrate-force: ## Force set migration version
	@if [ -z "$(version)" ]; then \
		echo "Please provide version number. Use: make migrate-force version=x"; \
		exit 1; \
	fi
	$(DOCKER_EXEC) app migrate -path migrations -database "$(DB_URL)" force $(version)

db-connect: ## Connect to database
	$(DOCKER_EXEC) db psql -U $(DB_USER) -d $(DB_NAME)

db-create: ## Create database
	$(DOCKER_EXEC) db psql -U $(DB_USER) -c "CREATE DATABASE $(DB_NAME)"

db-drop: ## Drop database
	$(DOCKER_EXEC) db psql -U $(DB_USER) -c "DROP DATABASE IF EXISTS $(DB_NAME)"

# Additional Docker commands
docker-up: ## Start docker containers
	$(DOCKER_COMPOSE) up -d

docker-down: ## Stop docker containers
	$(DOCKER_COMPOSE) down

docker-logs: ## View docker logs
	$(DOCKER_COMPOSE) logs -f

docker-ps: ## List running containers
	$(DOCKER_COMPOSE) ps

# Combined commands
setup-dev: docker-up db-create migrate-up ## Setup development environment

teardown-dev: migrate-down docker-down ## Teardown development environment

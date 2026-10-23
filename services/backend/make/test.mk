.PHONY: test-% coverage

test: ## Run tests
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

test-integration: ## Run Postgres integration tests (starts a throwaway database via docker-compose.test.yml)
	docker compose -f ../../docker-compose.test.yml up -d --wait
	TEST_DATABASE_URL="postgres://postgres:mealplan_test@127.0.0.1:55433/recipe_mealplan_test?sslmode=disable" \
		$(GOTEST) -tags=integration -v ./internal/repository/...; status=$$?; \
		docker compose -f ../../docker-compose.test.yml down; exit $$status

coverage: test-coverage ## Generate coverage report
	$(GOCMD) tool cover -func=coverage.out
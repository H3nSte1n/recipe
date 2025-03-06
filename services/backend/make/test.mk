.PHONY: test-% coverage

test: ## Run tests
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

test-integration: ## Run integration tests
	$(GOTEST) -tags=integration -v ./...

coverage: test-coverage ## Generate coverage report
	$(GOCMD) tool cover -func=coverage.out
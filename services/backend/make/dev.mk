.PHONY: dev-% lint fmt

dev-tools: ## Install development tools
	go install github.com/golang/mock/mockgen@latest
	go install github.com/golangci/golint/golint@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest

lint: ## Run linters
	golangci-lint run

fmt: ## Format code
	$(GOFMT) ./...

dev-setup: deps dev-tools ## Setup development environment
	@echo "Development environment setup complete"
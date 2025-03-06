.PHONY: build run clean deps

build: ## Build the application
	$(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)

run: ## Run the application
	$(GORUN) $(MAIN_PACKAGE)

clean: ## Clean the build directory
	rm -rf bin/
	rm -rf vendor/

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) verify

deps-upgrade: ## Upgrade dependencies
	$(GOMOD) tidy
	$(GOGET) -u all
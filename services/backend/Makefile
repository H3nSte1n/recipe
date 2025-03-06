# Include variables
include make/variables.mk

# Include other makefiles
include make/app.mk
include make/db.mk
include make/dev.mk
include make/test.mk

.DEFAULT_GOAL := help

.PHONY: help
help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
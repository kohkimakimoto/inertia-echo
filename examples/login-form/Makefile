.DEFAULT_GOAL := help

SHELL := bash
BUILD_LDFLAGS = "-s -w -X main.BuildMode=production"

.PHONY: help
help: ## Show help
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[/0-9a-zA-Z_-]+:.*?## .*$$' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'


.PHONY: dev
dev: ## Run the application in development mode
	@if [[ ! -d "node_modules" ]]; then \
		npm install; \
	fi
	@go run .


.PHONY: run
run: build ## Run the application in production mode
	@./app


.PHONY: build
build: ## Build the application
	@if [[ ! -d "node_modules" ]]; then \
		npm install; \
	fi
	@npm run build
	@go build -ldflags=$(BUILD_LDFLAGS) -o app


.PHONY: clean
clean: ## Clean
	@npm run clean

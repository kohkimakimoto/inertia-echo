.DEFAULT_GOAL := help

SHELL := bash
BUILD_LDFLAGS = "-s -w -X main.BuildMode=production"

# Load .env file if it exists.
ifneq (,$(wildcard ./.env))
  include .env
  export
endif

.PHONY: help
help: ## Show help
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[/0-9a-zA-Z_-]+:.*?## .*$$' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'


.PHONY: run
run: ## Run the application in development mode
	@../../_tools/process-starter.py \
		'{"command": "go run main.go -dir=$(DIR)", "prefix": "[go]", "prefixColor": "blue"}' \
		'{"command": "npm run start-ssr", "prefix": "[ssr]", "prefixColor": "magenta"}' \
		'{"command": "npm run dev", "prefix": "[vite]", "prefixColor": "green"}'


.PHONY: run/production
run/production: build ## Run the application in production mode
	@../../_tools/process-starter.py \
		'{"command": ".generated/ssrnodejs -dir=$(DIR)", "prefix": "[go]", "prefixColor": "blue"}' \
		'{"command": "npm run start-ssr", "prefix": "[ssr]", "prefixColor": "magenta"}'


.PHONY: build
build: ## Build the application
	@mkdir -p .generated
	@npm run build
	@go build -ldflags=$(BUILD_LDFLAGS) -o .generated/ssrnodejs


.PHONY: clean
clean: ## Clean
	@rm -rf .generated
	@npm run clean

.DEFAULT_GOAL := help

SHELL := bash
BUILD_LDFLAGS = "-s -w -X main.BuildMode=production"

.PHONY: help
help: ## Show help
	@echo "Usage: make [target]"
	@echo ""
	@echo "PoC: Inertia SSR inside the Go process via liustack/slowjs (no Node SSR server)."
	@echo ""
	@echo "Available targets:"
	@grep -E '^[/0-9a-zA-Z_-]+:.*?## .*$$' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'


.PHONY: dev
dev: ssr-bundle ## Run in debug mode (Vite for client assets; slowjs for SSR)
	@if [[ ! -d "node_modules" ]]; then \
		npm install; \
	fi
	@go run .


.PHONY: run
run: build ## Run the production binary
	@./app


.PHONY: ssr-bundle
ssr-bundle: ## Build the thin SSR IIFE for slowjs
	@if [[ ! -d "node_modules" ]]; then \
		npm install; \
	fi
	@npm run build:ssr


.PHONY: build
build: ## Build client assets, SSR bundle, and the Go binary
	@if [[ ! -d "node_modules" ]]; then \
		npm install; \
	fi
	@npm run build
	@go build -ldflags=$(BUILD_LDFLAGS) -o app


.PHONY: clean
clean: ## Clean build artifacts
	@npm run clean

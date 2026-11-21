.DEFAULT_GOAL := help

SHELL := bash

.PHONY: help
help: ## Show help
	@echo "Usage: make [target]"
	@echo ""
	@echo "Compare examples/ssr-slowjs (in-process) vs examples/ssr-node (Node HTTP SSR)."
	@echo ""
	@grep -E '^[/0-9a-zA-Z_-]+:.*?## .*$$' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: bundle
bundle: ## Build the slowjs SSR IIFE
	@$(MAKE) -C ../ssr-slowjs ssr-bundle

.PHONY: bench
bench: bundle ## Run comparison (default -n 200)
	@go run . -n 200 -warmup 20

.PHONY: bench-long
bench-long: bundle ## Longer run (-n 1000)
	@go run . -n 1000 -warmup 50

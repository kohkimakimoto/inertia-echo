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
bench: bundle ## Latency (sequential, -n 200, slowjs -pool 1)
	@go run . -mode latency -n 200 -warmup 20 -pool 1

.PHONY: bench-long
bench-long: bundle ## Latency longer (-n 1000, slowjs -pool 1)
	@go run . -mode latency -n 1000 -warmup 50 -pool 1

.PHONY: bench-throughput
bench-throughput: bundle ## Throughput (concurrent workers=GOMAXPROCS, slowjs pool=GOMAXPROCS)
	@go run . -mode throughput -n 5000 -warmup 50 -pool 0 -workers 0

.PHONY: bench-all
bench-all: bundle ## Latency + throughput (-n 1000 latency / same n throughput)
	@go run . -mode both -n 1000 -warmup 50

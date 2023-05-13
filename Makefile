.DEFAULT_GOAL := help

SHELL := bash
PATH := $(CURDIR)/.dev/gopath/bin:$(PATH)

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


.PHONY: dev/setup
dev/setup: ## Setup development environment
	@mkdir -p .dev/gopath
	@export GOPATH=$(CURDIR)/.dev/gopath && \
		go install honnef.co/go/tools/cmd/staticcheck@latest && \
		go install github.com/axw/gocov/gocov@latest && \
		go install github.com/matm/gocov-html/cmd/gocov-html@latest && \
		go install github.com/cosmtrek/air@latest


.PHONY: dev/clean
dev/clean: ## Clean up development environment
	@export GOPATH=$(CURDIR)/.dev/gopath && go clean -modcache
	@rm -rf .dev


.PHONY: format
format: ## Format source code
	@go fmt ./...


.PHONY: lint
lint: ## Static code analysis
	@go vet ./...
	@staticcheck ./...


.PHONY: test
test: ## Test go code
	@go test -race -timeout 30m -cover $$(go list ./... | grep -v /examples)


.PHONY: test/verbose
test/verbose: ## Run all tests with verbose outputting.
	@go test -race -timeout 30m -v -cover  $$(go list ./... | grep -v /examples)


.PHONY: test/coverage
test/coverage: ## Run tests with coverage report
	@mkdir -p .dev
	@go test -race -timeout 30m -cover $$(go list ./... | grep -v /examples) -coverprofile=.dev/coverage.out
	@gocov convert .dev/coverage.out | gocov-html > .dev/coverage.html


.PHONY: open-coverage-html
open-coverage-html: ## Open coverage report
	@open .dev/coverage.html


.PHONY: clean
clean: ## Clean generated files
	@rm -rf .dev/coverage.html
	@rm -rf .dev/coverage.out


.PHONY: run/helloworld
run/helloworld: ## Run example helloworld app
	@mkdir -p examples/helloworld/.generated
	@cd examples/helloworld && ./scripts/process-starter.py \
		'{"command": "air", "prefix": "[air] ", "prefixColor": "blue"}'


#.PHONY: clean
#clean: ## Clean the generated contents
#	@rm -rf coverage-report.html
#	@rm -rf examples/helloworld/gen/dist
#	@rm -rf examples/helloworld/main
#	@rm -rf examples/helloworld/.tmp

#.PHONY: examples/helloworld/start
#examples/helloworld/start: examples/helloworld/yarn-install ## start example helloworld app
#	@if [[ ! -e examples/helloworld/gen/dist ]]; then cd examples/helloworld && yarn build; fi
#	@cd examples/helloworld && ./process-starter.py --run "yarn dev" "air"
#
#.PHONY: examples/helloworld/build
#examples/helloworld/build: examples/helloworld/yarn-install ## build example helloworld app
#	@cd examples/helloworld && yarn build && go build -o main main.go
#
#.PHONY: examples/helloworld/yarn-install
#examples/helloworld/yarn-install:
#	@if [[ ! -e examples/helloworld/node_modules ]]; then cd examples/helloworld && yarn install; fi

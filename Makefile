.DEFAULT_GOAL := help

# Environment
GO111MODULE := on
PATH := $(CURDIR)/.external-tools/bin:$(PATH)
SHELL := bash

# Output help message
# see https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help:
	@grep -E '^[/0-9a-zA-Z_-]+:.*?## .*$$' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-28s\033[0m %s\n", $$1, $$2}'

.PHONY: format
format: ## Format go code
	@find . -print | grep --regex '.*\.go$$' | xargs goimports -w -l -local "github.com/kohkimakimoto/inertia-echo"

.PHONY: deps
deps: ## Install go modules
	@go mod tidy

.PHONY: tools/install
tools/install: ## Install dev tools
	@export GOBIN=$(CURDIR)/.external-tools/bin && \
		go install golang.org/x/tools/cmd/goimports@latest && \
		go install github.com/axw/gocov/gocov@latest && \
		go install github.com/matm/gocov-html@latest && \
		go install github.com/cosmtrek/air@latest

.PHONY: tools/clean
tools/clean: ## Clean installed tools
	@rm -rf $(CURDIR)/dev/.external-tools

.PHONY: test
test: ## Test go code
	@go test -race -timeout 30m -cover $$(go list ./... | grep -v /example)

.PHONY: test/verbose
test/verbose: ## Run all tests with verbose outputting.
	@go test -race -timeout 30m -v -cover  $$(go list ./... | grep -v /example)

.PHONY: test/coverage
test/coverage: ## Run all tests with coverage report outputting.
	@gocov test $$(go list ./... | grep -v /example) | gocov-html > coverage-report.html

.PHONY: clean
clean: ## Clean the generated contents
	@rm -rf coverage-report.html
	@rm -rf examples/helloworld/gen/dist
	@rm -rf examples/helloworld/main
	@rm -rf examples/helloworld/.tmp

.PHONY: examples/helloworld/start
examples/helloworld/start: examples/helloworld/yarn-install ## start example helloworld app
	@if [[ ! -e examples/helloworld/gen/dist ]]; then cd examples/helloworld && yarn build; fi
	@cd examples/helloworld && ./process-starter.py --run "yarn dev" "air"

.PHONY: examples/helloworld/build
examples/helloworld/build: examples/helloworld/yarn-install ## build example helloworld app
	@cd examples/helloworld && yarn build && go build -o main main.go

.PHONY: examples/helloworld/yarn-install
examples/helloworld/yarn-install:
	@if [[ ! -e examples/helloworld/node_modules ]]; then cd examples/helloworld && yarn install; fi

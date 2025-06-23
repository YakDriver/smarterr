SHELL := /bin/bash

default: build

build: ## Build the project and install the binary to $GOBIN
	@echo "make: Building..."
	@go install

coverage-internal: ## Run coverage for internal package
	@go test -coverprofile=internal.coverage.out ./internal/...
	@go tool cover -func=internal.coverage.out

coverage-internal-check: ## Check that internal package coverage meets minimum threshold
	@$(MAKE) coverage-internal
	@total=$$(go tool cover -func=internal.coverage.out | grep total: | awk '{print substr($$3, 1, length($$3)-1)}'); \
	min=65.0; \
	awk "BEGIN {exit ($$total < $$min) ? 1 : 0}"

coverage-root-cli: ## Run coverage for root and CLI packages
	@go test -coverprofile=other.coverage.out . ./cmd/smarterr/...
	@go tool cover -func=other.coverage.out

fmt: ## Format Go source code in all packages
	@echo "make: Formatting source code with gofmt..."
	@find . -name '*.go' -exec gofmt -s -w {} +

fmt-check: ## Check Go source formatting (fails if not formatted)
	@echo "make: Checking source code formatting with gofmt..."
	@fmt_out=$$(find . -name '*.go' -exec gofmt -s -l {} +); \
	if [ -n "$$fmt_out" ]; then \
		echo "$$fmt_out"; \
		echo 'Code is not gofmt formatted. Run `make fmt`.'; \
		exit 1; \
	fi

help: ## Show this help message
	@echo "Available targets:"; \
	awk 'BEGIN {FS = ":.*?## "}; /^[a-zA-Z0-9][^:]*:.*?## / {printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: build ## Build and install the binary

lint: ## Run golangci-lint
	@golangci-lint run

modern: ## Fix code to use modern Go idioms
	@echo "make: Fixing checks for modern Go code..."
	@go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix -test ./...

modern-check: ## Check for modern Go idioms
	@echo "make: Checking for modern Go code..."
	@go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -test ./...

staticcheck: ## Run staticcheck linter
	@staticcheck ./...

test: ## Run unit tests
	@echo "make: Running unit tests..."
	go test ./... -timeout=15m -vet=off

tidy: ## Run go mod tidy to fix go.mod and go.sum
	@echo "make: Tidying go.mod and go.sum..."
	@go mod tidy

tidy-check: ## Check if go.mod and go.sum are tidy (no changes)
	@echo "make: Checking if go.mod and go.sum are tidy..."
	@cp go.mod go.mod.bak
	@cp go.sum go.sum.bak
	@go mod tidy
	@git diff --exit-code go.mod go.sum
	@mv go.mod.bak go.mod
	@mv go.sum.bak go.sum

vet: ## Run go vet on all packages
	@go vet ./...

vulncheck: ## Run govulncheck for known vulnerabilities
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Please keep targets in alphabetical order
.PHONY: \
	build \
	coverage-internal \
	coverage-internal-check \
	coverage-root-cli \
	fmt \
	fmt-check \
	help \
	install \
	lint \
	modern \
	modern-check \
	staticcheck \
	test \
	tidy \
	tidy-check \
	vet \
	vulncheck

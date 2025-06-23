SHELL := /bin/bash

default: build

build:
	@echo "make: Building..."
	@go install

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

install: build

modern-check:
	@echo "make: Checking for modern Go code..."
	@go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -test ./...

modern-fix:
	@echo "make: Fixing checks for modern Go code..."
	@go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix -test ./...

test: ## Run unit tests
	@echo "make: Running unit tests..."
	go test ./... -timeout=15m -vet=off

vet:
	@go vet ./...

staticcheck:
	@staticcheck ./...

vulncheck:
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./...

lint:
	@golangci-lint run

coverage-internal:
	@go test -coverprofile=internal.coverage.out ./internal/...
	@go tool cover -func=internal.coverage.out

coverage-root-cli:
	@go test -coverprofile=other.coverage.out . ./cmd/smarterr/...
	@go tool cover -func=other.coverage.out

coverage-check-internal:
	@$(MAKE) coverage-internal
	@total=$$(go tool cover -func=internal.coverage.out | grep total: | awk '{print substr($$3, 1, length($$3)-1)}'); \
	min=65.0; \
	awk "BEGIN {exit ($$total < $$min) ? 1 : 0}"

# Please keep targets in alphabetical order
.PHONY: \
	build \
	coverage-check-internal \
	coverage-internal \
	coverage-root-cli \
	fmt \
	fmt-check \
	install \
	lint \
	modern-check \
	modern-fix \
	staticcheck \
	test \
	vet \
	vulncheck

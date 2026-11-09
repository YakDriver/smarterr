SHELL := /bin/bash

# TEMPORARY: The go fix bundled with the current Go release lacks some
# analyzers available in the standalone modernize command. The fix-extra and
# fix-extra-check targets run only the ones go fix is still missing. Once
# `go tool fix help` lists all of MODERNIZE_EXTRA_ANALYZERS, those targets skip
# themselves; remove them (and these variables) at that point.
MODERNIZE_EXTRA_ANALYZERS ?= importcomment reflecttypeassert slicesclip
MODERNIZE_VERSION         ?= v0.52.0
MODERNIZE                 := golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@$(MODERNIZE_VERSION)

# Prints modernize flags (e.g., -slicesclip) for analyzers go fix doesn't have.
MODERNIZE_MISSING_FLAGS = \
	available=$$(go tool fix help | awk '/^Registered analyzers:/{f=1;next} /^By default/{f=0} f && NF{print $$1}') || exit 1; \
	flags=""; \
	for a in $(MODERNIZE_EXTRA_ANALYZERS); do \
		grep -qx "$$a" <<<"$$available" || flags="$$flags -$$a"; \
	done

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

fix: ## Apply go fix plus modernize analyzers missing from go fix
	@echo "make: Applying go fix..."
	@go fix ./...
	@$(MAKE) --no-print-directory fix-extra

fix-check: ## [CI] Go Fix Check / go fix plus modernize analyzers missing from go fix
	@echo "make: Checking go fix..."
	@rc=0; \
	go fix -diff ./... || { echo 'Code needs go fix. Run `make fix`.'; rc=1; }; \
	$(MAKE) --no-print-directory fix-extra-check || rc=1; \
	exit $$rc

fix-extra: ## Apply modernize analyzers missing from go fix (temporary)
	@$(MODERNIZE_MISSING_FLAGS); \
	if [ -z "$$flags" ]; then \
		echo "make: go fix now includes all of: $(MODERNIZE_EXTRA_ANALYZERS). Remove fix-extra targets from GNUmakefile."; \
		exit 0; \
	fi; \
	echo "make: Applying modernize$$flags..."; \
	go run $(MODERNIZE) -fix $$flags ./...

fix-extra-check: ## Check modernize analyzers missing from go fix (temporary)
	@$(MODERNIZE_MISSING_FLAGS); \
	if [ -z "$$flags" ]; then \
		echo "make: go fix now includes all of: $(MODERNIZE_EXTRA_ANALYZERS). Remove fix-extra targets from GNUmakefile."; \
		exit 0; \
	fi; \
	echo "make: Checking modernize$$flags..."; \
	go run $(MODERNIZE) $$flags ./... || { echo 'Code needs modernize fixes. Run `make fix`.'; exit 1; }

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
	fix \
	fix-check \
	fix-extra \
	fix-extra-check \
	fmt \
	fmt-check \
	help \
	install \
	lint \
	staticcheck \
	test \
	tidy \
	tidy-check \
	vet \
	vulncheck

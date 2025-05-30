SHELL := /bin/bash

default: build 

build: 
	@echo "make: Building..."
	@go install

fmt: ## Fix Go source formatting
	@echo "make: Fixing source code with gofmt..."
	gofmt -s -w ./...

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

# Please keep targets in alphabetical order
.PHONY: \
	build \
	fmt \
	install \
	modern-check \
	modern-fix \
	test

.DEFAULT_GOAL := help

# ============================================================================
# Variables
# ============================================================================
GO      := go
VERSION := $(shell git describe --tags --always 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)

# ============================================================================
# Primary targets
# ============================================================================

all: lint test build ## lint + test + build (CI gate)

build: ## Compile all packages
	$(GO) build ./...

test: ## Run tests (fast, no race)
	$(GO) test ./...

test-verbose: ## Run tests with verbose output
	$(GO) test -v ./...

test-race: ## Run tests with race detector
	$(GO) test -race ./...

test-cover: ## Run tests with coverage report
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ============================================================================
# Code quality
# ============================================================================

lint: ## Run linters (golangci-lint + govulncheck)
	golangci-lint run ./...
	@govulncheck ./... 2>/dev/null || echo "WARNING: govulncheck not installed or found issues"

fmt: ## Format code (go fmt + gofumpt)
	$(GO) fmt ./...
	@gofumpt -w . 2>/dev/null || true

check: lint test ## Pre-commit check: lint + test

tidy: ## Tidy go modules
	$(GO) mod tidy

tools: ## Install dev tools (golangci-lint, gofumpt, govulncheck)
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest

# ============================================================================
# Cleanup
# ============================================================================

clean: ## Clean test cache and coverage artifacts
	$(GO) clean -testcache
	rm -f coverage.out coverage.html

# ============================================================================
# Help (auto-generated from ## comments)
# ============================================================================

help: ## Show this help
	@echo "Usage: make <target>"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: all build test test-verbose test-race test-cover lint fmt check tidy tools clean help

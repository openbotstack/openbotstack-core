.PHONY: all build test test-v lint clean tidy

GO := go
MODULE := github.com/openbotstack/openbotstack-core

all: lint test build

build:
	$(GO) build ./...

test:
	$(GO) test -v -race ./...

test-v:
	$(GO) test -v ./...

lint:
	$(GO) vet ./...
	@command -v staticcheck >/dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed"

clean:
	$(GO) clean -cache -testcache

tidy:
	$(GO) mod tidy

# Build Variables
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Linker Flags to inject variables
LDFLAGS := -s -w \
	-X 'github.com/theshedman/shedman/pkg/core/cmd.Version=$(VERSION)' \
	-X 'github.com/theshedman/shedman/pkg/core/cmd.GitCommit=$(COMMIT)' \
	-X 'github.com/theshedman/shedman/pkg/core/cmd.BuildDate=$(DATE)'

# Go command
GO := go

.PHONY: all build install clean test help

all: build

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the shedman binary
	@echo "Building shedman $(VERSION)..."
	$(GO) build -ldflags "$(LDFLAGS)" -o shedman ./cmd/shedman

install: ## Install shedman to GOBIN
	@echo "Installing shedman..."
	$(GO) install -ldflags "$(LDFLAGS)" ./cmd/shedman

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f shedman
	$(GO) clean

test: ## Run tests
	$(GO) test -v ./...

verify: ## Run full verification
	$(GO) vet ./...
	$(GO) test -v ./...

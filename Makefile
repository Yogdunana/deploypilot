.PHONY: build build-mcp test lint coverage clean docker-build run

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
GOVET = $(GOCMD) vet

# Build targets
build: ## Build CLI binary
	$(GOBUILD) $(LDFLAGS) -o bin/deploypilot ./cmd/deploypilot/

build-mcp: ## Build MCP server binary
	$(GOBUILD) $(LDFLAGS) -o bin/mcp-server ./cmd/mcp-server/

build-all: build build-mcp ## Build all binaries

# Test targets
test: ## Run all tests with race detector
	$(GOTEST) -race -count=1 ./...

coverage: ## Generate coverage report
	$(GOTEST) -race -coverprofile=c.out ./...
	$(GOCMD) tool cover -func=c.out
	@echo "Coverage report: c.out"

coverage-html: coverage ## Generate HTML coverage report
	$(GOCMD) tool cover -html=c.out -o coverage.html
	@echo "HTML coverage: coverage.html"

# Quality targets
lint: ## Run golangci-lint
	golangci-lint run ./...

vet: ## Run go vet
	$(GOVET) ./...

check: vet lint test ## Run all checks

# Docker targets
docker-build: ## Build Docker image
	docker build -t deploypilot:$(VERSION) .
	@echo "Built deploypilot:$(VERSION)"

# Run targets
run: build-mcp ## Build and run MCP server
	./bin/mcp-server --config config.yaml

# Clean targets
clean: ## Remove build artifacts
	rm -rf bin/ c.out coverage.html

# Help target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Default target
.DEFAULT_GOAL := help

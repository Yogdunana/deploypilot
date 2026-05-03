.PHONY: build build-mcp build-api build-all test lint coverage clean docker-build run swagger dev dev-up dev-down dev-logs dev-clean dev-backend dev-frontend

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-s -w -X github.com/Yogdunana/deploypilot/internal/version.Version=$(VERSION) -X github.com/Yogdunana/deploypilot/internal/version.GitCommit=$(GIT_COMMIT) -X github.com/Yogdunana/deploypilot/internal/version.BuildTime=$(BUILD_TIME)"

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

build-api: ## Build API server binary
	$(GOBUILD) $(LDFLAGS) -o bin/api-server ./cmd/api-server/

build-all: build build-mcp build-api ## Build all binaries

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

# Swagger targets
swagger: ## Generate Swagger documentation
	swag init -g cmd/api-server/main.go -o docs/swagger

# ─── Development targets ────────────────────────────────────────────────

# Start all dev dependencies (postgres, redis)
dev-up:
	docker compose -f docker-compose.dev.yml up -d

# Stop dev dependencies
dev-down:
	docker compose -f docker-compose.dev.yml down

# View dev logs
dev-logs:
	docker compose -f docker-compose.dev.yml logs -f

# Clean dev volumes
dev-clean:
	docker compose -f docker-compose.dev.yml down -v

# Run backend with hot reload (requires air)
dev-backend:
	air -c .air.toml

# Run frontend dev server
dev-frontend:
	cd web && npm run dev

# Full dev: start deps + backend + frontend
dev: dev-up
	@echo "Dependencies started. Run 'make dev-backend' and 'make dev-frontend' in separate terminals."

# Help target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Default target
.DEFAULT_GOAL := help

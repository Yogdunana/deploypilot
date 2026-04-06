APP_NAME := deploypilot
VERSION := $(shell git describe --tags --always)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build build-mcp test lint vuln run clean

build:
	go build $(LDFLAGS) -o bin/$(APP_NAME) ./cmd/deploypilot/

build-mcp:
	go build $(LDFLAGS) -o bin/mcp-server ./cmd/mcp-server/

test:
	go test -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

vuln:
	govulncheck ./...

run: build
	./bin/$(APP_NAME) serve --config configs/config.yaml

clean:
	rm -rf bin/ coverage.out

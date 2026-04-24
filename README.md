# Deploypilot

AI-native deployment middleware — MCP-powered server deployment for AI IDEs.

Deploypilot bridges AI assistants (like Claude) with your servers via the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/), enabling natural-language-driven container deployment, health monitoring, and rollback.

## Features

- **MCP Server** — Deploy, manage, and monitor containers through AI conversation
- **Preflight Checks** — Automatic validation of SSH connectivity, Docker availability, port conflicts, and TCP reachability before deployment
- **Multi-Server Support** — Manage deployments across multiple servers via SSH
- **Credential Encryption** — AES-256-GCM encrypted credential storage
- **Deployment Records** — Full observability with structured preflight results and deployment history
- **DNS Integration** — Cloudflare DNS record management
- **Notifications** — Webhook and email alerts for deployment events
- **Health Checks** — HTTP/TCP health monitoring with automatic alerting

## Architecture

```
┌──────────────┐     MCP      ┌──────────────────┐    SSH    ┌──────────────┐
│  AI IDE      │ ──────────►  │  Deploypilot     │ ───────► │  Target      │
│  (Claude)    │              │  MCP Server      │          │  Server      │
└──────────────┘              │                  │          │  (Docker)    │
                              │  ┌────────────┐  │          └──────────────┘
                              │  │ Bridge     │  │
                              │  │ Service    │  │
                              │  └────────────┘  │
                              │  ┌────────────┐  │          ┌──────────────┐
                              │  │ Providers  │──┼─────────►│  Cloudflare  │
                              │  └────────────┘  │          │  Webhook     │
                              └──────────────────┘          └──────────────┘
```

## Quick Start

### Prerequisites

- Go 1.23+
- Docker (on target servers)
- SSH access to target servers

### Build

```bash
git clone https://github.com/Yogdunana/deploypilot.git
cd deploypilot
go mod download
go build -o deploypilot ./cmd/deploypilot/
go build -o mcp-server ./cmd/mcp-server/
```

### Configure

Create a config file (default: `config.yaml`):

```yaml
database:
  type: sqlite
  path: ./deploypilot.db

server:
  host: 0.0.0.0
  port: 8080

mcp:
  transport: stdio
```

Or copy the example:
```bash
cp .env.example .env
```

### Run as MCP Server

```bash
./mcp-server
```

Add to your AI IDE's MCP configuration:

```json
{
  "mcpServers": {
    "deploypilot": {
      "command": "./mcp-server",
      "args": ["--config", "config.yaml"]
    }
  }
}
```

### Run as CLI

```bash
./deploypilot serve --config config.yaml
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `deploy_app` | Deploy a Docker container with preflight checks |
| `get_deploy_status` | Get container status with preflight history |
| `remove_app` | Remove a deployed container |
| `rollback` | Rollback to a previous container image |
| `health_check` | Check application health via HTTP/TCP |
| `detect_env` | Detect server environment (OS, CPU, memory, Docker) |
| `add_server` | Register a new target server |
| `list_servers` | List all registered servers |
| `create_app` | Create a new application configuration |
| `list_apps` | List all applications |
| `create_credential` | Store encrypted credentials |
| `backup_app` | Backup application state |

## Development

### Run Tests

```bash
go test -race -count=1 ./...
```

### Run Linter

```bash
golangci-lint run ./...
```

### Coverage Report

```bash
go test -race -coverprofile=c.out ./...
go tool cover -func=c.out
go tool cover -html=c.out -o coverage.html
```

## Project Structure

```
deploypilot/
├── cmd/
│   ├── deploypilot/     # CLI entry point
│   └── mcp-server/      # MCP server entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── crypto/          # AES-256-GCM encryption
│   ├── database/        # Database migrations & seeding
│   ├── engine/
│   │   ├── builder/     # Dockerfile templates
│   │   └── deployer/    # Docker container operations
│   ├── mcp/             # MCP server & tool handlers
│   ├── model/           # GORM models
│   ├── provider/
│   │   ├── dns/         # Cloudflare DNS
│   │   ├── notify/      # Webhook/Email notifications
│   │   └── server/      # SSH client
│   └── service/         # Business logic (Bridge)
├── pkg/errors/          # Error handling utilities
└── tests/e2e/           # End-to-end tests
```

## License

[MIT](LICENSE)

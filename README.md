<p align="center">
  <h1 align="center">Deploypilot</h1>
  <p align="center">
    <strong>AI-native deployment platform</strong> — MCP-powered container deployment, monitoring, and self-healing for AI IDEs
  </p>
  <p align="center">
    <a href="https://github.com/Yogdunana/deploypilot/actions/workflows/ci.yml">
      <img src="https://github.com/Yogdunana/deploypilot/actions/workflows/ci.yml/badge.svg" alt="CI">
    </a>
    <img src="https://img.shields.io/badge/Go-1.23.6-00ADD8?logo=go" alt="Go">
    <img src="https://img.shields.io/badge/License-MIT-green.svg" alt="License">
    <img src="https://img.shields.io/badge/Coverage-79.6%25-yellow" alt="Coverage">
  </p>
</p>

Deploypilot bridges AI assistants with your infrastructure via the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/), enabling natural-language-driven container deployment, real-time monitoring, self-healing, and multi-cloud DNS management — all from your AI IDE.

## ✨ Features

### AI Integration
- **MCP Server** — 40+ tools for deployment, monitoring, DNS, notifications, and diagnostics via stdio transport
- **REST API** — 60+ endpoints with JWT auth and RBAC for programmatic access
- **Natural Language** — Deploy and manage infrastructure through conversation with Claude, TRAE, Cursor, etc.

### Deployment Engine
- **Three Deploy Modes** — Direct deploy, git-based build, and CI/CD trigger
- **Built-in Builder** — Git clone → Docker build → container launch pipeline
- **Preflight Checks** — SSH, Docker, port conflict, and TCP reachability validation
- **Health Checks** — HTTP/TCP probing with configurable retries
- **App Templates** — 9 preset templates (Node.js, Python, Go, Java, Rust, etc.)
- **Backup & Restore** — Full container state backup and one-click rollback

### Operations & Monitoring
- **Self-Healing Engine** — Auto-restart crashed/OOM containers, auto-rollback after max restarts exceeded
- **Monitoring System** — CPU, memory, disk metrics collection with 3-level alerting (critical/warning/info)
- **Agent Reverse Tunnel** — WebSocket tunnel for NAT traversal, no inbound ports required
- **SSH Terminal** — In-browser terminal via WebSocket proxy

### Security & Access Control
- **JWT Authentication** — Token-based auth with configurable expiry
- **RBAC** — Role-based access control (owner > admin > dev > viewer)
- **Credential Encryption** — AES-256-GCM + bcrypt for sensitive data
- **Audit Logging** — All mutating operations logged with user, action, IP, and timestamp
- **Rate Limiting** — Token bucket with role-differentiated limits (50–200 req/min)
- **Security Headers** — X-Content-Type-Options, X-Frame-Options, CSP, Referrer-Policy

### Real-Time Communication
- **WebSocket Log Streaming** — Real-time container log push (`/ws/logs/:app_id`)
- **SSE Deploy Progress** — Server-Sent Events for step-by-step deploy updates
- **Redis Pub/Sub** — Horizontal scaling for multi-instance deployments

### Provider Ecosystem
| Type | Providers |
|------|-----------|
| **DNS** | Cloudflare, Alibaba Cloud (Aliyun), Tencent Cloud (DNSPod) |
| **Notification** | Webhook, Email, Telegram, DingTalk, Feishu/Lark |
| **CI/CD** | GitHub Actions |

### Web Dashboard
- **Vue 3 + Element Plus + Vite** — Modern reactive frontend
- **14 Management Pages** — Dashboard, Apps, Servers, DNS, Credentials, Providers, Notifications, Templates, Deployments, Audit Logs, Users, Roles, System Settings, Monitoring
- **Real-Time Features** — Live log streaming, SSH terminal, deploy progress bar

## 🏗 Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Integration Layer                             │
│   MCP Server (stdio)  │  REST API (JWT+RBAC)  │  WebSocket / SSE   │
├─────────────────────────────────────────────────────────────────────┤
│                        Core Engine                                   │
│   App Management │ Credential Mgmt │ DNS │ Deploy Engine │ Notify   │
│   Health Check   │ Backup/Restore  │ Monitoring │ Self-Healing │ RBAC │
├─────────────────────────────────────────────────────────────────────┤
│                        Provider Plugins                              │
│   ServerProvider (SSH) │ DNSProvider (×3) │ NotifyProvider (×5)     │
│   CICDProvider (GitHub) │ TemplateProvider (×9 presets)              │
├─────────────────────────────────────────────────────────────────────┤
│                        Data Layer                                    │
│   SQLite / PostgreSQL  │  Redis (Pub/Sub)  │  File Storage  │ Metrics│
└─────────────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Go 1.23+
- Docker (on target servers)
- SSH access to target servers
- Node.js 18+ (for web dashboard build)

### Build

```bash
git clone https://github.com/Yogdunana/deploypilot.git
cd deploypilot
go mod download

# Build all three binaries
go build -o mcp-server ./cmd/mcp-server/
go build -o deploypilot ./cmd/deploypilot/
go build -o api-server ./cmd/api-server/
```

Or use Makefile:

```bash
make build-all
```

### Configure

```bash
cp configs/config.yaml.example config.yaml
```

Edit `config.yaml`:

```yaml
database:
  driver: sqlite
  dsn: ./deploypilot.db

server:
  host: 0.0.0.0
  port: 8080

auth:
  jwt_secret: "your-secret-key"
  token_expiry: 24h

deploy:
  health_check_timeout: 60s
  rollback_on_failure: true

monitor:
  enabled: true
  collect_interval: 30s
```

### Run as MCP Server

For AI IDE integration (Claude, TRAE, Cursor, etc.):

```bash
./mcp-server --config config.yaml
```

Add to your AI IDE's MCP configuration:

```json
{
  "mcpServers": {
    "deploypilot": {
      "command": "/path/to/mcp-server",
      "args": ["--config", "config.yaml"]
    }
  }
}
```

### Run as API Server

For REST API + Web Dashboard:

```bash
./api-server --config config.yaml
```

The API server starts at `http://localhost:8080` with:
- REST API at `/api/v1/`
- WebSocket endpoints at `/ws/`
- SSE endpoints at `/sse/`

### Run as CLI

```bash
./deploypilot serve --config config.yaml
```

### Web Dashboard

Build the frontend:

```bash
cd web && npm install && npm run build
```

The built files are served by the API server at `/`. For production, use Nginx as a reverse proxy:

```nginx
server {
    listen 80;
    server_name deploy.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### Docker

```bash
docker build -t deploypilot .
docker run -p 8080:8080 -v ./config.yaml:/app/config.yaml deploypilot
```

## 🔧 MCP Tools

Deploypilot registers **40+ MCP tools** organized by category:

| Category | Tools |
|----------|-------|
| **Deployment** | `deploy_app`, `get_deploy_status`, `rollback_app`, `batch_deploy` |
| **App Management** | `list_apps`, `create_app`, `delete_app`, `update_app`, `get_app_detail` |
| **Server Management** | `list_servers`, `add_server`, `delete_server`, `update_server`, `test_server` |
| **Credentials** | `list_credentials`, `add_credential`, `delete_credential`, `update_credential` |
| **DNS** | `add_dns_record`, `update_dns_record`, `delete_dns_record`, `list_dns_records`, `batch_dns` |
| **Monitoring** | `heal_container`, `get_container_metrics`, `get_system_metrics`, `list_alerts`, `list_alert_rules` |
| **CI/CD** | `trigger_ci_build`, `get_ci_build_status` |
| **Notifications** | `send_notification` |
| **Tasks** | `get_task_status`, `list_tasks` |
| **Templates** | `list_templates`, `get_template` |
| **Diagnostics** | `detect_environment`, `health_check`, `doctor` |

## 📡 REST API

The API server exposes **60+ REST endpoints** under `/api/v1/`:

| Resource | Endpoints | Example |
|----------|-----------|---------|
| **Auth** | 2 | `POST /api/v1/auth/register`, `POST /api/v1/auth/login` |
| **Apps** | 14 | `POST /api/v1/apps`, `POST /api/v1/apps/:id/deploy` |
| **Servers** | 7 | `POST /api/v1/servers`, `POST /api/v1/servers/:id/detect` |
| **Credentials** | 4 | `POST /api/v1/credentials` |
| **DNS** | 4 | `POST /api/v1/dns/records` |
| **Providers** | 4 | `POST /api/v1/providers` |
| **Notifications** | 4 | `POST /api/v1/notifications` |
| **Templates** | 4 | `GET /api/v1/templates` |
| **Users & Roles** | 5 | `GET /api/v1/users`, `PUT /api/v1/users/:id/role` |
| **Audit Logs** | 1 | `GET /api/v1/audit-logs` |
| **System** | 3 | `GET /api/v1/system/health` |
| **Deployments** | 2 | `GET /api/v1/deployments` |
| **Backups** | 2 | `GET /api/v1/apps/:id/backups` |
| **Monitor** | 6 | `GET /api/v1/monitor/system`, `POST /api/v1/monitor/heal/:name` |
| **CI/CD** | 2 | `POST /api/v1/cicd/trigger` |
| **Real-Time** | 3 | `WS /ws/logs/:app_id`, `WS /ws/terminal/:server_id`, `SSE /sse/deploy/:app_id` |

## 📁 Project Structure

```
deploypilot/
├── cmd/
│   ├── api-server/          # REST API server entry point
│   ├── deploypilot/         # CLI entry point (Cobra)
│   └── mcp-server/          # MCP server entry point
├── internal/
│   ├── agent/               # Agent reverse tunnel (WebSocket)
│   ├── api/                 # REST API handlers (Gin)
│   ├── auth/                # JWT authentication + RBAC middleware
│   ├── config/              # Configuration management (Viper)
│   ├── crypto/              # AES-256-GCM encryption
│   ├── database/            # Database migrations & seeding (GORMigrate)
│   ├── engine/
│   │   ├── builder/         # Dockerfile templates (9 presets)
│   │   ├── deployer/        # Docker container operations
│   │   ├── detector/        # Environment detection (OS/Docker/Port/Service)
│   │   └── healer/          # Self-healing engine
│   ├── mcp/                 # MCP server & tool registration
│   ├── middleware/           # HTTP middleware (audit, rate-limit, security headers)
│   ├── model/               # GORM data models
│   ├── monitor/             # Metrics collection & alerting
│   ├── provider/
│   │   ├── cicd/            # CI/CD providers (GitHub Actions)
│   │   ├── dns/             # DNS providers (Cloudflare, Aliyun, Tencent)
│   │   ├── notify/          # Notification providers (Webhook, Email, Telegram, DingTalk, Feishu)
│   │   ├── registry/        # Container registry (planned)
│   │   └── server/          # SSH client with PTY support
│   ├── server/              # HTTP server setup
│   └── service/             # Business logic (Bridge — 46 methods)
├── web/                     # Vue 3 + Element Plus + Vite frontend
│   └── src/
│       ├── api/             # Axios API client
│       ├── views/           # 14 page components
│       ├── router/          # Vue Router
│       └── layout/          # Main layout with sidebar
├── configs/                 # Configuration templates
├── scripts/                 # Build & deployment scripts
├── tests/e2e/               # End-to-end tests
├── docs/                    # Documentation
├── pkg/errors/              # Error handling utilities
├── .golangci.yml            # Linter configuration
├── Dockerfile               # Multi-stage build (3 binaries)
├── Makefile                 # 13 build targets
└── go.mod                   # Go module definition
```

## 🛠 Development

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
make coverage
# or manually:
go test -race -coverprofile=c.out ./...
go tool cover -func=c.out
go tool cover -html=c.out -o coverage.html
```

### Makefile Targets

```bash
make build          # Build deploypilot CLI
make build-mcp      # Build MCP server
make build-api      # Build API server
make build-all      # Build all binaries
make test           # Run tests with race detection
make coverage       # Generate coverage report
make lint           # Run golangci-lint
make vet            # Run go vet
make check          # vet + lint + test
make docker-build   # Build Docker image
make clean          # Remove build artifacts
```

## Roadmap

- [ ] Structured logging (`log.Printf` → `slog`)
- [ ] MCP permission control (role-based tool access)
- [ ] OpenAPI / Swagger documentation
- [ ] SSL certificate management (Let's Encrypt)
- [ ] 1Panel / BT-Panel integration
- [ ] MCP context memory (session-scoped operation history)

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Write tests for your changes
4. Ensure `make check` passes (vet + lint + test)
5. Commit with conventional commits (`feat:`, `fix:`, `docs:`, etc.)
6. Push and open a Pull Request

## License

[MIT](LICENSE)

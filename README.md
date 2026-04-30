<p align="center">
  <img src="docs/logo/logo.svg" alt="DeployPilot" width="280">
</p>

<h1 align="center">DeployPilot</h1>

<p align="center">
  <strong>The AI-native deployment gateway</strong> — bridge sandboxed AI IDEs to your infrastructure.
</p>

<p align="center">
  <a href="#quick-start"><b>Quick Start</b></a> &middot;
  <a href="#features"><b>Features</b></a> &middot;
  <a href="#architecture"><b>Architecture</b></a> &middot;
  <a href="#supported-ai-ides"><b>AI IDEs</b></a> &middot;
  <a href="#documentation"><b>Docs</b></a> &middot;
  <a href="#contributing"><b>Contributing</b></a>
</p>

<p align="center">
  <a href="README_zh-CN.md">中文文档</a>
</p>

<p align="center">
  <img src="https://img.shields.io/github/actions/workflow/status/Yogdunana/deploypilot/ci.yml?branch=main&amp;style=flat-square" alt="CI">
  <img src="https://img.shields.io/github/v/release/Yogdunana/deploypilot?style=flat-square" alt="Release">
  <img src="https://img.shields.io/badge/License-BSL_1.1-blue?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&amp;logo=go&amp;logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/Vue-3.5-4FC08D?style=flat-square&amp;logo=vue.js&amp;logoColor=white" alt="Vue">
  <img src="https://img.shields.io/docker/pulls/ghcr.io/yogdunana/deploypilot?style=flat-square" alt="Docker Pulls">
</p>

---

## The Problem

AI coding assistants such as **Claude**, **Cursor**, **TRAE**, **Coze**, and **SOLO** are powerful, but they all share a fundamental limitation: they run inside sandboxed cloud environments with **no direct SSH access** to your servers.

This means your AI assistant cannot:

- Deploy applications to production servers
- Configure reverse proxies or allocate ports
- Manage DNS records across providers
- Provision or renew SSL certificates
- Open firewall rules on server management panels

The situation is even harder in enterprise environments where servers sit behind **bastion hosts** or **jump servers** that cloud-based IDEs cannot reach at all. Manual deployment remains tedious, error-prone, and wastes developer time.

**DeployPilot bridges this gap** by running on your server and exposing a standard **MCP (Model Context Protocol)** interface that any AI IDE can call — securely, remotely, and autonomously.

---

## What is DeployPilot?

DeployPilot is a **self-hosted AI IDE deployment gateway** that uses the MCP protocol to let sandboxed AI assistants manage your server infrastructure. It installs in one line and gives your AI IDE full control over deployment, DNS, SSL, monitoring, and more.

> **One command to install. One prompt to deploy.**

```bash
docker compose up -d
```

Then tell your AI:

> *"Deploy my project, set up DNS, and provision SSL."*

DeployPilot handles the rest.

---

## Quick Start

### Docker (recommended)

```bash
# Clone the repository
git clone https://github.com/Yogdunana/deploypilot.git
cd deploypilot

# Set required environment variables
export JWT_SECRET=$(openssl rand -base64 24)
export REDIS_PASSWORD=$(openssl rand -base64 24)

# Start the service
docker compose up -d
```

Open **http://localhost:8080** and register an admin account.

### One-line install (binary)

```bash
curl -fsSL https://raw.githubusercontent.com/Yogdunana/deploypilot/main/scripts/install.sh | bash
```

The install script automatically downloads the latest binary, generates admin credentials, and sets up systemd services.

### From source

```bash
git clone https://github.com/Yogdunana/deploypilot.git
cd deploypilot &amp;&amp; make build-all
```

See [DEPLOY.md](DEPLOY.md) for the full deployment guide, including PostgreSQL setup, reverse proxy configuration, and production hardening.

---

## Architecture

DeployPilot uses a **three-layer architecture** designed for both AI agents and human operators:

```mermaid
graph LR
    subgraph "AI IDE (Sandbox)"
        A[Claude / Cursor / TRAE / Coze / SOLO]
    end

    subgraph "DeployPilot Gateway (Your Server)"
        B[MCP Server&lt;br&gt;52+ tools]
        C[REST API&lt;br&gt;JWT + RBAC]
        D[WebSocket / SSE]
        E[Deploy Engine]
        F[Provider Plugins]
    end

    subgraph "Infrastructure"
        G[1Panel / BT Panel]
        H[Docker / Kubernetes]
        I[Cloudflare / Aliyun DNS]
        J[GitHub Actions / Gitea]
    end

    A -- "MCP (stdio)" --&gt; B
    A -- "REST + JWT" --&gt; C
    C --&gt; E
    B --&gt; E
    E --&gt; F
    F --&gt; G
    F --&gt; H
    F --&gt; I
    F --&gt; J
```

| Layer | Protocol | Purpose |
|-------|----------|----------|
| **MCP Server** | stdio | Native AI IDE integration — 52+ tools for deployment, DNS, SSL, monitoring |
| **REST API** | HTTP + JWT | Programmatic access with full RBAC — 68+ endpoints, Swagger docs at `/swagger/` |
| **WebSocket / SSE** | ws://, text/event-stream | Real-time log streaming, SSH terminal, deployment progress |

The embedded **web dashboard** (Vue 3 + TypeScript + Tailwind CSS) provides a full management UI accessible at `http://localhost:8080`.

---

## Features

### MCP Protocol Integration

52+ MCP tools covering the complete deployment lifecycle. AI IDEs connect via stdio transport and can autonomously manage your infrastructure through natural language.

### Multi-Server Management

Register and manage multiple servers. Supports SSH-based remote servers, Docker (local and remote), and Kubernetes multi-cluster deployments. Detects server environments and installed panels automatically.

### Docker Container Lifecycle

Full container management — create, deploy, start, stop, remove, and monitor containers. Supports automatic port allocation, health checks with configurable retries, backup and rollback, and self-healing with auto-restart and auto-rollback on failure thresholds.

### DNS Management

Unified DNS record management across multiple providers:

| Provider | ID |
|----------|----|
| Cloudflare | `dns-cloudflare` |
| Alibaba Cloud DNS | `dns-aliyun` |
| Tencent Cloud DNSPod | `dns-tencent` |
| WestDNS / west.cn | `dns-west-dns` |

### SSL Certificate Automation

Request, list, renew, and delete SSL certificates directly from the MCP interface or web dashboard.

### CI/CD Pipeline Integration

Trigger and monitor CI/CD builds without leaving your AI IDE. Supports GitHub Actions and Gitea.

### Monitoring &amp; Self-Healing

Real-time container and system metrics, configurable alert rules, and automatic container healing when failures are detected.

### OAuth Login

Sign in with your GitHub or Gitee account — no separate password needed.

### RBAC Authorization

Four-tier role-based access control: **owner &gt; admin &gt; dev &gt; viewer**. Fine-grained permissions on every resource.

### Audit Logging

Full change tracking with user identity, action type, IP address, and timestamp. Every operation is recorded and auditable.

### i18n

Web dashboard available in English and Chinese, with a pluggable locale system.

---

## Supported AI IDEs

DeployPilot integrates with any IDE that supports the MCP protocol. Configuration details for each IDE are available in [docs/ide-skills.md](docs/ide-skills.md).

| IDE | Transport | Setup |
|-----|-----------|-------|
| **Claude Desktop** | MCP stdio | Add to `claude_desktop_config.json` |
| **Cursor** | MCP stdio | Add in Settings &gt; MCP |
| **TRAE** | MCP stdio | Add in IDE MCP settings |
| **Coze / 扣子** | MCP stdio | Configure as MCP plugin |
| **SOLO** | MCP stdio | Add to MCP server config |

All IDEs use the same configuration pattern:

```json
{
  "mcpServers": {
    "deploypilot": {
      "command": "/path/to/mcp-server",
      "args": ["--config", "/path/to/config.yaml"]
    }
  }
}
```

See [docs/ide-skills.md](docs/ide-skills.md) for IDE-specific instructions, skill files, and rules templates.

---

## Configuration

DeployPilot is configured via `config.yaml`. All settings can also be overridden with environment variables using the `DEPLOYPILOT_` prefix (e.g., `DEPLOYPILOT_SERVER_PORT` for `server.port`).

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  cors_allowed_origins:
    - "*"

database:
  type: sqlite              # sqlite | postgres
  dsn: ./data/deploypilot.db

redis:
  addr: localhost:6379
  password: ""
  db: 0

auth:
  jwt_secret: "change-me-in-production"  # Required: set a strong random value
  token_expire: 24h

deploy:
  default_mode: api
  build_timeout: 10m
  health_check_interval: 30s
  health_check_retries: 3
  rollback_on_failure: true

log:
  level: info               # debug | info | warn | error
  format: json              # json | text
  file: ./logs/deploypilot.log
  enable_tracing: true

monitor:
  enabled: true
  metrics_port: 9091
```

See [configs/config.yaml.example](configs/config.yaml.example) for the full example and [DEPLOY.md](DEPLOY.md) for all environment variables.

---

## Documentation

| Document | Description |
|----------|-------------|
| [docs/PRD.md](docs/PRD.md) | Product Requirements Document |
| [docs/ide-skills.md](docs/ide-skills.md) | IDE integration guide with setup instructions for each AI IDE |
| [docs/troubleshooting.md](docs/troubleshooting.md) | Troubleshooting guide for common issues |
| [DEPLOY.md](DEPLOY.md) | Full deployment guide (Docker, binary, source, reverse proxy, production) |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contributing guidelines |

---

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, coding conventions, and pull request guidelines.

---

## License

This project is licensed under the **Business Source License 1.1 (BSL 1.1)**.

&gt; **Non-commercial use is permitted** — including personal use, internal use within organizations, testing/evaluation, and non-commercial educational use. See [LICENSE](LICENSE) for full terms.
&gt;
&gt; On **2029-04-28**, this license automatically converts to [MIT](https://opensource.org/licenses/MIT).

&amp;copy; 2026 Yogdunana
# Architecture

## Overview

DeployPilot uses a three-layer architecture: **MCP + REST + WebSocket/SSE**.

```
┌──────────────────────────────────────────────────┐
│                  AI IDE (Sandbox)                 │
│   TRAE Solo / Claude / Cursor / Coze / SOLO      │
└────────────────────┬─────────────────────────────┘
                     │ MCP Protocol (stdio / HTTP)
                     ▼
┌──────────────────────────────────────────────────┐
│              DeployPilot (Your Server)            │
│                                                    │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │
│  │MCP Server│ │REST API  │ │WebSocket / SSE   │  │
│  │(52+tools)│ │(68+endpts)│ │(logs/progress)   │  │
│  └────┬─────┘ └────┬─────┘ └────────┬─────────┘  │
│       └────────────┼────────────────┘             │
│                    ▼                               │
│  ┌─────────────────────────────────────────────┐  │
│  │           Bridge Service (Business Logic)    │  │
│  └─────┬─────┬─────┬─────┬─────┬─────┬────────┘  │
│        ▼     ▼     ▼     ▼     ▼     ▼            │
│  ┌─────┐┌────┐┌───┐┌────┐┌────┐┌────┐┌──────┐  │
│  │Deploy││DNS ││SSL││Mon ││Notif││Heal││Audit │  │
│  │Engine││Mgr ││Cert││Alert││Push││Engine││Log  │  │
│  └──┬──┘└──┬─┘└─┬─┘└──┬─┘└──┬─┘└──┬─┘└──┬───┘  │
│     └─────┴────┴────┴────┴────┴────┴────┘       │
│                    ▼                               │
│  ┌─────────────────────────────────────────────┐  │
│  │           Provider Plugin Layer              │  │
│  │ Server │ DNS │ SSL │ Notify │ CICD │ Registry│  │
│  └─────────────────────────────────────────────┘  │
│                    ▼                               │
│  ┌─────────────────────────────────────────────┐  │
│  │           Data Layer                         │  │
│  │ SQLite/PG │ Redis │ File Storage │ Metrics  │  │
│  └─────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────┘
```

## Components

### MCP Server
- Exposes 52+ tools via Model Context Protocol
- Supports stdio transport (local) and HTTP transport (remote)
- Used by AI IDEs to manage deployments

### REST API
- 68+ endpoints for comprehensive server management
- JWT authentication with argon2id/bcrypt password hashing
- Role-based access control (Owner, Admin, Developer, Viewer)
- Swagger documentation at `/swagger/index.html`

### WebSocket / SSE
- Real-time container log streaming
- Deployment progress updates via Server-Sent Events
- Terminal emulation (xterm.js)

### Web Dashboard
- Vue 3 + TypeScript single-page application
- Embedded into the Go binary
- 27 pages covering all features
- i18n support (English / Chinese)

### Panel HTTP Server

DeployPilot 的 Web Dashboard 由 Go 二进制直接提供服务，**不依赖** Apache、Nginx 或 OpenResty 作为前端服务器。

- **HTTP 服务器**: Go `net/http.Server` + Gin 框架
- **前端嵌入**: Vue SPA 构建产物通过 Go `embed` 嵌入二进制（`web/embed.go`）
- **静态文件**: 由 `internal/server/server.go` 的 `serveStaticFiles()` 提供 SPA fallback
- **端口配置**: 通过 `server.port` 配置项可调（默认 8080，非固定）
- **端口管理**: 用户可通过 `deploypilot reset port --port <新端口>` 修改

### Provider Layer
- Pluggable architecture for external services
- DNS: Cloudflare, Alibaba Cloud, Tencent Cloud, WestDNS
- Notifications: Telegram, DingTalk, Feishu, WeCom, Webhook
- Panels: 1Panel, BaoTa
- CI/CD: GitHub Actions, Gitea

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.23, Gin, GORM |
| Frontend | Vue 3.5, TypeScript, Vite |
| Database | SQLite / PostgreSQL |
| Cache | Redis |
| Protocol | MCP (Model Context Protocol) |
| Security | JWT, argon2id, AES-256-GCM, RBAC |

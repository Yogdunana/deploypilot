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

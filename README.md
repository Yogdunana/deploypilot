<p align="center">
  <h1 align="center">DeployPilot</h1>
  <p align="center">
    <strong>AI 原生部署管理平台</strong> — 基于 MCP 协议，用自然语言驱动容器部署、监控与自愈
  </p>
  <p align="center">
    <a href="https://github.com/Yogdunana/deploypilot/actions/workflows/ci.yml">
      <img src="https://github.com/Yogdunana/deploypilot/actions/workflows/ci.yml/badge.svg" alt="CI">
    </a>
    <a href="https://github.com/Yogdunana/deploypilot/actions/workflows/docker.yml">
      <img src="https://github.com/Yogdunana/deploypilot/actions/workflows/docker.yml/badge.svg" alt="Docker">
    </a>
    <img src="https://img.shields.io/badge/Go-1.23-00ADD8?logo=go" alt="Go">
    <img src="https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vue.js" alt="Vue">
    <img src="https://img.shields.io/badge/License-MIT-green.svg" alt="License">
    <img src="https://img.shields.io/badge/Coverage-90%25-brightgreen" alt="Coverage">
    <img src="https://img.shields.io/badge/Platform-amd64%20%7C%20arm64-blue" alt="Multi-arch">
  </p>
</p>

---

DeployPilot 通过 [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) 将 AI 助手与你的基础设施连接起来。在 Claude、Cursor、Windsurf 等 AI IDE 中用自然语言即可完成容器部署、实时监控、自动修复和多云 DNS 管理。

## ✨ 核心特性

### 🤖 AI 集成
- **MCP Server** — 37 个工具，覆盖部署、监控、DNS、通知和诊断，支持 stdio 传输
- **REST API** — 68 个端点，JWT 认证 + RBAC 权限控制
- **Swagger 文档** — 内置交互式 API 文档 (`/swagger/`)
- **自然语言操作** — 在 AI IDE 中对话式管理基础设施

### 🚀 部署引擎
- **三种部署模式** — 直接部署、Git 构建、CI/CD 触发
- **内置构建器** — Git clone → Docker build → 容器启动全流程
- **预检机制** — SSH 连通性、Docker 可用性、端口冲突、TCP 可达性验证
- **健康检查** — HTTP/TCP 探针，可配置重试策略
- **应用模板** — 9 种预设模板（Node.js、Python、Go、Java、Rust 等）
- **备份与回滚** — 完整容器状态备份，一键回滚

### 📊 运维与监控
- **自愈引擎** — 自动重启崩溃/OOM 容器，超过最大重启次数后自动回滚
- **监控体系** — CPU、内存、磁盘指标采集，三级告警（严重/警告/信息）
- **Agent 反向隧道** — WebSocket 隧道穿透 NAT，无需开放入站端口
- **SSH 终端** — 浏览器内终端，基于 xterm.js + WebSocket 代理

### 🔒 安全与权限
- **JWT 认证** — Token 认证，可配置过期时间
- **RBAC 权限** — 四级角色（owner > admin > dev > viewer）
- **凭据加密** — AES-256-GCM + bcrypt，数据库中不存明文
- **审计日志** — 所有变更操作记录用户、动作、IP 和时间戳
- **限流保护** — 令牌桶算法，按角色分级（50–200 req/min）
- **安全头** — X-Content-Type-Options、X-Frame-Options、CSP、Referrer-Policy

### 🔌 实时通信
- **WebSocket 日志流** — 容器日志实时推送 (`/ws/logs/:app_id`)
- **SSE 部署进度** — Server-Sent Events 逐步推送部署状态
- **Redis Pub/Sub** — 支持多实例水平扩展

### 🌐 服务商生态
| 类型 | 服务商 |
|------|--------|
| **DNS** | Cloudflare、阿里云、腾讯云 (DNSPod) |
| **通知** | Webhook、Email、Telegram、钉钉、飞书/Lark |
| **CI/CD** | GitHub Actions |

### 💻 Web 管理面板
- **Vue 3 + TypeScript + Tailwind CSS 4** — 现代响应式前端
- **27 个页面** — 仪表盘、应用管理、服务器管理、DNS、凭据、服务商、通知、模板、部署记录、审计日志、用户管理、系统设置、监控告警、SSL 证书、CI/CD 等
- **实时功能** — 实时日志流、SSH 终端、部署进度条、指标轮询

---

## 🏗 架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                        接入层                                       │
│   MCP Server (stdio)  │  REST API (JWT+RBAC)  │  WebSocket / SSE   │
├─────────────────────────────────────────────────────────────────────┤
│                        核心引擎                                      │
│   应用管理 │ 凭据管理 │ DNS │ 部署引擎 │ 通知 │ 健康检查             │
│   备份恢复 │ 监控告警 │ 自愈 │ SSL 证书 │ 审计 │ RBAC               │
├─────────────────────────────────────────────────────────────────────┤
│                        服务商插件                                     │
│   ServerProvider (SSH) │ DNSProvider (×3) │ NotifyProvider (×5)     │
│   CICDProvider (GitHub) │ TemplateProvider (×9 预设)                 │
├─────────────────────────────────────────────────────────────────────┤
│                        数据层                                        │
│   SQLite / PostgreSQL  │  Redis (Pub/Sub)  │  文件存储  │ 指标存储   │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 🚀 快速开始

### Docker 一键部署（推荐）

```bash
# 克隆仓库
git clone https://github.com/Yogdunana/deploypilot.git
cd deploypilot

# 启动服务
docker compose up -d

# 访问 http://localhost:8080，注册管理员账号即可使用
```

> 也可以直接拉取预构建镜像：
> ```bash
> docker run -d -p 8080:8080 -v deploypilot-data:/app/data ghcr.io/yogdunana/deploypilot:latest
> ```

### 从源码构建

**前置要求**：Go 1.23+、Node.js 20+

```bash
# 1. 克隆并构建前端
git clone https://github.com/Yogdunana/deploypilot.git
cd deploypilot
cd web && npm ci && npm run build && cd ..

# 2. 构建后端（三个二进制文件）
go build -o deploypilot ./cmd/deploypilot/
go build -o api-server ./cmd/api-server/
go build -o mcp-server ./cmd/mcp-server/

# 3. 或者使用 Makefile
make build-all
```

### 配置

```bash
cp configs/config.yaml.example config.yaml
```

核心配置项：

```yaml
database:
  driver: sqlite          # 可选 postgres
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

> 所有配置项均支持 `DEPLOYPILOT_` 前缀的环境变量覆盖，详见 [DEPLOY.md](DEPLOY.md)。

### 运行

```bash
# API 服务器（REST API + Web 面板）
./api-server --config config.yaml

# MCP 服务器（AI IDE 集成）
./mcp-server --config config.yaml

# CLI 工具
./deploypilot serve --config config.yaml
```

### MCP 集成配置

在 Claude Desktop、Cursor、Windsurf 等 AI IDE 中添加：

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

### 反向代理

生产环境建议使用 Nginx 或 Caddy 反向代理：

```nginx
server {
    listen 443 ssl;
    server_name deploy.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

---

## 🔧 MCP 工具

DeployPilot 注册了 **37 个 MCP 工具**，按类别组织：

| 类别 | 工具 |
|------|------|
| **部署** | `deploy_app`, `get_deploy_status`, `rollback_app`, `batch_deploy` |
| **应用管理** | `list_apps`, `get_app_detail`, `create_app`, `update_app`, `delete_app` |
| **服务器管理** | `list_servers`, `add_server`, `update_server`, `delete_server`, `test_server` |
| **凭据管理** | `list_credentials`, `add_credential`, `update_credential`, `delete_credential` |
| **DNS 管理** | `list_dns_records`, `add_dns_record`, `update_dns_record`, `delete_dns_record`, `batch_dns` |
| **监控** | `heal_container`, `get_container_metrics`, `get_system_metrics`, `list_alerts`, `list_alert_rules` |
| **CI/CD** | `trigger_ci_build`, `get_ci_build_status` |
| **通知** | `send_notification` |
| **模板** | `list_templates`, `get_template` |
| **任务** | `get_task_status`, `list_tasks` |
| **诊断** | `detect_environment`, `health_check`, `doctor` |

---

## 📡 REST API

API 服务器在 `/api/v1/` 下暴露 **68 个端点**：

| 资源 | 端点数 | 示例 |
|------|--------|------|
| **认证** | 2 | `POST /api/v1/auth/register`, `POST /api/v1/auth/login` |
| **应用** | 14 | `POST /api/v1/apps`, `POST /api/v1/apps/:id/deploy` |
| **服务器** | 7 | `POST /api/v1/servers`, `POST /api/v1/servers/:id/detect` |
| **凭据** | 4 | `POST /api/v1/credentials` |
| **DNS** | 4 | `POST /api/v1/dns/records` |
| **服务商** | 4 | `POST /api/v1/providers` |
| **通知** | 4 | `POST /api/v1/notifications` |
| **模板** | 4 | `GET /api/v1/templates` |
| **用户与角色** | 5 | `GET /api/v1/users`, `PUT /api/v1/users/:id/role` |
| **审计日志** | 1 | `GET /api/v1/audit-logs` |
| **系统** | 3 | `GET /api/v1/system/health` |
| **部署记录** | 2 | `GET /api/v1/deployments` |
| **备份** | 2 | `GET /api/v1/apps/:id/backups` |
| **监控** | 6 | `GET /api/v1/monitor/system`, `POST /api/v1/monitor/heal/:name` |
| **CI/CD** | 2 | `POST /api/v1/cicd/trigger` |
| **SSL** | 4 | `GET /api/v1/ssl/certificates` |
| **实时通信** | 4 | `WS /ws/logs/:app_id`, `WS /ws/terminal/:server_id`, `SSE /sse/deploy/:app_id` |

启动后访问 `/swagger/` 查看完整的交互式 API 文档。

---

## 📁 项目结构

```
deploypilot/
├── cmd/
│   ├── api-server/          # REST API + Web 面板入口
│   ├── deploypilot/         # CLI 工具入口 (Cobra)
│   └── mcp-server/          # MCP Server 入口
├── internal/
│   ├── agent/               # Agent 反向隧道 (WebSocket)
│   ├── api/                 # REST API 处理器 (Gin)
│   ├── auth/                # JWT 认证 + RBAC 中间件
│   ├── config/              # 配置管理 (Viper)
│   ├── crypto/              # AES-256-GCM 加密
│   ├── database/            # 数据库迁移与种子 (GORMigrate)
│   ├── engine/
│   │   ├── builder/         # Dockerfile 模板 (9 种预设)
│   │   ├── deployer/        # Docker 容器操作
│   │   ├── detector/        # 环境检测 (OS/Docker/端口/服务)
│   │   └── healer/          # 自愈引擎
│   ├── mcp/                 # MCP Server & 工具注册
│   ├── middleware/           # HTTP 中间件 (审计、限流、安全头)
│   ├── model/               # GORM 数据模型
│   ├── monitor/             # 指标采集 & 告警
│   ├── provider/
│   │   ├── cicd/            # CI/CD (GitHub Actions)
│   │   ├── dns/             # DNS (Cloudflare, 阿里云, 腾讯云)
│   │   ├── notify/          # 通知 (Webhook, Email, Telegram, 钉钉, 飞书)
│   │   ├── registry/        # 容器镜像仓库 (规划中)
│   │   └── server/          # SSH 客户端 (PTY 支持)
│   ├── server/              # HTTP 服务器 & 静态文件
│   └── service/             # 业务逻辑层 (Bridge — 46 个方法)
├── web/                     # Vue 3 + TypeScript + Tailwind CSS 前端
│   ├── src/
│   │   ├── api/modules/     # 15 个 API 模块
│   │   ├── components/      # 22 个 UI 组件 + 8 个业务组件
│   │   ├── composables/     # useWebSocket, useSSE, usePolling
│   │   ├── views/           # 27 个页面组件
│   │   ├── stores/          # Pinia 状态管理
│   │   └── router/          # Vue Router
│   └── embed.go             # Go embed 嵌入前端构建产物
├── configs/                 # 配置文件模板
├── docs/                    # Swagger 文档 & MCP 工具规范
├── scripts/                 # 构建 & 部署脚本
├── tests/e2e/               # 端到端测试
├── pkg/errors/              # 错误处理工具
├── .github/workflows/       # CI/CD (测试、Lint、Docker 多架构构建)
├── Dockerfile               # 三阶段构建 (Node → Go → Alpine)
├── docker-compose.yml       # 生产部署
├── docker-compose.dev.yml   # 开发环境
├── Makefile                 # 14 个构建目标
└── go.mod                   # Go 模块定义
```

---

## 🛠 技术栈

| 层级 | 技术 |
|------|------|
| **后端** | Go 1.23、Gin、GORM、Cobra、Viper |
| **前端** | Vue 3.5、TypeScript 5.6、Vite 6、Tailwind CSS 4、Pinia、Radix Vue |
| **数据库** | SQLite（默认）/ PostgreSQL |
| **缓存** | Redis（可选，Pub/Sub 水平扩展） |
| **协议** | MCP (stdio)、REST、WebSocket、SSE |
| **安全** | JWT、AES-256-GCM、bcrypt、RBAC |
| **部署** | Docker、Docker Compose、GitHub Actions、GHCR |
| **测试** | Go testing、golangci-lint、govulncheck |

---

## 🛠 开发

### 运行测试

```bash
go test -race -count=1 ./...
```

### 代码检查

```bash
# Lint
golangci-lint run ./...

# 安全漏洞扫描
govulncheck ./...

# 一键检查（vet + lint + test）
make check
```

### 覆盖率

```bash
make coverage
# 生成 coverage.html 可视化报告
make coverage-html
```

### Swagger 文档

```bash
make swagger
# 等同于: swag init -g cmd/api-server/main.go -o docs/swagger
```

### Makefile 目标

```bash
make build          # 构建 CLI 工具
make build-mcp      # 构建 MCP Server
make build-api      # 构建 API Server
make build-all      # 构建所有二进制
make test           # 运行测试（race 检测）
make coverage       # 生成覆盖率报告
make coverage-html  # 生成 HTML 覆盖率报告
make lint           # 运行 golangci-lint
make vet            # 运行 go vet
make check          # vet + lint + test
make swagger        # 生成 Swagger 文档
make docker-build   # 构建 Docker 镜像
make clean          # 清理构建产物
```

---

## 🗺 路线图

- [ ] MCP 上下文记忆（会话级操作历史）
- [ ] 容器镜像仓库管理（Docker Hub、GHCR）
- [ ] 1Panel / 宝塔面板集成
- [ ] 多集群 Kubernetes 支持
- [ ] Prometheus / Grafana 指标导出
- [ ] 移动端适配

---

## 🤝 贡献

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/my-feature`)
3. 编写测试
4. 确保 `make check` 通过（vet + lint + test）
5. 使用 Conventional Commits 提交 (`feat:`, `fix:`, `docs:` 等)
6. Push 并创建 Pull Request

---

## 📄 许可证

[MIT](LICENSE) © 2026 Yogdunana

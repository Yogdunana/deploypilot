# DeployPilot 部署文档

## 快速开始（Docker 部署）

### 前置要求

- Docker 20.10+
- Docker Compose V2+

### 一键部署

```bash
# 克隆仓库
git clone https://github.com/Yogdunana/deploypilot.git
cd deploypilot

# 启动服务
docker compose up -d

# 查看日志
docker compose logs -f
```

访问 http://localhost:8080

### 首次使用

1. 打开 http://localhost:8080
2. 注册管理员账号
3. 登录后进入仪表盘

---

## 配置说明

DeployPilot 使用 [Viper](https://github.com/spf13/viper) 管理配置，支持 YAML 配置文件和环境变量两种方式。环境变量使用 `DEPLOYPILOT_` 前缀，层级用 `_` 分隔（例如 `DEPLOYPILOT_SERVER_PORT` 对应 `server.port`）。

### 配置文件方式

```bash
cp configs/config.yaml.example config.yaml
```

编辑 `config.yaml`：

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  type: "sqlite"
  dsn: "data/deploypilot.db"

auth:
  jwt_secret: "change-me-in-production"
  token_expire: "24h"

log:
  level: "info"
  format: "json"

monitor:
  enabled: true
```

### 环境变量

| 环境变量 | 对应配置项 | 默认值 | 说明 |
|---------|-----------|--------|------|
| `DEPLOYPILOT_SERVER_HOST` | `server.host` | `0.0.0.0` | 监听地址 |
| `DEPLOYPILOT_SERVER_PORT` | `server.port` | `8080` | 服务端口 |
| `DEPLOYPILOT_SERVER_MCP_PORT` | `server.mcp_port` | `9090` | MCP 服务端口 |
| `DEPLOYPILOT_SERVER_WEB_PORT` | `server.web_port` | `3000` | Web 开发服务器端口 |
| `DEPLOYPILOT_SERVER_CORS_ALLOWED_ORIGINS` | `server.cors_allowed_origins` | `["*"]` | CORS 允许的来源 |
| `DEPLOYPILOT_DATABASE_TYPE` | `database.type` | `sqlite` | 数据库驱动（`sqlite` / `postgres`） |
| `DEPLOYPILOT_DATABASE_DSN` | `database.dsn` | `./data/deploypilot.db` | 数据库连接字符串 |
| `DEPLOYPILOT_AUTH_JWT_SECRET` | `auth.jwt_secret` | `change-me-in-production` | JWT 签名密钥（**生产环境必须修改**） |
| `DEPLOYPILOT_AUTH_TOKEN_EXPIRE` | `auth.token_expire` | `24h` | Token 过期时间 |
| `DEPLOYPILOT_AUTH_WS_TICKET_EXPIRE` | `auth.ws_ticket_expire` | `30s` | WebSocket Ticket 过期时间 |
| `DEPLOYPILOT_DEPLOY_DEFAULT_MODE` | `deploy.default_mode` | `api` | 默认部署模式 |
| `DEPLOYPILOT_DEPLOY_BUILD_TIMEOUT` | `deploy.build_timeout` | `10m` | 构建超时时间 |
| `DEPLOYPILOT_DEPLOY_HEALTH_CHECK_INTERVAL` | `deploy.health_check_interval` | `30s` | 健康检查间隔 |
| `DEPLOYPILOT_DEPLOY_HEALTH_CHECK_RETRIES` | `deploy.health_check_retries` | `3` | 健康检查重试次数 |
| `DEPLOYPILOT_DEPLOY_ROLLBACK_ON_FAILURE` | `deploy.rollback_on_failure` | `true` | 失败时自动回滚 |
| `DEPLOYPILOT_LOG_LEVEL` | `log.level` | `info` | 日志级别（`debug` / `info` / `warn` / `error`） |
| `DEPLOYPILOT_LOG_FORMAT` | `log.format` | `json` | 日志格式（`json` / `console`） |
| `DEPLOYPILOT_LOG_FILE` | `log.file` | `./logs/deploypilot.log` | 日志文件路径 |
| `DEPLOYPILOT_MONITOR_ENABLED` | `monitor.enabled` | `true` | 是否启用监控 |
| `DEPLOYPILOT_MONITOR_METRICS_PORT` | `monitor.metrics_port` | `9091` | Metrics 端口 |
| `DEPLOYPILOT_REDIS_ADDR` | `redis.addr` | `localhost:6379` | Redis 地址（可选，用于 Pub/Sub） |
| `DEPLOYPILOT_REDIS_PASSWORD` | `redis.password` | `` | Redis 密码 |
| `DEPLOYPILOT_REDIS_DB` | `redis.db` | `0` | Redis 数据库编号 |
| `DEPLOYPILOT_ENCRYPTION_KEY` | - | - | 凭据加密密钥（不设置则每次重启生成临时密钥） |

### 使用 PostgreSQL

修改 docker-compose.yml 环境变量：

```yaml
environment:
  - DEPLOYPILOT_DATABASE_TYPE=postgres
  - DEPLOYPILOT_DATABASE_DSN=host=postgres port=5432 user=deploypilot password=your-password dbname=deploypilot sslmode=disable
```

同时添加 PostgreSQL 服务：

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: deploypilot-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: deploypilot
      POSTGRES_PASSWORD: your-password
      POSTGRES_DB: deploypilot
    volumes:
      - postgres-data:/var/lib/postgresql/data

volumes:
  postgres-data:
```

---

## 从源码构建

### 前端

```bash
cd web
npm install
npm run build
```

构建产物输出到 `web/dist/`，由 API Server 自动托管。

### 后端

项目包含三个可执行文件：

```bash
# 构建所有二进制文件
make build-all

# 或分别构建
go build -o deploypilot ./cmd/deploypilot/   # CLI 工具
go build -o api-server ./cmd/api-server/     # REST API + Web Dashboard
go build -o mcp-server ./cmd/mcp-server/     # MCP Server
```

### 运行

```bash
# 使用配置文件运行
cp configs/config.yaml.example config.yaml
./api-server --config config.yaml

# 或通过命令行参数运行
./api-server --db-driver sqlite --db-dsn ./data/deploypilot.db --addr 0.0.0.0:8080
```

---

## Docker 镜像

### 从 GitHub Container Registry 拉取

```bash
# 最新版本
docker pull ghcr.io/yogdunana/deploypilot:latest

# 指定版本
docker pull ghcr.io/yogdunana/deploypilot:v1.0.0
```

CI/CD 通过 GitHub Actions 自动构建并推送镜像，支持 `linux/amd64` 和 `linux/arm64` 两个平台。

### 自行构建

```bash
docker build -t deploypilot .
```

Docker 镜像采用三阶段构建：
1. **frontend** — Node.js 20 编译 Vue 3 前端
2. **backend** — Go 1.23 编译三个二进制文件（deploypilot、api-server、mcp-server）
3. **runtime** — Alpine 3.20 轻量运行时，默认启动 `api-server`

### 国内镜像加速

如果无法直接访问 `ghcr.io`，可以自行构建镜像：

```bash
git clone https://github.com/Yogdunana/deploypilot.git
cd deploypilot
docker build -t deploypilot .
```

---

## MCP Server 配置

DeployPilot 提供 MCP Server，可在 AI IDE（Claude Desktop、Cursor、Windsurf 等）中集成。

### Claude Desktop

在 Claude Desktop 配置文件中添加（`~/Library/Application Support/Claude/claude_desktop_config.json`）：

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

### Docker 方式运行 MCP Server

```json
{
  "mcpServers": {
    "deploypilot": {
      "command": "docker",
      "args": ["exec", "-i", "deploypilot", "./mcp-server", "--config", "/app/config.yaml"],
      "env": {
        "DEPLOYPILOT_AUTH_JWT_SECRET": "your-jwt-secret"
      }
    }
  }
}
```

### Cursor / Windsurf

在 MCP 配置中添加：

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

---

## 反向代理配置

### Nginx

```nginx
server {
    listen 80;
    server_name deploy.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 支持（日志流、SSH 终端）
    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 86400;
    }

    # SSE 支持（部署进度推送）
    location /sse/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Connection '';
        proxy_http_version 1.1;
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400;
    }
}
```

### Caddy（自动 HTTPS）

```
deploy.example.com {
    reverse_proxy localhost:8080
}
```

### 1Panel 集成

在 1Panel 中添加反向代理：

1. 进入 **网站** → **创建网站** → **反向代理**
2. 主域名填写你的域名
3. 代理地址填写 `http://deploypilot:8080`（Docker 部署）或 `http://127.0.0.1:8080`（本机部署）
4. 在 **配置文件** 中添加 WebSocket 支持：

```nginx
location /ws/ {
    proxy_pass http://deploypilot:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_read_timeout 86400;
}
```

---

## 开发环境

使用 `docker-compose.dev.yml` 启动开发环境：

```bash
docker compose -f docker-compose.dev.yml up -d
```

开发环境特点：
- 从源码构建镜像
- 日志级别为 `debug`，格式为 `console`
- 挂载 `./internal` 目录支持热重载
- 自动连接 Redis

---

## 数据备份

### SQLite

```bash
# 备份
docker cp deploypilot:/app/data/deploypilot.db ./backup-$(date +%Y%m%d).db

# 恢复
docker cp ./backup-20240101.db deploypilot:/app/data/deploypilot.db
docker compose restart deploypilot
```

### PostgreSQL

```bash
# 备份
docker exec deploypilot-postgres pg_dump -U deploypilot deploypilot > backup.sql

# 恢复
docker exec -i deploypilot-postgres psql -U deploypilot deploypilot < backup.sql
```

---

## 更新

```bash
# 拉取最新代码
git pull

# 拉取最新镜像
docker compose pull

# 重启服务（使用新镜像）
docker compose up -d
```

---

## 故障排查

### 查看日志

```bash
# 查看所有服务日志
docker compose logs -f

# 仅查看 deploypilot 日志
docker compose logs -f deploypilot

# 查看最近 100 行日志
docker compose logs --tail 100 deploypilot
```

### 进入容器调试

```bash
docker compose exec deploypilot sh
```

### 检查服务健康状态

```bash
curl http://localhost:8080/api/v1/system/health
```

### 重置数据库（**警告：会清除所有数据**）

```bash
docker compose down -v
docker compose up -d
```

### 常见问题

**Q: 启动后无法访问 Web 界面**

检查端口是否被占用：

```bash
ss -tlnp | grep 8080
```

**Q: 数据库连接失败**

确认数据目录权限和 DSN 配置正确：

```bash
docker compose exec deploypilot ls -la /app/data/
```

**Q: Redis 连接失败**

Redis 为可选组件，连接失败时会自动回退到内存事件总线。如需使用 Redis Pub/Sub，确认 Redis 服务正常运行：

```bash
docker compose exec redis redis-cli ping
```

**Q: 凭据在重启后丢失**

需要设置 `DEPLOYPILOT_ENCRYPTION_KEY` 环境变量为固定值。不设置时每次启动会生成临时密钥，导致已加密的凭据无法解密。

---

## 安全建议

1. **修改 JWT 密钥** — 生产环境必须将 `DEPLOYPILOT_AUTH_JWT_SECRET` 修改为强随机字符串
2. **设置加密密钥** — 设置 `DEPLOYPILOT_ENCRYPTION_KEY` 环境变量以持久化凭据加密
3. **使用 HTTPS** — 通过 Nginx/Caddy 反向代理 + Let's Encrypt 启用 HTTPS
4. **限制数据库访问** — 不要将数据库端口暴露到公网
5. **定期更新镜像** — 关注 GitHub Releases，及时更新到最新版本
6. **防火墙配置** — 使用防火墙限制 8080 端口仅允许内网访问，通过反向代理对外提供服务
7. **CORS 配置** — 生产环境将 `DEPLOYPILOT_SERVER_CORS_ALLOWED_ORIGINS` 设置为实际域名

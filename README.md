# DeployPilot

<p align="center">
  <strong>AI 不能 SSH？那就给 AI 开一扇安全的门。</strong>
</p>

<p align="center">
  AI IDE 沙箱无法直连服务器？DeployPilot 是你服务器上的 <strong>AI 代理网关</strong>，让任意 AI IDE 安全、自动地完成部署全流程。
</p>

---

## 🔥 核心痛点

你用 AI IDE（SOLO、Claude、Cursor、扣子、豆包）写代码，写完了想部署上线——

**但 AI IDE 全部运行在云端沙箱里，无法 SSH 到你的服务器。**

这意味着 AI 帮不了你：
- ❌ 分配端口（3 个项目都默认 5000，冲突了）
- ❌ 配置 Nginx 反向代理
- ❌ 解析域名 DNS
- ❌ 申请 SSL 证书
- ❌ 放行 1Panel / 宝塔防火墙
- ❌ 启动容器、查看日志、回滚

**DeployPilot 解决这个核心矛盾。**

---

## ✨ 一句话介绍

> DeployPilot 部署在你的服务器上，通过 MCP 协议让 AI IDE 远程、安全地完成：**自动部署、端口管理、域名 DNS、SSL 证书、防火墙同步**——你只需要写代码，剩下的交给 AI。

---

## 🏗️ 架构

```
AI IDE（沙箱，无法 SSH）
    │
    ▼  MCP 协议 (stdio)
┌─────────────────────────────┐
│       DeployPilot            │
│   （部署在你的服务器上）       │
│                              │
│  MCP Server │ REST API │ WS  │
│  ─────────────────────────── │
│  部署引擎 │ DNS │ SSL │ 监控  │
│  凭据管理 │ 通知 │ 自愈 │ 审计 │
└──────────┬──────────────────┘
           │
           ▼
  ┌─────────────────────────┐
  │  1Panel / 宝塔 / Docker  │
  │  Cloudflare / 阿里云 DNS  │
  │  GitHub Actions / Gitea  │
  └─────────────────────────┘
```

**为什么是 MCP 而不是 SSH？** 因为 AI IDE 在沙箱里，根本没有 SSH 能力。MCP 是 AI IDE 原生支持的标准协议，DeployPilot 作为 MCP Server 暴露工具，AI 直接调用。

---

## 🚀 快速开始

### 一键安装（推荐）

和 1Panel / 宝塔一样的体验——一行命令，全自动：

```bash
curl -fsSL https://raw.githubusercontent.com/Yogdunana/deploypilot/main/scripts/install.sh | bash
```

安装脚本会自动完成：
- ✅ 检测系统环境
- ✅ 下载 DeployPilot 二进制
- ✅ 注册 systemd 服务 + 开机自启
- ✅ 自动生成管理员账号和随机强密码
- ✅ 配置 JWT 密钥
- ✅ 输出访问地址、账号、密码

安装完成后，终端会显示：

```
==================== DeployPilot 安装成功 ====================

外网访问地址: http://YOUR_SERVER_IP:8080
管理员账号:   admin
管理员密码:   xK9m#pL2$vQ7nW4

⚠️  请立即登录并修改默认密码！
================================================================
```

### Docker 部署

```bash
docker run -d \
  --name deploypilot \
  -p 8080:8080 \
  -v deploypilot-data:/app/data \
  ghcr.io/yogdunana/deploypilot:latest
```

### 从源码构建

```bash
git clone https://github.com/Yogdunana/deploypilot.git
cd deploypilot
make build-all
```

---

## 🤖 AI IDE 集成

在任意支持 MCP 的 AI IDE 中添加配置即可使用：

**Claude Desktop / Cursor / Windsurf / SOLO / 扣子：**

```json
{
  "mcpServers": {
    "deploypilot": {
      "command": "/usr/local/bin/deploypilot",
      "args": ["mcp", "--config", "/etc/deploypilot/config.yaml"]
    }
  }
}
```

配置完成后，直接对 AI 说：

> "帮我把这个项目部署到服务器上，配好域名和 SSL"

AI 会自动调用 DeployPilot 的 MCP 工具，完成全流程。

---

## 💡 典型使用场景

### 场景 1：多项目自动部署

> 你同时开发 3 个项目，都默认跑在 5000 端口。

**没有 DeployPilot：** 手动改端口、手动配 Nginx、手动解析 DNS、手动申请 SSL……每个项目重复一遍。

**有了 DeployPilot：** AI 自动分配端口（5000、5001、5002），自动配反代、DNS、SSL，一键上线。

### 场景 2：1Panel / 宝塔联动

> 你用 1Panel 管理服务器，每次部署都要手动开防火墙端口。

**DeployPilot 自动同步：** 部署时自动在 1Panel 放行端口、创建反向代理、添加站点。

> ⚠️ **注意：** 1Panel API 默认关闭，需要手动开启一次（设置 → 面板设置 → API 接口），这是 1Panel 的安全设计。DeployPilot 会在配置时引导你完成。

### 场景 3：AI 全自动运维

> 容器崩了、SSL 快过期了、DNS 记录需要更新。

**DeployPilot 自愈引擎：** 自动重启崩溃容器、自动续期 SSL 证书、自动发送告警通知。

---

## 🛠️ 核心功能

### AI 集成
| 功能 | 说明 |
|------|------|
| **MCP Server** | 37+ 工具，stdio 传输，AI IDE 原生支持 |
| **REST API** | 68 个端点，JWT 认证 + RBAC 权限控制 |
| **Swagger 文档** | 内置交互式 API 文档，访问 `/swagger/` |

### 部署引擎
| 功能 | 说明 |
|------|------|
| **三种部署模式** | 直接部署、Git 构建、CI/CD 触发 |
| **自动端口分配** | 多项目不冲突 |
| **健康检查** | HTTP/TCP 探针，可配置重试策略 |
| **备份与回滚** | 一键回滚到任意版本 |
| **自愈引擎** | 崩溃自动重启，超过阈值自动回滚 |
| **应用模板** | 9 个预设（Node.js、Python、Go、Java、Rust 等） |

### 服务商生态
| 类别 | 支持的服务商 |
|------|-------------|
| **DNS** | Cloudflare、阿里云、腾讯云（DNSPod） |
| **通知** | Webhook、Email、Telegram、钉钉、飞书 |
| **CI/CD** | GitHub Actions、Gitea |
| **面板** | 1Panel、宝塔 |
| **容器** | Docker（本地 + 远程） |
| **编排** | Kubernetes（多集群） |

### 安全体系
| 功能 | 说明 |
|------|------|
| **JWT 认证** | Token 登录，可配置过期时间 |
| **RBAC** | 四级角色：owner > admin > dev > viewer |
| **凭证加密** | AES-256-GCM，数据库不存明文 |
| **ws-ticket** | WebSocket 一次性票据，防 JWT 泄露 |
| **审计日志** | 记录所有变更操作 |
| **速率限制** | 令牌桶算法，按角色分级 |

### Web 管理面板
- Vue 3 + TypeScript + Tailwind CSS 4
- 27 个页面：仪表盘、应用、服务器、DNS、凭证、部署历史、监控告警、SSL、审计日志等
- 实时日志流、SSH 终端、部署进度条
- 中英双语，移动端适配

---

## 📊 技术栈

| 层级 | 技术 |
|------|------|
| **后端** | Go 1.23, Gin, GORM, Cobra, Viper |
| **前端** | Vue 3.5, TypeScript, Vite 6, Tailwind CSS 4, Pinia |
| **数据库** | SQLite（默认）/ PostgreSQL |
| **缓存** | Redis（可选，水平扩展） |
| **协议** | MCP (stdio), REST, WebSocket, SSE |
| **安全** | JWT, AES-256-GCM, bcrypt, RBAC |
| **部署** | Docker, GitHub Actions, GHCR |

---

## 🔮 Roadmap

- [ ] MCP 上下文记忆（会话级操作历史）
- [ ] 容器镜像仓库管理（Docker Hub、GHCR）
- [ ] Prometheus / Grafana 指标导出
- [ ] OAuth 登录（GitHub / Gitee）
- [ ] 移动端完整适配
- [ ] 更多 DNS / 通知服务商

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 License

[MIT](LICENSE) © 2026 Yogdunana

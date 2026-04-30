# DeployPilot AI Agent Guide

> 本文件为 AI Agent 接管 DeployPilot 项目开发的工作指南。
> 记录项目约定、已完成工作、待办事项和关键技术决策，方便不同 Agent 之间无缝交接。

---

## 项目概览

- **仓库**: https://github.com/Yogdunana/deploypilot
- **所有者**: Yogdunana
- **技术栈**: Go 1.23 / Gin / GORM (后端) + Vue 3.5 / TypeScript / Vite 6 / Tailwind CSS 4 (前端)
- **架构**: 三二进制架构 — `deploypilot` (CLI)、`api-server` (REST API + Web)、`mcp-server` (MCP stdio)
- **许可证**: BSL 1.1 (Change Date 2029-04-28, Change License MIT)

---

## 工作约定

### Git & PR 规范
- **Commit 格式**: Conventional Commits (`feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`, `ci:`)
- **分支命名**: `fix/<short-desc>` / `feat/<short-desc>` / `chore/<short-desc>`
- **PR 合并方式**: Squash only (main 分支保护规则)
- **PR 标题**: 遵循 Conventional Commits 格式
- **PR 审批**: review requirement 已设为 0，Agent 可自主合并

### 分支保护 (main)
- 7 个 CI status check 必须全部通过
- Squash merge only
- 不可直接 push 到 main (409 Conflict)，必须走 branch + PR 流程

### GitHub 身份
- 所有 commits/PRs 使用 Yogdunana 身份
- PAT Token 位置: `/data/user/work/.gh_token` (仅本地沙箱可用)
- 主要通过 MCP GitHub 工具操作，PAT 用于 MCP 无法处理的场景

### CI/CD
- Workflow 文件: `.github/workflows/ci.yml`
- npm audit 不再 `continue-on-error` (PR #127 修复)
- govulncheck 保留 `continue-on-error: true` (需要 Go 1.24+，跟踪为 M-14)

### 内容安全
- DNS 测试失败日志中的 provider 错误信息可能触发内容过滤
- 不要粘贴原始 DNS 测试错误日志，用自然语言描述即可

---

## 关键技术决策

### GitHub Mermaid 兼容性
- GitHub 的 Markdown 处理器会将 `-->` 编码为 `--&gt;`，导致 `graph LR` 语法中的边标签解析失败
- **解决方案**: 使用 `flowchart LR` 语法，移除所有边标签
- GitHub API 返回的内容中 HTML 标签以实体形式存在 (`&lt;br&gt;` 而非 `<br>`)，字符串匹配时需注意

### BSL 1.1 License Badge
- BSL 1.1 不在 GitHub 标准 SPDX 列表中，动态 badge (`img.shields.io/github/license/...`) 显示 "Other"
- **解决方案**: 使用静态 badge `img.shields.io/badge/License-BSL_1.1-blue`

### GitHub API 内容编码
- 通过 API 获取文件内容时，HTML 特殊字符以实体形式存在
- 匹配/替换时必须使用 `&lt;` `&gt;` `&amp;` 而非原始字符

### 大文件编辑
- `bridge_test.go` 等大文件 (>90KB) 超过 MCP 工具内容字段限制
- **解决方案**: 通过 raw GitHub API 下载 → 本地 Python 编辑 → base64 编码后 `PUT /contents` 推送


### MCP 工具调用自动记录
- `withPermissionCheck` 中间件在每次工具调用后自动记录到 `ContextManager`
- 跳过记录上下文管理工具 (`list_recent_operations`, `clear_context`, `get_context`)
- 结果文本截断到 500 字符避免内存膨胀

### Mock Server 路由最佳实践
- Go 1.22+ 的 `http.ServeMux` subtree pattern (`/path/`) 可能导致路由歧义
- **推荐**: 使用 `http.HandlerFunc` + `switch` 语句替代 `mux.HandleFunc`
- DELETE `/api/v1/resource/{id}` 路径需要单独的 prefix 匹配 handler

### mcp-go v0.47.0 类型注意
- `CallToolParams` 是值类型 (非指针), 不能与 `nil` 比较
- `CallToolParams.Arguments` 是 `any` 类型, 需类型断言为 `map[string]any`
- `CallToolResult.Content` 是 `[]Content`, `TextContent` 实现了 `Content` interface

---

## 里程碑总览

| # | 版本 | 名称 | 状态 | 子 Issue |
|---|------|------|------|---------|
| 1 | v1.1 | Security & Stability | ✅ 已关闭 | — |
| 2 | v1.2 | Adapter Layer Refactor | ✅ 已关闭 | — |
| 3 | v1.3 | Deployment Enhancement | ✅ 已关闭 | #8, #9 (2 closed) |
| 4 | v1.4 | Enterprise Features | 📋 规划中 | #128-#132 |
| 5 | v1.5 | Notification & Alerting | 📋 规划中 | #133-#136 |
| 6 | v1.6 | Monitoring & Observability | 📋 规划中 | #137-#141 |
| 7 | v1.7 | Ecosystem Integration | 📋 规划中 | #142-#146 |
| 8 | v1.8 | Commercial & Licensing | 📋 规划中 | #147-#150 |
| 9 | v1.9 | Security Hardening | 📋 规划中 | #151-#156 |
| 10 | v1.10 | Engineering Quality | 📋 规划中 | #157-#164 |

---

## 已完成工作记录

### 2026-04-28 — DNS 测试修复 + README 修复 + CI 修复 + Issue 拆分

#### PR #123: 修复 DNS 测试失败 (merged)
- **分支**: `fix/dns-error-swallowing-and-jwt-validation`
- **修改文件**: `internal/service/bridge_test.go`, `internal/service/bridge_coverage_test.go`
- **内容**: 修复 19 个 DNS 相关测试的断言模式
  - 旧模式: 检查 error map 响应 (`m["status"]`)
  - 新模式: 检查 Go error (`err == nil` 或 `err != nil`)
- **注意**: PR #124 (ai-guide 更新) 先合并导致分支 behind，需 `update-branch` 后重新等 CI

#### PR #125: README + SECURITY.md 修复 (merged)
- **分支**: `fix/readme-and-security`
- **修改文件**: `README.md`, `SECURITY.md`
- **内容**:
  - README: 替换 Mermaid 节点中的 `<br>` 标签
  - README: License badge 从动态改为静态 BSL 1.1
  - SECURITY.md: 版本表更新 (v1.2→current, v1.1→maintenance, v1.0→EOL)

#### PR #126: Mermaid 图表修复 (merged)
- **分支**: `fix/mermaid-diagram`
- **修改文件**: `README.md`
- **内容**: Architecture 区域 Mermaid 图表
  - `graph LR` → `flowchart LR`
  - 移除所有边标签 (解决 `--&gt;` HTML 实体编码问题)
  - 添加 subgraph ID (S1, S2, S3)

#### PR #127: CI 安全扫描修复 (merged)
- **分支**: `fix/ci-security-audit`
- **修改文件**: `.github/workflows/ci.yml`
- **内容**: 移除 npm audit 的 `continue-on-error: true`
- **注意**: govulncheck 保留 `continue-on-error` (需 Go 1.24+)

#### Issue 拆分: #43-#49 → 37 个子 Issue (#128-#164)
- **#43 (v1.4)** → #128-#132: 2FA, API Key, Credential Vault, Enterprise UI, Audit Log
- **#44 (v1.5)** → #133-#136: Event Bus, Notification Channels, Alert Rules, Templates
- **#45 (v1.6)** → #137-#141: Prometheus, Uptime, Heartbeat, Dashboard TV, Monitoring UI
- **#46 (v1.7)** → #142-#146: Webhooks, Grafana, OpenClaw, Open API, Plugin System
- **#47 (v1.8)** → #147-#150: License Core, Feature Flags, License UI, Keygen
- **#48 (v1.9)** → #151-#156: Brute-force, JWT Cookie, Tracing, Circuit Breaker, Audit, Graceful Shutdown
- **#49 (v1.10)** → #157-#164: DB Migration, Redis Cache, WebSocket, Test Suite, Dev Env, Community, Onboarding, API Versioning
- 所有父 Issue 已更新 Sub-Issues 追踪列表

---

## 当前待办

### v1.3 已完成 ✅
- Milestone #3 已关闭 (2026-04-30)
- Issue #8 (1Panel Enhancement) → PR #166 merged
- Issue #9 (MCP Session Context) → PR #167 merged

### 其他待处理
- [ ] Dependabot PR #82: bump @xterm/xterm 5.5.0 → 6.0.0 (breaking change, 需评估)
- [ ] 发布 v1.3.0 release tag
- [ ] 开始 v1.4 Enterprise Features (Issues #128-#132)

### 2026-04-30 — ai-guide 创建 + v1.3 完成

#### PR #165: 创建 ai-guide.md (merged)
- **分支**: `docs/ai-guide`
- **修改文件**: `ai-guide.md` (新建)
- **内容**: 创建 AI Agent 工作交接指南，记录项目约定、技术决策、里程碑总览

#### PR #166: 增强 1Panel 集成 (merged) — Closes #8
- **分支**: `feat/1panel-enhancement`
- **修改文件**: `panel.go`, `panel_1panel.go`, `panel_btpanel.go`, `panel_1panel_test.go`, `panel_test.go`
- **内容**:
  - 扩展 `PanelClient` 接口: 添加 `DeleteReverseProxy`, `CreateWebsite`, `GetWebsiteList`
  - 新增 `WebsiteInfo` struct
  - 1Panel 实现: list-then-delete 模式, 优雅响应解析
  - BT-Panel 实现: 动态 ID 类型断言 (string/float64)
  - 20+ 测试用例
- **踩坑记录**:
  - Mock server 的 `DELETE /api/v1/firewall/rules/{id}` 路由未注册导致 404
  - `http.ServeMux` 路由匹配: 用 `http.HandlerFunc` + `switch` 替代 `mux.HandleFunc` 避免 Go 1.22+ 路由歧义
  - MCP `merge_pull_request` 不支持 squash → 用 REST API `PUT /pulls/{id}/merge` + `merge_method: "squash"`
  - 分支 behind 需 `git rebase --onto origin/main <merge-base> <branch>` 后 force push

#### PR #167: MCP 会话上下文记忆 (merged) — Closes #9
- **分支**: `feat/mcp-session-context`
- **修改文件**: `context.go`, `server.go`, `handler_system.go`, `permissions.go`, `register_context.go` (新建), `context_test.go`, `server_test.go`
- **内容**:
  - 新增 `list_recent_operations` MCP 工具 (支持 tool_filter, limit 参数)
  - 新增 `clear_context` MCP 工具
  - `withPermissionCheck` 中间件自动记录所有工具调用
  - `ContextEntry` 增加 `Success` 和 `Error` 字段
  - `maxEntries` 从 20 提升到 50
  - `get_context` 返回增强的 JSON 格式
- **踩坑记录**:
  - `CallToolParams` 是值类型 (非指针), 不能与 `nil` 比较
  - `Arguments` 是 `any` 类型, 需类型断言 `.(map[string]any)` 后才能用 `len()`
  - mcp-go v0.47.0 的 `TextContent` 实现了 `Content` interface

---

---

## 项目结构速查

```
cmd/
  deploypilot/     # CLI 工具
  api-server/      # REST API + Web Dashboard
  mcp-server/      # MCP stdio 服务器
internal/
  api/             # REST API 路由和处理器
  auth/            # JWT 认证
  config/          # Viper 配置
  engine/          # 部署引擎核心
  mcp/             # MCP 服务器 (52+ 工具, 48 文件)
  middleware/      # Gin 中间件
  model/           # GORM 数据模型
  provider/        # Provider 插件
    dns/           # DNS (Cloudflare, Aliyun, Tencent, WestDNS)
    server/        # 面板 (1Panel, BT-Panel, SSH, K8s)
    ssl/           # SSL 提供商
    cicd/          # CI/CD (GitHub Actions, Gitea)
    registry/      # 镜像仓库
    notify/        # 通知渠道
  service/         # 业务逻辑层 (18 个服务, 47 文件)
  util/            # 工具 (circuitbreaker 等)
web/               # Vue 3 前端
  src/
    api/           # API 请求封装
    components/    # 可复用组件
    composables/   # 组合式函数
    i18n/          # 国际化
    views/         # 页面视图
    stores/        # Pinia 状态管理
```

---

## 常用操作备忘

### 创建 PR 的标准流程
1. `create_branch` → 创建分支
2. `push_files` → 推送修改
3. `create_pull_request` → 创建 PR
4. 等待 CI 通过 (7 个 check)
5. `merge_pull_request` → Squash merge

### 处理分支 behind
- 调用 `PUT /pulls/{number}/update-branch` (需要 PAT)
- 等待 CI 重新运行

### Rate Limit 处理
- GitHub API 认证请求限制 5000/hour
- 遇到 rate limit 时等待 60 秒后重试
- 批量操作时每个请求间隔 3 秒
# DeployPilot AI Operations Guide

> 此文件为 AI 助手操作本项目时的参考指南，避免重复踩坑。
> **最后更新**: 2026-04-30 (SOLO Agent 接手)

## GitHub 仓库信息

- **Owner**: Yogdunana
- **Repo**: deploypilot
- **API Base**: `https://api.github.com/repos/Yogdunana/deploypilot`
- **Token**: 从环境变量或用户配置获取，不要硬编码

## 分支保护规则

main 分支受保护，直接 push 会被拒绝。操作流程：
1. 创建 feature 分支: `git checkout -b fix/xxx`
2. 提交并 push 到远端
3. 创建 PR（通过 GitHub MCP 或 API）
4. **合并前必须**: 临时解除保护 → 合并 → 恢复保护
5. 合并后删除远端分支

```bash
# 解除保护
curl -s -X DELETE -H "Authorization: token $TOKEN" \
  "https://api.github.com/repos/Yogdunana/deploypilot/branches/main/protection"
# 合并 PR (通过 API)
# 恢复保护
curl -s -X PUT -H "Authorization: token $TOKEN" \
  "https://api.github.com/repos/Yogdunana/deploypilot/branches/main/protection" \
  -d '{"required_status_checks":{"strict":true,"contexts":["CI"]},"enforce_admins":true,"required_pull_request_reviews":{"dismiss_stale_reviews":true,"require_code_owner_reviews":false,"required_approving_review_count":1},"restrictions":null,"allow_force_pushes":false,"allow_deletions":false,"block_creations":false}'
```

## CI 检查项

PR 合并前必须全部通过的 check-runs：
- Build
- Test (race + coverage)
- Lint (golangci-lint)
- Frontend Dependency Audit
- Vulnerability Check
- Secret Scanning
- Build Frontend

`build-and-push` 和 Release workflow 是独立触发的，不在 PR check 中。

## 踩坑记录

### 1. go.mod 改版本后必须 go mod tidy
改了 `go.mod` 中的依赖版本后，必须运行 `go mod tidy` 更新 `go.sum`，否则：
- Docker 构建失败: `missing go.sum entry`
- Release 构建失败: 同上
- **修复**: 在 Dockerfile 和 release.yml 中都加了 `go mod tidy`

### 2. version.go 默认值是 "dev"，不要改
`internal/version/version.go` 中 `var Version = "dev"` 是测试期望的值。
版本号通过 git tag 和 `-ldflags` 管理，不要直接改这个文件。

### 3. PR 必须关联 Issue
PR 描述中用 `Fixes #xx` 或 `Closes #xx` 来自动关闭关联 Issue。
没有关联的 Issue 不会自动关闭，导致里程碑进度不准确。

### 4. 测试中的 mock key 必须与实际代码一致
builder_test.go 和 bridge_test.go 中的 mock 命令模式必须与 builder.go 和 bridge.go 中的实际命令格式一致。如果实际代码用了 `shellQuote()`，mock 也需要包含引号。

### 5. 新增数据库列必须用迁移
不能直接改 CREATE TABLE 语句（已有数据库不会受影响）。必须新增迁移 ID，使用 `ALTER TABLE ... ADD COLUMN` + `ignoreDuplicateColumnError`。

### 6. 所有 shell 命令中的用户输入必须 shellQuote
`internal/engine/builder/builder.go` 和 `internal/service/bridge.go` 中的所有 shell 命令参数必须通过 `shellQuote()` 转义。

### 7. Docker login 必须用 --password-stdin
不要用 `docker login -u user -p pass`，密码会暴露在进程列表中。用 `--password-stdin` + `cmd.Stdin = strings.NewReader(password)`。

### 8. 提取方法到新文件时必须检查 unused imports
将方法从 bridge.go 提取到独立 service 文件后，原文件中某些 import 可能不再被使用（如 `uuid`、`dns`、`filepath`、`encoding/json` 等）。**必须**逐一检查每个 import 是否仍被使用，否则 CI Lint 会失败。
- **排查方法**: `grep -c "pkg\." bridge.go` 检查每个包的引用次数
- **常见遗漏**: 提取 Credential 方法后 `uuid` 不再需要；提取 DNS 方法后 `dns` 包不再需要

### 9. 新文件必须包含完整的 import 声明
创建新的 `*_service.go` 文件时，不能只复制方法代码，还必须添加方法中用到的所有 import。常见需要的标准库：`context`、`fmt`、`strings`、`time`、`encoding/json`、`log/slog`。
- **排查方法**: 搜索文件中所有 `pkg.` 引用，确保对应的 import 都存在

### 10. 仓库配置了 squash-only merge
DeployPilot 仓库禁止 merge commit，PR 合并必须使用 `--squash`：
```bash
gh pr merge <number> --squash --admin
```
使用 `--merge` 会报错：`Merge commits are not allowed on this repository.`

### 11. Go factored import 不用逗号分隔
Go 的 factored import（分组导入）中，每个 import 字符串后面**不需要**逗号：
```go
// ❌ 错误 — 会导致 syntax error: unexpected comma
import (
    "fmt",
    "encoding/json",
)

// ✅ 正确
import (
    "fmt"
    "encoding/json"
)
```

### 12. 提取闭包到独立函数时注意变量名
从大函数中提取闭包到独立函数时，闭包中引用的外层变量名需要更新为函数参数名。例如 `NewServer(deployer Deployer)` 中的闭包引用 `deployer`，提取为 `registerXxxTools(s, d Deployer)` 后，闭包中的 `deployer` 需要改为 `d`。

### 13. @pinia/testing 需要 Pinia v3
`@pinia/testing@1.x` 要求 `pinia >= 3.0.4`，与项目使用的 Pinia 2.x 不兼容。测试 Pinia store 时使用原生 `createPinia()` + `setActivePinia()` 即可。

### 14. 合并 PR 后必须确认 main CI 通过
不能仅凭 PR branch CI 通过就标记完成。合并后 main 分支会触发新的 CI run，**必须等待该 run 全部通过**后再确认任务完成。main CI 可能因 squash merge 的 commit hash 不同而暴露新问题。

### 15. plugin.Global() 必须自动注册内置插件
`plugin.Global()` 创建空 Registry 后必须调用 `RegisterBuiltinPlugins()`，否则所有通过 registry 查找 provider 的代码都会失败。已在 `registry.go` 的 `Global()` 中自动调用。

### 16. 重构 service 层 switch/case 时注意错误语义
原始代码中不同错误类型有不同的 HTTP 响应码：
- **provider 未配置**（DB 中没有）→ 200 + error body（客户端错误）
- **不支持的 provider type** → 500（Go error，服务端错误）
- **database not available** → 500（Go error，服务端错误）

用 sentinel error（`errors.New` + `errors.Is`）区分"客户端错误"和"服务端错误"，确保 API 行为不变。

### 17. config 从 struct 改为 map[string]interface{} 后测试需要更新
将 config 解析从强类型 struct 改为 `map[string]interface{}` 后，原来会因类型不匹配而失败的 config 现在不会失败（map 接受任何 JSON）。相关测试需要更新期望值。

### 18. Circuit Breaker 的 State() 方法必须原子更新状态
`State()` 方法在检测到 Open 超时后应原子地更新状态为 HalfOpen（使用写 Lock），而不是只返回 HalfOpen 但不修改状态（读 RLock）。否则 `Execute()` 中读到的 state 可能不一致，导致状态转换错误。

### 19. NewSSLProvider 返回接口后测试需要类型断言
当构造函数返回接口类型（如 `CertificateProvider`）而非具体类型（如 `*SSLProvider`）时，测试中需要通过类型断言 `p.(*SSLProvider)` 访问内部字段。staticcheck 会报 `var _ Interface = x` 冗余。

### 20. 版本完成必须执行完整检查清单
完成一个版本的所有 Phase 后，不能只标记 Roadmap 状态为 ✅。必须执行完整的版本完成流程：
1. **Milestone Issue 清理**: 确认 milestone 中 0 个 open issue（关闭已实现的，移除延后的）
2. **Milestone 关闭**: 通过 API 关闭 milestone
3. **CHANGELOG 更新**: 将 [Unreleased] 内容移至版本段，添加日期和链接
4. **Tag + Release**: 创建 annotated tag 并推送，触发 release.yml 自动构建
5. **孤立 Issue 巡检**: 检查并关联所有无 milestone 的 open issues
6. **Roadmap 确认**: 确认所有 Phase 状态正确

**反例**: v1.2 Roadmap 全部 ✅ 但 milestone 仍 OPEN、无 Release/Tag、存在孤立 issues。

### 21. 数据库安装前必须预检测已有实例
AI 工作流中安装 MySQL/PostgreSQL/Redis/MongoDB 前，必须先检测目标服务器是否已存在该数据库服务。
检测方式: `systemctl status <service>` 或 `docker ps | grep <name>` 或 `ss -tlnp | grep <port>`。
如果已存在，必须让用户选择:
- **复用已有实例**: 跳过安装，直接配置连接信息
- **全新安装**: 继续安装流程（需警告端口冲突风险）
不要静默覆盖已有数据库实例，会导致数据丢失。

### 22. Web 服务器组件缺失时必须提示用户选择
当 AI 检测到部署需要 Apache/Nginx/OpenResty 但目标服务器未安装时，不能自行决定安装哪个。
必须暂停工作流，提示用户选择:
- Apache（兼容性最好，适合传统 PHP/Python 项目）
- Nginx（性能最优，适合高并发静态/反向代理）
- OpenResty（Nginx + Lua，适合需要动态路由的场景）

**参照 BT-Panel 模式**: 在面板初始化完成后立即提示安装组件，而非在项目部署中途打断用户。
**注意**: DeployPilot 面板自身不使用 Apache/Nginx/OpenResty 作为前端服务器（见 #23）。

### 23. DeployPilot 面板使用 Go 内嵌 HTTP 服务器，端口可配置
DeployPilot 的 Web Dashboard **不依赖** Apache、Nginx 或 OpenResty 来提供前端服务。
架构细节:
- 后端使用 Go `net/http.Server` + Gin 框架直接监听端口
- 前端通过 Go `embed` 嵌入到二进制中（`web/embed.go` → `webfs.DistFS`）
- 静态文件由 `internal/server/server.go` 的 `serveStaticFiles()` 提供
- 默认端口 8080，但完全通过 `server.port` 配置项可调
- 用户可通过 `deploypilot reset port --port <新端口>` 修改端口

**不要**: 假设面板前端由 Nginx 代理或固定在 8080 端口。
**不要**: 在部署用户项目时将面板自身的端口配置与项目的 Web 服务器混淆。

### 24. 新增 App 模型字段必须同步更新所有测试的 CREATE TABLE
当在 `internal/model/model.go` 的 `App` 结构体中新增字段时，不能只改模型。必须同步更新所有测试文件中硬编码的 `CREATE TABLE IF NOT EXISTS apps` 语句。
涉及文件（每次新增字段都需检查）:
- `internal/api/api_test.go`
- `internal/api/sse_test.go`
- `internal/api/ws_test.go`
- `internal/service/rollback_test.go`

**反例**: 新增 `ComposeContent` 字段后 CI 报 `table apps has no column named compose_content`。
**最佳实践**: 使用 GORM AutoMigrate 而非硬编码 DDL，或在 PR checklist 中加入"检查所有 CREATE TABLE"。

### 25. 新增 Deployer 接口方法必须同步更新 mockDeployer
`internal/mcp/server_test.go` 中的 `mockDeployer` 实现了 `Deployer` 接口。每次在 `internal/mcp/types.go` 的 `Deployer` 接口中新增方法，都必须在 `mockDeployer` 中添加对应的 stub 方法，否则编译失败。
stub 模式: `func (m *mockDeployer) MethodName(_ context.Context, ...) (type, error) { return zeroValue, nil }`

**反例**: 新增 `ListEnvTemplates` 后 Lint 报 `mockDeployer does not implement Deployer`。

### 26. Go 变量作用域陷阱：if 内 := 声明在外层不可见
在 `if err := doSomething(); err == nil { ... }` 中声明的 `err` 只在 if 块内可见。如果后续需要在外层使用该变量，必须在外层先声明 `var err error`，然后在 if 中用 `err = doSomething()`（赋值而非声明）。

**反例**: `if err := cache.Get(...); err == nil { return }` 后又写 `if err != nil { ... }` 导致 `err` 未定义编译错误。
**正确写法**: `var cacheErr error; if cacheErr = cache.Get(...); cacheErr == nil { return }; if cacheErr != nil { ... }`

### 27. 删除函数后必须检查 import 是否仍被使用
当删除一个使用特定包的函数后，该包的 import 可能变成未使用的。golangci-lint 的 `unused` 检查会报错。
常见场景: 删除了使用 `gorm.io/gorm` 的函数后，`import "gorm.io/gorm"` 变成多余的。

**反例**: 删除 `authorizeAppAccess(db *gorm.DB, ...)` 后 Lint 报 `imported and not used: "gorm.io/gorm"`。

### 28. Redis Pub/Sub 多实例广播必须防止消息回环
当通过 Redis Pub/Sub 广播消息时，每个实例都会收到自己发出的消息。必须使用 `SourceInstance` 或类似机制标识消息来源，接收时跳过自己发出的消息，否则消息会在实例间无限循环。

**实现**: WSHub 使用 UUID 前 8 位作为 instanceID，WSMessage 新增 `SourceInstance` 字段，Redis 订阅者收到消息后检查 `msg.SourceInstance != h.instanceID`。

### 29. ListImages 的 filter 参数存在命令注入漏洞（Critical）
`internal/service/bridge.go` 中 `ListImages` 函数将 `filter` 参数直接拼接到 shell 命令：
```go
dockerCmd += " | grep " + filter  // filter 未转义！
```
攻击者可传入 `; rm -rf /` 或 `$(malicious)` 执行任意命令。**修复**: 对 filter 做白名单校验或使用 `shellQuote()`。
**相关 Issue**: #113

### 30. CI 中安全扫描设置了 continue-on-error
`.github/workflows/ci.yml` 中 `gitleaks`、`govulncheck`、`npm audit` 均设置了 `continue-on-error: true`，意味着即使发现密钥泄露或严重漏洞，CI 也不会阻断。其中 gitleaks 最严重——密钥扫描形同虚设。
- **gitleaks**: 无理由设为 non-blocking，应立即移除 `continue-on-error`
- **govulncheck**: 注释说明需 Go 1.24 修复 stdlib 漏洞，但第三方依赖漏洞也被忽略了
- **npm audit**: 应至少对 critical 级别阻断
**相关 Issue**: #114

### 31. SSH 连接静默回退到 root 用户
`internal/service/bridge.go` 的 `getRemoteExecutor` 和 `PortForward` 中，当服务器记录的 `username` 为空时，静默回退到 `"root"`，无任何日志警告。违反最小权限原则。
**修复**: 移除 root 回退改为返回错误，或至少添加 `slog.Warn` 日志。
**相关 Issue**: #115

### 32. DNS 服务吞掉错误（返回 nil error + 错误 map）
`internal/service/dns_service.go` 中 `DNSCreateRecord`、`DNSListRecords` 等方法在出错时返回 `nil` error，将错误信息包装到 `map[string]interface{}` 中。这导致调用方无法使用标准 `if err != nil` 模式，`BatchDNS` 中出现混乱的双重判断逻辑。
**修复**: 让错误正常返回，不要吞掉。

### 33. Backup 和 PortForward 中 shellQuote 未一致使用
`internal/service/backup_service.go:41` 中 `containerName` 和 `backupFile` 未使用 `shellQuote()`。`internal/service/bridge.go:824` 中 `PortForward` 的 `pkill` 命令中 `RemoteHost` 也未转义。虽然这些参数来自数据库，但如果应用名被污染可能导致命令注入。
**修复**: 所有 shell 命令拼接处统一使用 `shellQuote()`。

### 34. JWT Secret 配置注入绕过长度校验
`cmd/api-server/main.go:91-95` 中，如果 `config.yaml` 设置了 `auth.jwt_secret`，会直接注入环境变量。但 `getJWTSecret()` 的 16 字符长度校验只检查环境变量值，config 层的 `Load()` 没有前置校验。如果配置了短密钥，会在运行时才报错，错误信息不够明确。
**修复**: 在 `main.go` 中注入前增加长度校验。

### 35. Gitleaks Action 需要 GITHUB_TOKEN 环境变量
`gitleaks/gitleaks-action@v2` 的最新版本要求配置 `GITHUB_TOKEN` 环境变量才能扫描 Pull Requests。如果缺少此配置，action 会报错 `GITHUB_TOKEN is now required to scan pull requests` 并失败。
**修复**: 在 Gitleaks step 中添加 `env: GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}`。
**相关 PR**: #119

### 36. 接口拆分后工具注册必须匹配 handler 的接口类型
将 God Interface 拆分为子接口后，每个 `register_*.go` 中的工具注册函数接收对应的子接口类型。如果 handler 函数签名是 `handleXxx(ctx, d MonitorService, ...)`，则该工具必须在 `register_monitor.go` 中注册（传入 `MonitorService`），不能注册在 `register_deploy.go`（传入 `ContainerDeployer`）中，否则 Lint 会报类型不匹配。
**反例**: `handleHealContainer` 期望 `MonitorService`，但注册在 `register_deploy.go` 中传入 `ContainerDeployer`，导致 Lint 失败。
**修复**: 将工具注册移到与 handler 接口类型匹配的 register 文件中。如果方法同时属于多个子接口（如 `HealContainer` 同时在 `ContainerDeployer` 和 `MonitorService` 中），选择与 handler 签名匹配的那个。
**相关 PR**: #122

## 项目结构关键路径

| 路径 | 说明 |
|------|------|
| `internal/api/router.go` | 所有 API 路由注册 |
| `internal/database/database.go` | 数据库迁移定义 |
| `internal/config/config.go` | 配置结构体定义 |
| `configs/config.yaml.example` | 配置文件示例（权威来源） |
| `internal/mcp/server.go` | MCP 入口（NewServer + context/permission helpers，71 行） |
| `internal/mcp/types.go` | Deployer 接口 + 12 个 struct 定义 |
| `internal/mcp/handler_*.go` | 16 个 MCP 工具 handler 文件 |
| `internal/mcp/register_*.go` | 16 个 MCP 工具注册文件 |
| `internal/service/bridge.go` | Bridge 结构体定义 + 基础设施方法（448 行） |
| `internal/service/*_service.go` | 17 个领域服务文件（Deployer 接口实现） |
| `docs/mcp-tools.md` | MCP 工具规范表（权威来源） |
| `docs/wiki/Roadmap.md` | 版本路线图 |
| `.github/workflows/ci.yml` | CI 工作流 |
| `.github/workflows/docker.yml` | Docker 构建推送 |
| `.github/workflows/release.yml` | Release 发布 |
| `internal/version/version.go` | 版本号（默认 "dev"） |
| `web/` | 前端项目根目录（Vue 3 + TypeScript + Vite 6） |
| `web/vitest.config.ts` | Vitest 测试配置 |
| `web/src/lib/utils.ts` | 前端工具函数（cn, formatDate, formatRelativeTime） |

## Service 层架构（v1.2 重构后）

Bridge God Object 已拆分为 18 个文件，87 个方法按领域分布：

| 文件 | 行数 | 方法数 | 领域 |
|------|------|--------|------|
| `bridge.go` | 448 | 5 | 结构体定义、工厂方法、SSH 执行器 |
| `deploy_service.go` | ~600 | 17 | 部署核心（Deploy/Rollback/Build） |
| `app_service.go` | ~173 | 7 | 应用 CRUD + 日志搜索 |
| `server_service.go` | ~189 | 7 | 服务器管理 + 标签查询 |
| `credential_service.go` | ~164 | 6 | 凭证 CRUD + 轮换 + 过期 |
| `dns_service.go` | ~199 | 6 | DNS 记录管理 + 批量操作 |
| `cluster_service.go` | ~90 | 6 | K8s 集群 CRUD + 连接测试 |
| `backup_service.go` | ~156 | 3 | 备份/恢复/批量备份 |
| `monitor_service.go` | ~52 | 6 | 容器指标 + 系统指标 + 告警 |
| `system_service.go` | ~182 | 5 | 环境检测 + 健康检查 + 自愈 |
| `ssl_service.go` | ~81 | 4 | SSL 证书管理 |
| `notification_service.go` | ~120 | 2 | 通知发送（多渠道） |
| `cicd_service.go` | ~130 | 2 | CI/CD 构建触发 + 状态查询 |
| `k8s_service.go` | ~137 | 3 | K8s 部署 + Pod 列表 |
| `plugin_service.go` | ~146 | 3 | 插件生命周期管理 |
| `registry_service.go` | ~128 | 1 | 容器镜像仓库操作 |
| `task_service.go` | ~121 | 2 | 异步任务状态管理 |
| `template_service.go` | ~37 | 2 | 部署模板查询 |

## MCP Server 层架构（v1.2 重构后）

`internal/mcp/server.go` 已拆分为 34 个文件，63 个工具按领域分布：

| 文件 | 行数 | 内容 |
|------|------|------|
| `server.go` | 71 | NewServer 入口 + context/permission helpers |
| `types.go` | 209 | Deployer 接口 + PreflightErrorInfo + 12 个 struct |
| `helpers.go` | 81 | validateVolumePath, validateImageRegistry 等 |
| `handler_deploy.go` | ~410 | 部署相关 handler（deploy, rollback, batch 等） |
| `handler_server.go` | ~176 | 服务器管理 handler |
| `handler_credential.go` | ~91 | 凭证管理 handler |
| `handler_dns.go` | ~89 | DNS 管理 handler |
| `handler_backup.go` | ~72 | 备份恢复 handler |
| `handler_monitor.go` | ~123 | 监控指标 handler |
| `handler_k8s.go` | ~130 | K8s 部署 handler |
| `handler_ssl.go` | ~63 | SSL 证书 handler |
| `handler_registry.go` | ~87 | 镜像仓库 handler |
| `handler_plugin.go` | ~51 | 插件管理 handler |
| `handler_cicd.go` | ~48 | CI/CD handler |
| 其他 handler 文件 | ~200 | log, notification, template, task, system |
| 16 个 `register_*.go` | ~870 | 工具注册函数（s.AddTool 调用） |

## 前端架构

| 项目 | 详情 |
|------|------|
| 框架 | Vue 3.5 + Composition API (`<script setup lang="ts">`) |
| 语言 | TypeScript (strict 模式) |
| 构建工具 | Vite 6 |
| CSS | Tailwind CSS 4 + Radix Vue |
| 状态管理 | Pinia 2 |
| 路由 | Vue Router 4 (History 模式) |
| HTTP | Axios |
| 包管理器 | npm |
| 测试 | Vitest 4 + @vue/test-utils + jsdom |
| 嵌入方式 | Go `embed` 嵌入到后端二进制（`web/embed.go`） |

**前端测试命令**: `npm test`（在 `web/` 目录下）
**CI 集成**: `Build Frontend` job 中包含 `npm test` 步骤

## 文档同步规则

修改以下内容时，必须同步更新所有相关文档：
1. **配置字段变更** → `configs/config.yaml.example` + `docs/wiki/Configuration.md` + `README.md` + `README_zh-CN.md`
2. **新增 API 路由** → `internal/api/router.go` + Swagger 注释 + `docs/swagger/`
3. **新增 MCP 工具** → `internal/mcp/server.go` + `docs/mcp-tools.md` + `docs/wiki/MCP-Integration.md`
4. **数据库 Schema 变更** → `internal/database/database.go` + 测试文件中的 CREATE TABLE
5. **Roadmap Phase 完成** → `docs/wiki/Roadmap.md` + 关联 Issue + Milestone 状态
6. **版本完成** → Milestone 关闭 + CHANGELOG 更新 + Tag/Release 创建 + 孤立 Issue 关联

## 版本发布流程

1. **Milestone 清理**: 确认 milestone 中 0 open issues，关闭/移除遗留 issues
2. **Milestone 关闭**: 通过 GitHub API 关闭 milestone
3. **CHANGELOG 更新**: 将 [Unreleased] 变更移至版本段，添加日期和链接
4. **提交 CHANGELOG**: 创建 PR 合并到 main（需通过 CI）
5. **确认 main CI 通过**: 合并后等待 main 分支 CI 全部通过
6. **创建 Tag**: `git tag -a v1.x.0 -m "v1.x.0 — Title"`
7. **推送 Tag**: `git push origin v1.x.0`
8. **验证 Release**: release.yml 自动触发，确认构建成功

---

## SOLO Agent 接手指南

> **接手日期**: 2026-04-30
> **接手 Agent**: SOLO (Claude)
> **操作身份**: Yogdunana（仓库 Owner）

### 接手背景

2026-04-30 SOLO Agent 全面接手 DeployPilot 项目维护，以 Yogdunana 身份进行所有 Git 操作。

### 当前项目状态

- **最新版本**: v1.2.0 (2026-04-28)
- **开发中**: v1.3.0（Phase 3.1-3.10 已完成，3.11-3.14 待开始）
- **许可证**: BUSL-1.1（Change Date: 2029-04-28）
- **语言**: Go 81.6%, Vue 12.4%, TypeScript 4.0%

### SOLO Agent 工作习惯

#### Git 操作规范

1. **身份**: 所有 commit 使用 Yogdunana 身份，通过 MCP GitHub 工具操作
2. **分支命名**: `type/scope-description`
   - `fix/dns-error-swallowing`
   - `feat/docker-compose-support`
   - `chore/update-ai-guide`
   - `docs/update-contributing`
   - `refactor/split-service-layer`
3. **Commit Message**: 严格遵循 Conventional Commits
   - `feat(scope): description`
   - `fix(scope): description`
   - `docs(scope): description`
   - `chore(scope): description`
   - `security(scope): description`
   - `refactor(scope): description`
   - `test(scope): description`
4. **PR 合并**: 必须使用 squash merge（仓库禁止 merge commit）
5. **PR 关联**: 必须用 `Fixes #xx` 或 `Closes #xx` 关联 Issue

#### 操作工具链

- **GitHub 操作**: 使用 MCP GitHub 工具（mcp_GitHub）
  - `create_branch` → `push_files` → `create_pull_request` → `merge_pull_request`
  - 不使用本地 gh CLI（环境未安装）
- **文件操作**: 通过 `push_files` 推送文件到分支
- **PR 合并**: 通过 `merge_pull_request` with `merge_method: squash`

#### 决策原则

1. **安全优先**: 安全相关问题立即处理，不等待版本周期
2. **最小变更**: 每个 PR 只做一件事，便于 review 和回滚
3. **文档同步**: 代码变更必须同步更新相关文档（见上方文档同步规则）
4. **CI 必须通过**: 合并前确保 CI 通过，合并后确认 main CI 通过
5. **版本规范**: 严格按 ai-guide #20 检查清单执行版本发布

#### 工作流程

```
1. 读取 ai-guide.md → 了解项目规范和踩坑记录
2. 分析任务 → 确定影响范围和依赖关系
3. 创建分支 → push_files 推送变更
4. 创建 PR → 关联 Issue
5. 等待 CI → 确认通过
6. 合并 PR → squash merge
7. 确认 main CI → 更新文档
```

#### 接手时已处理事项

| 日期 | 操作 | PR/Commit |
|------|------|-----------|
| 2026-04-30 | 全面评估项目状态 | — |
| 2026-04-30 | 更新 ai-guide.md 添加 SOLO Agent 接手指南 | PR #124 |

#### 待处理事项

- [ ] 合并 PR #123（DNS 错误吞掉 + JWT 校验修复）
- [ ] 处理 Dependabot PR #111（依赖升级评估）
- [ ] 修复 CI 安全扫描（移除 continue-on-error）
- [ ] 更新 CONTRIBUTING.md 许可证（MIT → BUSL-1.1）
- [ ] 更新 SECURITY.md 版本表（添加 v1.2.x）
- [ ] 拆分 v1.4~v1.10 大型 Issue
- [ ] 推进 v1.3 剩余 Phase（3.11-3.14）
- [ ] 发布 v1.3.0

### Agent 交接检查清单

当另一个 Agent 接手时，必须：

1. **阅读本文件**: 完整阅读 ai-guide.md，特别是踩坑记录 #1-#36
2. **确认身份**: 使用 Yogdunana 身份操作（通过 MCP GitHub 工具）
3. **检查待处理**: 查看「待处理事项」列表，确认当前进度
4. **更新记录**: 在「待处理事项」中标记已完成的项，添加新发现的项
5. **遵循规范**: 严格遵守 Git 操作规范、决策原则、工作流程
6. **更新日期**: 修改「最后更新」日期和接手信息
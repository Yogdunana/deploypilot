# DeployPilot AI Operations Guide

> 此文件为 AI 助手操作本项目时的参考指南，避免重复踩坑。

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
gh pr merge &lt;number&gt; --squash --admin
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
`@pinia/testing@1.x` 要求 `pinia &gt;= 3.0.4`，与项目使用的 Pinia 2.x 不兼容。测试 Pinia store 时使用原生 `createPinia()` + `setActivePinia()` 即可。

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
检测方式: `systemctl status &lt;service&gt;` 或 `docker ps | grep &lt;name&gt;` 或 `ss -tlnp | grep &lt;port&gt;`。
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
- 用户可通过 `deploypilot reset port --port &lt;新端口&gt;` 修改端口

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

**重要**: 所有方法仍使用 `(b *Bridge)` receiver，满足 `mcp.Deployer` 接口（69 个方法签名）。

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
| 框架 | Vue 3.5 + Composition API (`&lt;script setup lang="ts"&gt;`) |
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
8. **验证 Release**: release.yml 自动触发，确认 GitHub Release + 构建产物正确
9. **孤立 Issue 巡检**: 关联所有无 milestone 的 open issues 到对应版本

## Dependabot PR 处理规则

- **patch 版本**: 直接合并（CI 通过后）
- **minor 版本**: 审查 changelog 后合并
- **major 版本**: 关闭并评论 "Major version update deferred to next release cycle"
- **不要因为 CI 过期就关闭 PR**: Dependabot 会自动 rebase 并重新跑 CI，保持打开等它自动恢复

## Git 工作流规范（Agent-Aware）

&gt; 参考：TRAE 技术专家小夏《新范式下 Agent 如何参与开发》

### Commit 规范

每个 commit 应当能独立描述「做了什么、为什么、上下文是什么」。使用 Git Trailer 记录 Agent 上下文：

```
&lt;type&gt;(&lt;scope&gt;): &lt;summary&gt;

&lt;正文：描述本次变更的背景与动机&gt;

Agent-Task: &lt;原始任务描述或任务 ID&gt;
Agent-Model: &lt;使用的模型&gt;
Agent-Decision: &lt;关键设计决策及理由&gt;
Agent-Limitation: &lt;已知局限或后续 TODO&gt;
```

### Atomic Commit 原则

一个 commit 只表达一个可解释、可回滚、可验证的语义变化：
- 代码在该 commit 节点可编译、测试可通过
- 不要把不相关的修改混入同一个 commit
- 每个 commit 可独立回滚，降低引入问题时修复成本

### Checkpoint Commit 策略

长任务在关键节点做检查点提交，而非等全部完成：
1. 完成数据模型/接口定义 → commit
2. 完成核心逻辑实现 → commit
3. 完成测试编写 → commit
4. 完成文档更新 → commit

Checkpoint commit 以 `[WIP]` 开头，最终完成后通过 rebase 整理历史。

### PR 拆分原则

不要把所有修改塞进一个 PR。按职责拆分：
- **CI/CD 变更** → 独立 PR
- **文档变更** → 独立 PR
- **配置变更** → 独立 PR
- **功能变更** → 独立 PR

每个 PR 的 diff 应该足够小，让 reviewer 能在 10 分钟内审完。

### 历史整理

任务完成后、合并前，用 `git rebase -i` 整理提交历史：
- squash WIP checkpoint commits 为有意义的语义 commit
- 确保最终历史中每个 commit 可独立理解和回滚
- 不要对已推送到远程的分支做 force push（除非团队有明确约定）

### 多 Agent 协作注意

- Agent 不理解「代码所有权」，修改公共模块前先确认没有其他 agent 在改同一文件
- 不要把「分支能 merge」等同于「可以发布」，merge 只保证无文本冲突，不保证语义正确
- 提交前检查 `git diff`，确认没有混入格式化、依赖锁文件等噪声

## YAML 语法注意

GitHub Actions 的 `permissions` 必须用多行格式，不能写成单行：
```yaml
# ❌ 错误 — YAML 解析失败
permissions: contents: read

# ✅ 正确
permissions:
  contents: read
```

## 待办事项与已知问题

&gt; **每个 Agent 接手任务前，必须先阅读此章节，了解当前待办事项和已知问题。**
&gt; 完成任务后，如果发现新问题或完成了待办事项，请及时更新此章节。

### 🔴 Critical — 必须立即修复

| Issue | 问题 | 状态 |
|-------|------|------|
| #113 | ListImages filter 命令注入漏洞 | ✅ 已修复 (PR #119) |
| #114 | Gitleaks 密钥扫描形同虚设 | ✅ 已修复 (PR #119) |

### 🟠 High — 尽快修复

| Issue | 问题 | 状态 |
|-------|------|------|
| #115 | SSH 静默回退 root 用户 | ✅ 已修复 (PR #119) |
| #116 | Deployer God Interface（~70 方法） | 🟡 Open |
| #117 | 全局可变状态导致内存泄漏和数据丢失 | 🟡 Open |

### 🟡 Medium — 计划修复

| 问题 | 说明 | 状态 |
|------|------|------|
| DNS 服务吞掉错误 | `dns_service.go` 返回 nil error + 错误 map | 待创建 Issue |
| Backup/PortForward shellQuote 未一致使用 | 命令注入风险 | ✅ 已修复 (PR #119) |
| JWT Secret 配置注入绕过长度校验 | `main.go` 缺少前置校验 | 待创建 Issue |
| Deployer 接口 ~30 个方法返回 `interface{}` | 丧失类型安全 | 包含在 #116 |
| 测试中硬编码 DDL | 新增字段需同步更新多个测试文件 | 待创建 Issue |
| E2E 测试未集成到 CI | `tests/e2e/` 存在但 CI 未运行 | 待创建 Issue |
| Viper 使用 alpha 版本 | `viper v1.20.0-alpha.6` | 待评估 |

### 📋 架构改进路线图（长期）

| 优先级 | 改进项 | 复杂度 | 说明 |
|--------|--------|--------|------|
| P1 | Deployer 接口拆分为聚焦接口 | 高 | 拆为 ContainerDeployer/DNSManager/SSLManager 等 |
| P1 | 消除 `interface{}` 返回值 | 高 | 为每个方法定义具体 struct |
| P2 | 全局状态迁移到 DB/Redis | 中 | tasks/backupApps/portForwards |
| P2 | 统一错误码体系 | 中 | sentinel errors + 错误码 |
| P3 | Bridge 逐步解耦为独立 Service | 高 | 从 (b *Bridge) 改为独立 struct |
| P3 | README 数据同步更新 | 低 | MCP 工具数、REST 端点数等 |

### 📌 里程碑提醒

- **v1.2 Unreleased**: 包含指标持久化 + 定时任务系统（PR #109 已合并），需要执行版本完成检查清单（见踩坑记录 #20）
- **v1.3 规划中**: #113（命令注入）和 #114（Gitleaks）已修复，建议将 #117（全局可变状态）作为 v1.3 的 P0 项
- **Dependabot PR #111**: Go 依赖批量更新（11 个），需要审查后合并
- **分支保护操作**: 使用 PAT token 通过 API 临时删除/恢复保护（见踩坑记录 #35），流程：备份配置 → DELETE → merge → PUT 恢复

## Agent 操作前必读清单

&gt; **每次 Agent 接手任务时，按以下顺序执行：**

1. **读取本文件** — 完整阅读 `.github/ai-guide.md`，了解项目规范和踩坑记录
2. **检查里程碑** — 通过 GitHub API 查看当前 milestone 的 open issues，了解版本进度
3. **检查 open issues** — 查看是否有与自己任务相关的 open issues，避免重复工作
4. **检查 Dependabot PRs** — 如果有积压的 Dependabot PR，评估是否需要先处理
5. **遵循 Git 工作流** — 使用 Conventional Commits + Agent Trailer，创建 feature 分支
6. **修改代码后同步文档** — 按照文档同步规则更新相关文档
7. **发现新问题及时记录** — 在本文件的「待办事项与已知问题」章节添加，并创建 GitHub Issue
8. **完成任务后更新本文件** — 如果修复了踩坑记录中的问题，标注状态；如果发现新坑，添加新记录
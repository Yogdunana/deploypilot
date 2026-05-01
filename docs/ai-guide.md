# DeployPilot AI Operations Guide

> 本文件为唯一权威版本，位于 `docs/ai-guide.md`。
> 此文件为 AI 助手操作本项目时的参考指南，避免重复踩坑。
> **最后更新**: 2026-05-01 (Issue 标准规范重写 + Phase Issue Form + Labels 体系 + 踩坑 #43)

---

## 项目概览

- **仓库**: https://github.com/Yogdunana/deploypilot
- **所有者**: Yogdunana
- **技术栈**: Go 1.23 / Gin / GORM (后端) + Vue 3.5 / TypeScript / Vite 6 / Tailwind CSS 4 (前端)
- **架构**: 三二进制架构 — `deploypilot` (CLI)、`api-server` (REST API + Web)、`mcp-server` (MCP stdio)
- **许可证**: BSL 1.1 (Change Date 2029-04-28, Change License MIT)

---

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

### 29. [安全] ListImages 的 filter 参数存在命令注入漏洞（Critical）
`internal/service/bridge.go` 中 `ListImages` 函数将 `filter` 参数直接拼接到 shell 命令：
```go
dockerCmd += " | grep " + filter  // filter 未转义！
```
攻击者可传入 `; rm -rf /` 或 `$(malicious)` 执行任意命令。**修复**: 对 filter 做白名单校验或使用 `shellQuote()`。
**相关 Issue**: #113

### 30. [安全] CI 中安全扫描设置了 continue-on-error
`.github/workflows/ci.yml` 中 `gitleaks`、`govulncheck`、`npm audit` 均设置了 `continue-on-error: true`，意味着即使发现密钥泄露或严重漏洞，CI 也不会阻断。其中 gitleaks 最严重——密钥扫描形同虚设。
- **gitleaks**: 无理由设为 non-blocking，应立即移除 `continue-on-error`
- **govulncheck**: 注释说明需 Go 1.24 修复 stdlib 漏洞，但第三方依赖漏洞也被忽略了
- **npm audit**: 应至少对 critical 级别阻断
**相关 Issue**: #114

### 31. [安全] SSH 连接静默回退到 root 用户
`internal/service/bridge.go` 的 `getRemoteExecutor` 和 `PortForward` 中，当服务器记录的 `username` 为空时，静默回退到 `"root"`，无任何日志警告。违反最小权限原则。
**修复**: 移除 root 回退改为返回错误，或至少添加 `slog.Warn` 日志。
**相关 Issue**: #115

### 32. [安全] DNS 服务吞掉错误（返回 nil error + 错误 map）
`internal/service/dns_service.go` 中 `DNSCreateRecord`、`DNSListRecords` 等方法在出错时返回 `nil` error，将错误信息包装到 `map[string]interface{}` 中。这导致调用方无法使用标准 `if err != nil` 模式，`BatchDNS` 中出现混乱的双重判断逻辑。
**修复**: 让错误正常返回，不要吞掉。

### 33. [安全] Backup 和 PortForward 中 shellQuote 未一致使用
`internal/service/backup_service.go:41` 中 `containerName` 和 `backupFile` 未使用 `shellQuote()`。`internal/service/bridge.go:824` 中 `PortForward` 的 `pkill` 命令中 `RemoteHost` 也未转义。虽然这些参数来自数据库，但如果应用名被污染可能导致命令注入。
**修复**: 所有 shell 命令拼接处统一使用 `shellQuote()`。

### 34. [安全] JWT Secret 配置注入绕过长度校验
`cmd/api-server/main.go:91-95` 中，如果 `config.yaml` 设置了 `auth.jwt_secret`，会直接注入环境变量。但 `getJWTSecret()` 的 16 字符长度校验只检查环境变量值，config 层的 `Load()` 没有前置校验。如果配置了短密钥，会在运行时才报错，错误信息不够明确。
**修复**: 在 `main.go` 中注入前增加长度校验。

### 35. [安全] Gitleaks Action 需要 GITHUB_TOKEN 环境变量
`gitleaks/gitleaks-action@v2` 的最新版本要求配置 `GITHUB_TOKEN` 环境变量才能扫描 Pull Requests。如果缺少此配置，action 会报错 `GITHUB_TOKEN is now required to scan pull requests` 并失败。
**修复**: 在 Gitleaks step 中添加 `env: GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}`。
**相关 PR**: #119

### 36. [安全] 接口拆分后工具注册必须匹配 handler 的接口类型
将 God Interface 拆分为子接口后，每个 `register_*.go` 中的工具注册函数接收对应的子接口类型。如果 handler 函数签名是 `handleXxx(ctx, d MonitorService, ...)`，则该工具必须在 `register_monitor.go` 中注册（传入 `MonitorService`），不能注册在 `register_deploy.go`（传入 `ContainerDeployer`）中，否则 Lint 会报类型不匹配。
**反例**: `handleHealContainer` 期望 `MonitorService`，但注册在 `register_deploy.go` 中传入 `ContainerDeployer`，导致 Lint 失败。
**修复**: 将工具注册移到与 handler 接口类型匹配的 register 文件中。如果方法同时属于多个子接口（如 `HealContainer` 同时在 `ContainerDeployer` 和 `MonitorService` 中），选择与 handler 签名匹配的那个。
**相关 PR**: #122

### 29-FE. [前端] 前端新文件推送后远程可能缺失

通过 GitHub MCP `create_or_update_file` 推送新文件时，如果文件内容过大或网络问题，文件可能未成功写入远程分支但本地认为已推送。**必须**在推送后通过 `get_file_contents` 验证文件确实存在于远程。

- **排查方法**: 推送后立即 `get_file_contents(path, branch)` 确认返回 200 而非 404
- **反例**: PR #173 中 `api/modules/apikeys.ts` 本地存在但远程 404，导致 CI Build Frontend 失败

### 30-FE. [前端] vue-tsc -b 的 tsbuildinfo 缓存可能导致类型检查跳过新文件

`vue-tsc -b`（build mode）使用 `tsconfig.tsbuildinfo` 增量编译缓存。如果该文件提交在仓库中且记录的文件列表不包含新增文件，`vue-tsc -b` 可能跳过对新文件的类型检查，导致 `Cannot find module` 错误在 CI 中出现但本地不出现。

- **修复**: 清空或删除 `tsconfig.tsbuildinfo` 文件，强制 `vue-tsc -b` 全量重建
- **最佳实践**: 将 `tsconfig.tsbuildinfo` 加入 `.gitignore`，不要提交到仓库

### 31-FE. [前端] 前端组件导入路径必须与文件实际位置完全匹配

Vue 组件中的 `import` 路径必须与 `web/src/` 下的文件结构完全一致。常见错误：

- `import { twofaApi } from '@/api/twofa'` — 错误（缺少 `modules/`）
- `import * as twofaApi from '@/api/modules/twofa'` — 正确

同时注意导入方式必须与模块的导出方式匹配：

- 模块使用命名导出（`export function setup()`）→ 用 `import * as twofaApi from '...'` 或 `import { setup, confirm } from '...'`
- 不要假设模块有默认导出或命名空间导出

- **反例**: PR #173 中 `SecuritySettings.vue` 使用 `import { twofaApi } from '@/api/twofa'`，路径错误且导入方式不匹配，导致 CI 报 `Cannot find module '@/api/twofa'`

### 37. 新增 struct 字段到 service 但忘记更新 NewXxxService 构造函数签名
创建新的 service struct 时，如果构造函数接收 `*sandbox.Sandbox` 参数但 struct 中不再使用该字段，需要：
1. 从 struct 中移除未使用的字段
2. 如果 API 层调用 `NewXxxService(db, sb)`，构造函数签名仍需保留 `sb` 参数（或同步修改调用方）
3. **排查方法**: `golangci-lint` 的 `unused` 检查会报错

**反例**: Phase 5.5 FirewallService 移除了 `sb` 字段但忘记保留 import，导致编译失败。

### 38. router.go 中变量作用域导致编译错误
在 router.go 的 `servers := protected.Group(...)` 块内创建的变量（如 `sshAPI`），在块外不可见。如果其他路由组需要引用该变量，必须在块外创建。
**正确做法**: 在 servers 组内使用 `NewSSHAPI(db).ListServerAuthorizations` 内联调用，在组外创建 `sshAPI := NewSSHAPI(db)` 供其他组使用。

**反例**: Phase 5.6 中 `sshAPI` 在 servers 组内创建，`sshGroup` 在组外引用导致 `undefined: sshAPI`。

### 39. 同一 package 内函数名重复导致编译失败
从 bridge.go 提取方法到独立 service 文件时，如果新文件定义了与 bridge.go 同名的函数（如 `shellQuote`），会导致 `shellQuote redeclared in this block` 编译错误。
**修复**: 删除重复定义，使用 bridge.go 中的原始定义。

**反例**: Phase 5.4 file_manager_service.go 定义了 `shellQuote`，与 bridge.go 冲突。

### 40. exec.Close() 返回值必须检查（errcheck 规则）
golangci-lint 的 `errcheck` 规则要求检查所有 error 返回值，包括 `defer exec.Close()`。
**修复**: 使用 `defer func() { _ = exec.Close() }()` 包装。

**反例**: Phase 5.4 中 3 处 `defer exec.Close()` 导致 Lint 失败。

### 41. Roadmap 表格缺少列头导致渲染异常
Markdown 表格的列头行必须与数据行列数一致。如果列头只有 2 列 `| Phase | 内容 |` 但数据行有 3 列（含状态），表格渲染会异常。
**修复**: 确保所有版本的表格列头统一为 `| Phase | 内容 | 状态 |`。

### 42. docs/wiki/ 目录更新不会自动同步到 GitHub Wiki
GitHub Wiki 是一个独立的 git 仓库（`<repo>.wiki.git`），修改 `docs/wiki/` 中的文件不会自动反映到 GitHub Wiki 页面。
**解决方案**: 使用 GitHub Actions（如 `spencerblack/actions-wiki@v1`）在 push 到 main 时自动同步。

### 43. Issue 元数据不完整导致进度追踪困难
创建 Issue 时如果不设置 assignee、labels、milestone、project，会导致 Project Board 和 Milestone 进度不准确。
**解决方案**: 使用 Issue Form 模板自动设置 assignee 和 labels；Guide 中定义了 Issue 标准规范，每个 Issue 必须包含完整元数据。

---

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

### 项目结构速查

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
| 框架 | Vue 3.5 + Composition API (`<script setup lang="ts">`) |
| 语言 | TypeScript (strict 模式) |
| 构建工具 | Vite 6 |
| CSS | Tailwind CSS 4 + Radix Vue |
| 状态管理 | Pinia 2 |
| 路由 | Vue Router 4 (History 模式) |
| HTTP | Axios |
| 包管理器 | npm |
| 组件库 | 自定义 shadcn/ui 风格组件（Button, Dialog, AlertDialog, Tabs, Badge, Input 等），CVA + lucide-vue-next 图标 |
| 测试 | Vitest 4 + @vue/test-utils + jsdom |
| 嵌入方式 | Go `embed` 嵌入到后端二进制（`web/embed.go`） |

**前端测试命令**: `npm test`（在 `web/` 目录下）
**CI 集成**: `Build Frontend` job 中包含 `npm test` 步骤

---

## 文档同步规则

修改以下内容时，必须同步更新所有相关文档：
1. **配置字段变更** → `configs/config.yaml.example` + `docs/wiki/Configuration.md` + `README.md` + `README_zh-CN.md`
2. **新增 API 路由** → `internal/api/router.go` + Swagger 注释 + `docs/swagger/`
3. **新增 MCP 工具** → `internal/mcp/server.go` + `docs/mcp-tools.md` + `docs/wiki/MCP-Integration.md`
4. **数据库 Schema 变更** → `internal/database/database.go` + 测试文件中的 CREATE TABLE
5. **Roadmap Phase 完成** → `docs/wiki/Roadmap.md` + 关联 Issue + Milestone 状态
6. **版本完成** → Milestone 关闭 + CHANGELOG 更新 + Tag/Release 创建 + 孤立 Issue 关联

## 全资源同步要求

> **每个 Phase 完成后，Agent 必须检查并同步以下所有 GitHub 资源。**

### 需要同步的资源清单

| 资源 | URL | 同步内容 | 检查频率 |
|------|-----|----------|----------|
| **Roadmap** | `docs/wiki/Roadmap.md` | Phase 状态标记 ✅ | 每个 Phase 完成后 |
| **GitHub Wiki** | `github.com/Yogdunana/deploypilot/wiki` | 由 Action 自动同步 | push 到 main 时自动 |
| **Issues** | `github.com/Yogdunana/deploypilot/issues` | 关闭已完成的 Issue | 每个 Phase 完成后 |
| **Milestones** | `github.com/Yogdunana/deploypilot/milestones` | 更新 open/closed 计数 | 每个 Phase 完成后 |
| **Projects** | `github.com/users/Yogdunana/projects/3` | 更新 Project 卡片状态 | 每个版本完成后 |
| **Discussions** | `github.com/Yogdunana/deploypilot/discussions` | 发布版本公告/变更日志 | 每个版本发布后 |
| **ai-guide.md** | `docs/ai-guide.md` | 更新踩坑记录 + 当前待办 | 每个 Phase 完成后 |

### Phase 完成后的同步 Checklist

每个 Phase 完成并合并 PR 后，Agent 必须执行以下检查：

- [ ] **Roadmap**: `docs/wiki/Roadmap.md` 中该 Phase 标记为 `✅ 已完成 (PR #xxx)`
- [ ] **Issue**: 确认 Issue 已关闭（通过 PR 的 `Closes #xx` 或手动关闭）
- [ ] **Milestone**: 确认 Milestone 的 open/closed 计数正确
- [ ] **踩坑记录**: 如果开发中遇到新坑，添加到 `docs/ai-guide.md` 踩坑记录章节
- [ ] **当前待办**: 更新 `docs/ai-guide.md` 中"当前待办"章节的进度
- [ ] **Wiki 同步**: 确认 push 到 main 后 wiki-sync Action 自动运行（无需手动操作）

### 版本完成后的同步 Checklist

一个版本所有 Phase 完成后，额外执行：

- [ ] **Milestone 关闭**: 通过 API 关闭 Milestone
- [ ] **CHANGELOG**: 更新 CHANGELOG.md
- [ ] **Tag + Release**: 创建 tag 并发布 Release
- [ ] **Project Board**: 更新 `DeployPilot Roadmap` Project 中该版本的状态
- [ ] **Discussions**: 发布版本发布公告
- [ ] **孤立 Issue**: 检查所有 open issue 都关联了 milestone

## Issue & Milestone 生命周期管理

> 这是 AI Agent 协作开发的核心流程。所有 Agent 必须遵循此流程。

### Issue 标准规范

**每个 Issue 必须包含的元数据**：

| 字段 | 要求 | 示例 |
|------|------|------|
| **Title** | `[vX.X] Phase 名称` 或 `🐛 Bug: 描述` | `[v1.5] Event Bus System` |
| **Assignees** | 必须指定 `@Yogdunana` | — |
| **Labels** | 至少 2 个: type + priority | `enhancement`, `priority: medium` |
| **Milestone** | 必须关联到对应版本 | `v1.5: Notification & Alerting` |
| **Project** | 必须添加到 DeployPilot Roadmap | — |
| **Body** | 必须包含 Tasks (checkbox) + Notes | — |

**标准 Labels 体系**：

| 类别 | Labels | 说明 |
|------|--------|------|
| **Type** | `bug`, `enhancement`, `documentation`, `refactor`, `testing` | 工作类型 |
| **Priority** | `priority: critical`, `priority: high`, `priority: medium`, `priority: low` | 优先级 |
| **Status** | `in progress`, `blocked` | 当前状态 |
| **Area** | `area: backend`, `area: frontend`, `area: infra`, `area: docs` | 代码区域 |
| **Special** | `security`, `architecture`, `ui/ux`, `developer-experience` | 特殊标记 |

### Issue 生命周期

```
创建 → 认领 → 开发 → PR 关联 → Review → 合并关闭
```

#### 1. 创建 Issue
- 使用 Issue Form 模板（Phase Issue / Bug Report / Feature Request）
- 自动获得: assignee (@Yogdunana) + labels (enhancement, in progress) + title prefix
- 手动关联: Milestone + Project Board
- Body 必须包含: Tasks (checkbox list) + Notes

#### 2. 认领 Issue（开始开发前）
- 确认 assignee 已设置
- 确认 milestone 已关联
- 确认 labels 完整（type + priority + area）
- 确认 Project Board 已添加
- 创建开发分支: `feat/phase-X.X` 或 `fix/xxx`

#### 3. 开发过程中
- 在 Issue 中评论进度（可选）
- 遇到阻塞时: 添加 `blocked` label + 评论说明原因
- PR 创建后: PR 描述中写 `Closes #xx` 或 `Fixes #xx`
- Development 面板会自动关联 PR

#### 4. Review 阶段
- CI 通过后等待 review
- Review 通过后合并

#### 5. 合并关闭
- PR 合并时通过 `Closes #xx` 自动关闭 Issue
- 如果未自动关闭，手动关闭并评论说明
- 从 Project Board 移到 Done 列

### Development 关联说明

> **何时关联**: PR 创建时自动关联，无需手动操作。
>
> **关联方式**: PR 描述中包含 `Closes #xx` / `Fixes #xx` / `Resolves #xx`。
>
> **效果**: GitHub 自动在 Issue 的 Development 面板显示关联的 PR 和分支。
>
> **注意**: 不要在开发前就关联分支，应在 PR 创建时通过 `Closes #xx` 一次性关联。

### Milestone 管理

**何时创建**: 版本开始开发前
**标题格式**: `v1.x: 简短描述`
**何时关闭**: 版本所有 Phase 完成 + Release 创建后

### 新需求接入流程

```
评估需求 → 创建/确认 Milestone → 创建 Issue (用 Form) → 关联 Project → 开发 → PR (Closes #xx) → 合并 → 关闭 Issue
```

### 多 Agent 协作规则

1. **开始前检查**: Issue 是否有 assignee / in-progress label
2. **开发中更新**: 遇到阻塞及时评论
3. **完成后确认**: Issue 自动/手动关闭

### 自动化检查清单

**Phase 完成后**:
- [ ] Issue 已关闭 (Closes #xx)
- [ ] Milestone 关联正确
- [ ] Roadmap.md 标记 ✅
- [ ] 踩坑记录更新（如有新坑）
- [ ] 当前待办更新
- [ ] **Issue 元数据**: 确认 assignee + labels + milestone + project 完整

**版本完成后**:
- [ ] Milestone 0 open issues → 关闭
- [ ] CHANGELOG 更新
- [ ] Tag + Release 创建
- [ ] Project Board 更新
- [ ] Discussions 发布公告
- [ ] 重复 Issue 清理: 检查新版本 Milestone 中的 Issue 是否与已实现功能重复，关闭并评论 "已在 vX.X Phase X.X 中实现"

```

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

> 参考：TRAE 技术专家小夏《新范式下 Agent 如何参与开发》

### Commit 规范

每个 commit 应当能独立描述「做了什么、为什么、上下文是什么」。使用 Git Trailer 记录 Agent 上下文：

```
<type>(<scope>): <summary>

<正文：描述本次变更的背景与动机>

Agent-Task: <原始任务描述或任务 ID>
Agent-Model: <使用的模型>
Agent-Decision: <关键设计决策及理由>
Agent-Limitation: <已知局限或后续 TODO>
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

---

## 里程碑总览

| # | 版本 | 名称 | 状态 | 子 Issue |
|---|------|------|------|---------|
| 1 | v1.1 | Security & Stability | ✅ 已关闭 | — |
| 2 | v1.2 | Adapter Layer Refactor | ✅ 已关闭 | — |
| 3 | v1.3 | Deployment Enhancement | ✅ 已关闭 | #8, #9 (2 closed) |
| 4 | v1.4 | Enterprise Features | ✅ 已关闭 | #128-#132 |
| 5 | v1.5 | Notification & Alerting | 🔄 进行中 (6/9 phases) | #133-#136 |
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

## 当前待办

### v1.3 ✅ 已完成
- Milestone #3 已关闭 (2026-04-30)
- Issue #8 (1Panel Enhancement) → PR #166 merged
- Issue #9 (MCP Session Context) → PR #167 merged

### v1.4 ✅ 已完成
- Milestone #4 已关闭
- Issues #128-#132 (Enterprise Features) 全部完成

### v1.5 🔄 进行中 (6/9 phases done)
- Phase 5.1-5.6 已完成 (PR #188-#195)
- Phase 5.7-5.9 待开始
- Milestone #5 open (1 open issue remaining)

### 其他待处理
- [ ] Dependabot PR #82: bump @xterm/xterm 5.5.0 → 6.0.0 (breaking change, 需评估)
- [ ] 发布 v1.3.0 release tag
- [ ] 发布 v1.4.0 release tag

---

## SOLO Agent 接手指南

> **接手日期**: 2026-04-30
> **接手 Agent**: SOLO (Claude)
> **操作身份**: Yogdunana（仓库 Owner）

### 接手背景

2026-04-30 SOLO Agent 全面接手 DeployPilot 项目维护，以 Yogdunana 身份进行所有 Git 操作。

### 当前项目状态

- **最新版本**: v1.4.0
- **开发中**: v1.5.0 Notification & Alerting（Phase 5.1-5.6 已完成，5.7-5.9 待开始）
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

- [ ] 推进 v1.5 剩余 Phase（5.7-5.9）
- [ ] 发布 v1.5.0
- [ ] 开始 v1.6 Monitoring & Observability (Issues #137-#141)

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

---

## Agent 交接检查清单

当另一个 Agent 接手时，必须：

1. **阅读本文件**: 完整阅读 ai-guide.md，特别是踩坑记录 #1-#36 及前端踩坑 #29-FE ~ #31-FE
2. **确认身份**: 使用 Yogdunana 身份操作（通过 MCP GitHub 工具）
3. **检查待处理**: 查看「当前待办」列表，确认当前进度
4. **更新记录**: 在「当前待办」中标记已完成的项，添加新发现的项
5. **遵循规范**: 严格遵守 Git 操作规范、决策原则、工作流程
6. **更新日期**: 修改「最后更新」日期和接手信息

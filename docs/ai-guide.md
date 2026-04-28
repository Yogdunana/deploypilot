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

## 版本发布流程

1. 确认 main CI 全部通过（包括 build-and-push 和 Release）
2. 创建 git tag: `git tag -a v1.x.0 -m "v1.x.0 — Title"`
3. 推送 tag: `git push origin v1.x.0`
4. Release workflow 自动触发，创建 GitHub Release + 构建产物
5. 如需手动创建 Release，用 GitHub API

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

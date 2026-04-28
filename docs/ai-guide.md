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

## 项目结构关键路径

| 路径 | 说明 |
|------|------|
| `internal/api/router.go` | 所有 API 路由注册 |
| `internal/database/database.go` | 数据库迁移定义 |
| `internal/config/config.go` | 配置结构体定义 |
| `configs/config.yaml.example` | 配置文件示例（权威来源） |
| `internal/mcp/server.go` | MCP 工具注册（63 个） |
| `docs/mcp-tools.md` | MCP 工具规范表（权威来源） |
| `docs/wiki/Roadmap.md` | 版本路线图 |
| `.github/workflows/ci.yml` | CI 工作流 |
| `.github/workflows/docker.yml` | Docker 构建推送 |
| `.github/workflows/release.yml` | Release 发布 |
| `internal/version/version.go` | 版本号（默认 "dev"） |

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

---
title: DeployPilot AI Operations Guide
version: 2.1.0
repository: Yogdunana/deploypilot
source: 本文件 + ai-guide-reference.md 共同构成唯一权威版本
last_updated: 2026-05-02
scope: AI 助手操作本项目时的参考指南，避免重复踩坑
---

# DeployPilot AI 工作流程规范

> **阅读指引**：本文件包含你需要的全部核心规则（~90 行）。详细参考内容在 `ai-guide-reference.md` 中，按需查阅即可。

## Agent 快速指令

> **你是 DeployPilot 仓库的 AI 开发助手。以下是你的操作规范，必须严格遵守。**

### 你是谁

- 仓库：`Yogdunana/deploypilot`
- 身份：所有 commits/PRs 使用 `Yogdunana` 身份
- PAT Token：通过环境变量 `GH_TOKEN` 或 `GITHUB_TOKEN` 提供，也可配置在 `~/.config/gh/hosts.yml` 中
- 主要通过 MCP GitHub 工具操作，PAT 用于 MCP 无法处理的场景

### 核心铁律（违反任何一条都会导致 CI 失败或操作被拒）

1. **禁止直接 push 到 main** -- 必须走 `feature branch -> PR -> squash merge`
2. **禁止使用 `--merge` 合并 PR** -- 必须用 `--squash`
3. **禁止修改 `internal/version/version.go`** -- 版本号通过 `-ldflags` 注入
4. **禁止硬编码 Token/密码** -- 通过环境变量获取
5. **禁止提交 `.env` 文件`
6. **修改 `go.mod` 后必须执行 `go mod tidy`**
7. **所有 shell 命令参数必须 `shellQuote()`** -- 防命令注入
8. **新增接口方法必须同步 mock** -- 否则编译失败
9. **新增 App 模型字段必须同步 4 个测试文件的 CREATE TABLE** -- 否则运行时崩溃
10. **新增列必须用迁移** -- 禁止直接改 CREATE TABLE
11. **PR 必须关联 Issue** -- 描述中写 `Fixes #xx` 或 `Closes #xx`
12. **合并 PR 后必须等 main CI 全部通过** -- 不能仅凭 branch CI
13. **使用 `flowchart LR`** -- 禁止 `graph LR`（GitHub Mermaid 兼容性）
14. **使用 `--password-stdin` 做 Docker login** -- 禁止 `-p pass`
15. **使用原生 `createPinia()` 测试 Pinia** -- 禁止 `@pinia/testing`

### 操作流程速查

```
开发任务 -> brainstorming -> writing-plans -> TDD 实现 -> git-commit -> gh-cli 创建 PR -> 等 CI 通过 -> squash merge -> 等 main CI -> 删分支
```

### 遇到异常怎么办

| 异常场景 | 处理方式 |
|----------|----------|
| `make check` 报错 | 读错误信息定位检查项，查 `ai-guide-reference.md` 附录 B 踩坑记录 |
| CI 某项检查失败 | `gh run view --log` 查日志；常见原因：go.sum（#1）、import（#8/#9/#11/#27）、mock（#4/#25）、DDL（#24） |
| 分支保护解除失败（403） | 确认 PAT 有 `repo` scope + admin 权限，检查是否过期 |
| PR 合并被拒（405/409） | 确认 `--squash`、分支保护已解除、CI 全部通过 |
| `go mod tidy` 后仍报 missing go.sum | 先 `go mod download` 再 `go mod tidy` |
| 前端 `npm ci` 失败 | 删 `node_modules` + `package-lock.json`，重新 `npm install`，检查 Node >= 20 |
| Docker 构建失败 | 检查 go.sum 完整性（#1）、Dockerfile Go 版本与 go.mod 一致 |
| MCP 参数解析失败 | 检查 `CallToolParams.Arguments` 断言为 `map[string]any`（mcp-go v0.47.0） |
| mock 不匹配 | 查 #4（key 一致性）和 #25（接口方法同步） |
| 数据库迁移失败 | 查 #5（用 ALTER TABLE 而非改 CREATE TABLE） |

**通用排查原则**：先查踩坑记录（80% 是已知坑）→ 读完整错误信息 → 未知问题修复后记录。

### 遇到新坑时

在 `ai-guide-reference.md` 附录 B 对应类别表格末尾追加新记录，编号在现有最大值基础上 +1 递增。格式见附录 B 开头的"维护规范"。

### 详细参考索引

需要详细信息时，查阅 `ai-guide-reference.md` 对应章节：

| 章节 | 内容 | 何时查阅 |
|------|------|----------|
| 第 1 章 | 仓库概览、技术栈、架构、凭证 | 开始任何任务前 |
| 第 2 章 | Git 分支/Commit/PR/Issue/Tag 流程、仓库模板、Label、Milestone | 提交代码/创建 PR/Issue 时 |
| 第 3 章 | CI 7 项检查、依赖关系、本地预检查、Release 流程 | 提交 PR 前 / 发版时 |
| 第 4 章 | Go/Vue/DB 代码规范、文档同步、大文件处理、MCP 开发 | 写代码时 |
| 第 5 章 | 安全扫描、编码安全、已知漏洞 | 处理安全问题/写 shell 命令时 |
| 第 6 章 | 测试规范、覆盖率要求 | 写测试时 |
| 第 7 章 | 环境搭建、新增 MCP/API 流程、依赖升级、Code Review | 新功能开发/审查 PR 时 |
| 附录 A | 项目结构关键路径 | 需要定位文件时 |
| 附录 B | 踩坑记录（按类别索引） | 遇到奇怪错误时 |
| 附录 C | 常用命令速查 | 需要命令但记不清参数时 |
| 附录 D | 全部 Skill 触发条件和使用说明 | 需要调用 Skill 时 |
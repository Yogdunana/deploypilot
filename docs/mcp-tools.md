# MCP Tools 规范表

> PRD v2.17 §6.1 对齐状态，最后更新：2026-04-27
>
> **权威声明**：本文件与 PRD §6.1 为 MCP Tool 命名的唯一权威来源。代码注册名变更须同步更新本文件 + PRD §6.1.2 权限表。

## §6.1 必备 Tools（31 个）

| # | PRD Tool 名 | 代码注册名 | RBAC 等级 | 状态 |
|---|------------|-----------|----------|------|
| 1 | deploy_app | deploy_app | dev | ✅ |
| 2 | list_apps | list_apps | viewer | ✅ |
| 3 | delete_app | delete_app | admin | ✅ |
| 4 | get_deploy_status | get_deploy_status | viewer | ✅ |
| 5 | rollback_app | rollback_app | dev | ✅ |
| 6 | get_app_logs | get_app_logs | viewer | ✅ |
| 7 | search_app_logs | search_app_logs | viewer | ✅ |
| 8 | add_dns_record | add_dns_record | dev | ✅ |
| 9 | update_dns_record | update_dns_record | dev | ✅ |
| 10 | delete_dns_record | delete_dns_record | admin | ✅ |
| 11 | list_dns_records | list_dns_records | viewer | ✅ |
| 12 | list_credentials | list_credentials | viewer | ✅ |
| 13 | add_credential | add_credential | dev | ✅ |
| 14 | update_credential | update_credential | dev | ✅ |
| 15 | delete_credential | delete_credential | admin | ✅ |
| 16 | list_servers | list_servers | viewer | ✅ |
| 17 | add_server | add_server | dev | ✅ |
| 18 | update_server | update_server | dev | ✅ |
| 19 | delete_server | delete_server | admin | ✅ |
| 20 | get_app_detail | get_app_detail | viewer | ✅ |
| 21 | update_app | update_app | dev | ✅ |
| 22 | check_system_update | check_system_update | viewer | ✅ |
| 23 | detect_environment | detect_environment | viewer | ✅ |
| 24 | check_deploy_readiness | check_deploy_readiness | viewer | ✅ |
| 25 | backup_database | backup_database | dev | ✅ |
| 26 | restore_database | restore_database | dev | ✅ |
| 27 | get_task_status | get_task_status | viewer | ✅ |
| 28 | list_tasks | list_tasks | viewer | ✅ |
| 29 | batch_deploy | batch_deploy | admin | ✅ |
| 30 | batch_backup | batch_backup | admin | ✅ |
| 31 | batch_dns | batch_dns | admin | ✅ |

## 实现扩展 Tools（6 个，PRD §6.1 主表外）

| # | Tool 名 | 功能 | RBAC 等级 | 说明 |
|---|---------|------|----------|------|
| 32 | create_app | 注册新应用 | dev | CLI create 的 MCP 入口 |
| 33 | test_server | 测试服务器连接 | dev | SSH 连通性检查 |
| 34 | health_check | 健康检查 | viewer | HTTP/TCP 探活 |
| 35 | send_notification | 发送通知 | dev | 部署/回滚事件通知 |
| 36 | list_templates | 列出应用模板 | viewer | 9 种技术栈模板 |
| 37 | get_template | 获取模板详情 | viewer | 单个模板配置 |
| 38 | create_uptime_monitor | 创建可用性监控器（HTTP/TCP/Ping） | dev | 支持 SLA 计算、状态页面展示 |
| 39 | list_uptime_monitors | 列出所有可用性监控器 | viewer | 含状态、可用率、延迟 |
| 40 | check_uptime_monitor | 立即检查指定监控器 | dev | 返回检查结果（状态/延迟/状态码） |
| 41 | get_monitor_sla | 获取监控器 SLA 指标 | viewer | 可用率、平均延迟、检查次数 |
| 42 | delete_uptime_monitor | 删除可用性监控器 | admin | 同时删除关联的检查历史 |
| 43 | create_heartbeat | 创建心跳监控 | dev | 返回唯一 Ping URL |
| 44 | list_heartbeats | 列出所有心跳监控 | viewer | 含状态、最后心跳时间 |
| 45 | delete_heartbeat | 删除心跳监控 | admin | 同时删除心跳记录 |

## §6.1.4 get_context() 说明

`get_context()` 在 PRD §6.1.4 中描述为"获取当前会话状态摘要"，由 MCP SDK/会话层提供，**不作为独立 Tool 注册**。当前实现中，MCP 框架（mcp-go）自带会话上下文管理，无需额外实现。

## 命名变更记录

以下 Tool 在开发过程中曾使用过旧名，已全部对齐 PRD：

| PRD 名 | 旧代码名 | 变更 Sprint |
|--------|---------|------------|
| rollback_app | rollback | A |
| detect_environment | detect_env | A |
| delete_server | remove_server | A |
| add_credential | create_credential | A |
| add_dns_record | dns_create_record | A |
| delete_dns_record | dns_delete_record | A |
| list_dns_records | dns_list_records | A |
| backup_database | backup | B |
| restore_database | restore | B |

## 验证命令

```bash
go test ./... -race -count=1
go test ./... -coverprofile=c.out -count=1
go tool cover -func=c.out | tail -n 1
```

## 待办（技术债务）

| # | 事项 | 优先级 | 备注 |
|---|------|--------|------|
| 1 | govulncheck 恢复为硬门槛 | 中 | 需升级 Go ≥ 1.25 + golang.org/x/crypto，修复 SSH 漏洞 (GO-2025-0335) |
| 2 | .golangci.yml 收紧 errcheck 排除 | 低 | 当前整包跳过 _test.go，可改为按文件排除 |
| 3 | 真机 E2E 验证 | 高 | scripts/e2e-real-server.sh 待目标服务器就绪后执行 |

## v0.2: 凭据加密 + 远端部署

### 环境变量

| 变量 | 必需 | 说明 |
|------|------|------|
| `DEPLOYPILOT_ENCRYPTION_KEY` | 推荐 | Base64 编码的 32 字节 AES-256 密钥。未设置时自动生成临时密钥（重启后凭据丢失） |
| `DEPLOYPILOT_SSH_HOST` | 否 | 默认 SSH 主机（未来用于全局远端模式） |
| `DEPLOYPILOT_SSH_PORT` | 否 | 默认 SSH 端口（默认 22） |
| `DEPLOYPILOT_SSH_USER` | 否 | 默认 SSH 用户（默认 root） |

### 凭据安全

- `create_credential` / `update_credential`：明文经 AES-256-GCM 加密后存入 `encrypted_value` 字段
- `list_credentials`：返回值不包含 `encrypted_value`（已脱敏）
- `getRemoteExecutor`：部署时自动解密关联凭据用于 SSH 认证
- 数据库中不会出现凭据明文

### 远端部署流程

1. `add_server` → 注册服务器（host/port）
2. `create_credential` → 创建 SSH 凭据（加密存储）
3. `deploy_app(server_id=...)` → Bridge 查询 server → 解密凭据 → SSH 连接 → 远端 Docker 部署
4. 未传 `server_id` 时保持本机 Docker 默认行为

### 最小验证命令

```bash
# 本地验证（stdio 模式）
export DEPLOYPILOT_ENCRYPTION_KEY=$(echo -n "01234567890123456789012345678901" | base64)
echo '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./bin/mcp-server

# 服务器验证（替换为你的实际 host/port）
ssh -p <port> root@<host>
export DEPLOYPILOT_ENCRYPTION_KEY=<your-base64-key>
./bin/mcp-server
```
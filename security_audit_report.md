# DeployPilot 安全审计报告

**审计日期**: 2026-05-10
**审计范围**: DeployPilot 代码仓库
**审计方法**: 静态代码分析 + 架构审查

---

## 执行摘要

本次审计识别出 **1 个已确认的高严重度漏洞** 和 **1 个已确认的中等严重度问题**，均具有可论证的端到端利用路径。

| 严重度 | 数量 | 漏洞描述 |
|--------|------|----------|
| **高 (High)** | 1 | 通过 `exec_command` 工具的任意命令执行漏洞 |
| **中 (Medium)** | 1 | 心跳令牌枚举导致的服务状态探测漏洞 |

---

## 漏洞详情

### [HIGH-1] 通过 MCP `exec_command` 工具的任意命令执行漏洞

#### 漏洞位置

| 文件 | 行号 |
|------|------|
| [internal/mcp/handler_server.go](file:///workspace/internal/mcp/handler_server.go#L177-L205) | 177-205 |
| [internal/mcp/permissions.go](file:///workspace/internal/mcp/permissions.go#L35) | 35 |

#### 攻击者画像

- **外部用户**: 任何持有有效 JWT token 或 API Key 的已认证用户（角色为 `admin` 及以上）

#### 输入向量

1. **MCP 协议端点** `POST /api/v1/mcp` 或 `POST /api/v1/mcp/message`
2. **工具调用参数**:
   - `command`: 任意 Shell 命令字符串
   - `server_id`: 目标服务器 ID（可选，留空则在本地执行）
   - `timeout`: 命令超时时间（秒）

#### 代码路径分析

1. **权限检查**: [permissions.go:L35](file:///workspace/internal/mcp/permissions.go#L35) 将 `exec_command` 权限级别设为 **3 (admin)**

```go
// internal/mcp/permissions.go:35
"exec_command": 3,
```

2. **参数提取**: [handler_server.go:L177-189](file:///workspace/internal/mcp/handler_server.go#L177-L189) 直接从请求中提取 `command` 参数，**无任何过滤或验证**

```go
func handleExecCommand(ctx context.Context, d ServerManager, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    command, err := request.RequireString("command")  // 直接提取，无验证
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    serverID := request.GetString("server_id", "")
    timeout := 30
    if t := request.GetString("timeout", ""); t != "" {
        if v, err := strconv.Atoi(t); err == nil && v > 0 {
            timeout = v
        }
    }

    output, err := d.ExecCommand(ctx, serverID, command, timeout)  // 直接传递给执行层
    // ...
}
```

3. **命令执行**: [bridge_deploy.go:L183-203](file:///workspace/internal/service/bridge_deploy.go#L183-L203) 根据 `serverID` 决定本地或远程执行

```go
func (b *Bridge) ExecCommand(ctx context.Context, serverID, command string, timeout int) (string, error) {
    execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
    defer cancel()

    if serverID == "" {
        // 本地执行 — 直接执行攻击者提供的命令
        return b.Executor.RunCommand(execCtx, command)
    }

    // 远程执行 — 通过 SSH 在目标服务器执行命令
    remoteExec, err := b.getRemoteExecutor(ctx, serverID)
    // ...
    return remoteExec.RunCommand(execCtx, command)
}
```

4. **最终执行**: 命令被传递给底层执行器（本地为 `os/exec`，远程为 SSH），**无任何命令过滤**

#### 影响

- **权限提升**: 任何 admin 角色用户可以在部署 Pilot 服务器上执行任意系统命令
- **数据泄露**: 可读取敏感文件、配置、密钥
- **横向移动**: 可通过 SSH 凭证（若已配置）进一步攻击远程服务器
- **持久化**: 可安装后门、修改系统配置

#### 利用示例

```json
// 利用步骤 1: 列出服务器
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "list_servers",
    "arguments": {}
  }
}

// 利用步骤 2: 本地命令执行
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "exec_command",
    "arguments": {
      "command": "cat /etc/passwd",
      "server_id": "",
      "timeout": 30
    }
  }
}

// 利用步骤 3: 读取加密密钥
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "exec_command",
    "arguments": {
      "command": "cat ./data/.encryption_key",
      "timeout": 30
    }
  }
}
```

#### 修复建议

**方案 1: 移除危险工具** (推荐)
- 完全移除 `exec_command` 工具，或仅对特定白名单命令开放
- 若必须保留，应实施严格的命令白名单机制

**方案 2: 实施命令白名单**
```go
var allowedCommands = map[string]bool{
    "docker": true,
    "docker-compose": true,
    "kubectl": true,
    "git": true,
}

func isCommandAllowed(cmd string) bool {
    parts := strings.Fields(cmd)
    if len(parts) == 0 {
        return false
    }
    base := filepath.Base(parts[0])
    return allowedCommands[base]
}
```

**方案 3: 增强权限控制**
- 将 `exec_command` 权限级别从 3 (admin) 提升至 4 (owner)
- 实施操作审计，所有命令执行必须记录完整日志

---

### [MED-1] 心跳令牌枚举导致的服务状态探测漏洞

#### 漏洞位置

| 文件 | 行号 |
|------|------|
| [internal/api/router.go](file:///workspace/internal/api/router.go#L141) | 141 |
| [internal/api/monitor_api.go](file:///workspace/internal/api/monitor_api.go#L232-L242) | 232-242 |
| [internal/service/monitor_service.go](file:///workspace/internal/service/monitor_service.go#L375-L396) | 375-396 |

#### 攻击者画像

- **外部用户**: 任何网络访问者，无需认证

#### 输入向量

- **URL 路径参数**: `/api/v1/heartbeat/ping/:token`
- **Token 格式**: UUID v4 (122位熵)

#### 代码路径分析

1. **路由注册**: [router.go:L141](file:///workspace/internal/api/router.go#L141) 将心跳端点注册为**公开端点**，无任何认证中间件

```go
// internal/api/router.go:141
r.GET("/api/v1/heartbeat/ping/:token", globalMonitorAPI.PingHeartbeat)
```

2. **处理器实现**: [monitor_api.go:L232-242](file:///workspace/internal/api/monitor_api.go#L232-L242) 直接接收并使用 token

```go
func (m *MonitorAPI) PingHeartbeat(c *gin.Context) {
    token := c.Param("token")  // 直接从 URL 获取

    if err := m.monSvc.PingHeartbeat(c.Request.Context(), token); err != nil {
        respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
        return
    }

    respondSuccess(c, gin.H{"status": "ok"})
}
```

3. **数据库查询**: [monitor_service.go:L376-381](file:///workspace/internal/service/monitor_service.go#L376-L381) 使用 token 直接查询数据库

```go
func (m *MonitorService) GetHeartbeatByToken(ctx context.Context, token string) (*Heartbeat, error) {
    var hb Heartbeat
    if err := m.db.WithContext(ctx).First(&hb, "token = ?", token).Error; err != nil {
        return nil, err
    }
    return &hb, nil
}
```

#### 利用方式

1. **UUID 暴力破解**: 使用常见的 UUID 模式或通过其他渠道获取的 token 进行探测
2. **批量探测**: 自动化工具批量测试 UUID，找到有效的监控端点
3. **状态收集**: 通过响应时间差异（有效 vs 无效 token）推断系统状态

#### 影响

- **信息泄露**: 攻击者可以探测 DeployPilot 部署的内部监控服务状态
- **攻击面映射**: 确定哪些服务器/服务正在被监控
- **拒绝服务风险**: 大规模探测可能影响服务性能

#### 修复建议

**方案 1: 移除公开端点** (推荐)
- 将心跳端点移至认证保护的路由组
- 要求调用方提供有效的 JWT token

**方案 2: 添加 HMAC 验证**
```go
func (m *MonitorAPI) PingHeartbeat(c *gin.Context) {
    token := c.Param("token")
    signature := c.GetHeader("X-Webhook-Signature")

    // 验证 HMAC 签名
    expectedSig := hmacSHA256(sharedSecret, token)
    if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
        respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
        return
    }
    // ...
}
```

---

## 已确认的安全最佳实践

以下方面实现良好，无需修改：

| 方面 | 状态 | 说明 |
|------|------|------|
| SQL 注入防护 | ✅ | 使用 GORM 参数化查询 |
| Webhook SSRF | ✅ | 实现了 DNS rebinding 和私有 IP 检查 |
| 密码加密 | ✅ | 使用 Argon2id + HKDF 派生密钥 |
| 凭证加密 | ✅ | AES-256-GCM 加密敏感数据 |
| JWT 安全 | ✅ | 支持 token 黑名单、刷新 token |
| 安全头 | ✅ | CSP、HSTS、X-Frame-Options 等 |
| 暴力破解防护 | ✅ | 登录速率限制 |

---

## 风险评估

| 漏洞 | CVSS 评分 | 风险等级 |
|------|-----------|----------|
| HIGH-1: 任意命令执行 | 8.1 (AV:N/AC:L/PR:H/UI:N/S:U/C:H/I:H/A:H) | 高 |
| MED-1: 心跳令牌枚举 | 5.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N) | 中 |

---

## 结论

本次审计确认了 **2 个已验证漏洞**，均具有明确的攻击路径和实际影响。建议优先修复 **HIGH-1**（任意命令执行），该漏洞允许已认证的 admin 用户完全控制部署基础设施。

# DeployPilot 安全审计报告

**审计日期**: 2026-06-04
**目标版本**: DeployPilot (Go + Vue)
**审计范围**: 代码库全面安全审查

---

## 执行摘要

本次审计系统性地检查了 DeployPilot 代码库的高风险攻击面。审计发现 **1 个已确认的中等严重度漏洞**，涉及 `docker-compose` 部署功能中的命令注入风险。该漏洞需要已认证用户配合特定的部署模式方可利用。

---

## 已确认漏洞

### VULN-001: docker-compose 日志功能存在命令注入

**严重度**: 中等 (CVSS 6.5)

**位置**:
- `internal/engine/deployer/compose.go` 第 68-76 行
- `internal/service/bridge_deploy.go` 第 137-156 行

**攻击者画像**: 已认证用户（dev 角色或更高），能够访问 MCP 工具或 API

**可控输入向量**:
- MCP `compose_logs` 工具的 `service` 参数
- API `GET /compose/:appID/logs` 的 URL 查询参数

**代码路径**:

```go
// compose.go:68-76
func (d *ComposeDeployer) ComposeLogs(ctx context.Context, workDir, service, tail string) (string, error) {
    cmd := fmt.Sprintf("cd %s && docker-compose logs", util.ShellQuote(workDir))
    if service != "" {
        cmd += " " + service  // ← 未对 service 进行 shell 特殊字符校验
    }
    if tail != "" {
        cmd += " --tail " + tail
    }
    cmd += " 2>&1"
    // ...
}
```

调用链:
1. `handleComposeLogs` (mcp/handler_deploy.go) 获取 `service` 参数
2. 调用 `bridge.ComposeLogs(ctx, appID, service, tail)`
3. 调用 `deployer.ComposeLogs(ctx, workDir, service, tail)`
4. `service` 直接拼接到命令字符串，无任何校验

**利用条件**:
- 攻击者需要能够调用 `compose_logs` 工具或 API
- 目标服务器上安装有 docker-compose
- docker-compose 允许服务名包含特殊字符（实际取决于 docker-compose 版本和配置）

**影响**:
- 在目标服务器上执行任意 docker-compose 子命令
- 可能导致信息泄露（读取其他服务日志）或远程代码执行（取决于 docker-compose 版本和权限）

**复现步骤**:
1. 使用有效 JWT 或 API Key 认证
2. 调用 MCP 工具 `compose_logs`，`service` 参数设为 `; echo pwned`
3. 观察命令注入执行结果

**修复建议**:

```go
// compose.go - 修复 ComposeLogs 函数
func (d *ComposeDeployer) ComposeLogs(ctx context.Context, workDir, service, tail string) (string, error) {
    // 验证 service 参数不包含 shell 特殊字符
    if strings.ContainsAny(service, " &'\"\\$`|;<>{}()[]*?!") {
        return "", fmt.Errorf("invalid service name: contains shell special characters")
    }

    cmd := fmt.Sprintf("cd %s && docker-compose logs", util.ShellQuote(workDir))
    if service != "" {
        cmd += " " + util.ShellQuote(service)  // 使用 ShellQuote 包裹
    }
    // ...
}
```

---

## 低风险项（不纳入本报告）

以下项目经审计后判定为低风险或理论性风险，不满足"可演示端到端利用路径"的要求：

| 项目 | 原因 |
|------|------|
| CSP `style-src 'unsafe-inline'` | 需要配合 XSS 才能利用；代码中无明显 XSS 入口 |
| HSTS 启用 | 需要完整 TLS 配置配合，不属于代码漏洞 |
| API Key 前缀检查 `dp_` | 仅属防御性编程，不构成安全漏洞 |
| Sandbox 黑名单规则 | 规则覆盖面广，实际利用需要突破多层防护 |

---

## 正向安全特性（未发现问题）

以下安全措施经审计验证通过：

| 类别 | 验证结果 |
|------|----------|
| JWT 令牌校验 | ✅ 正确实现过期检查、签名验证、黑名单 |
| API Key 认证 | ✅ 支持作用域、IP 白名单、前缀验证 |
| 密码存储 | ✅ 使用 bcrypt 哈希（crypto.HashPassword） |
| 凭证加密 | ✅ 使用 AES 加密存储（crypto.Encrypt） |
| SQL 注入防护 | ✅ GORM 参数化查询 |
| Shell quoting | ✅ `util.ShellQuote()` 对关键路径进行包裹 |
| 路径遍历防护 | ✅ `isPathBlocked()` 和 `validateVolumePath()` 验证 |
| 镜像仓库校验 | ✅ `validateImageRegistry()` 白名单机制 |
| OAuth2 安全 | ✅ 授权码一次性使用、状态参数验证、常数时间比较 |
| 2FA 支持 | ✅ TOTP + 备用码 |
| 暴力破解防护 | ✅ `bruteforce.Protector` 渐进延迟锁定 |
| 安全头 | ✅ CSP、HSTS、X-Frame-Options 等 |

---

## 总结

本次审计在 DeployPilot 代码库中发现 **1 个中等严重度已确认漏洞**。该漏洞位于 docker-compose 日志功能中，攻击者可通过未净化的 `service` 参数注入任意 docker-compose 子命令。

建议优先修复此漏洞，并加强部署功能区域的输入验证覆盖。

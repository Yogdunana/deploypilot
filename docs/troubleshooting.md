# 故障排查

## 常见错误与解决方案

### `crypto/aes: invalid key size 44`

**原因**：`DEPLOYPILOT_ENCRYPTION_KEY` 设置为 base64 字符串（如 `openssl rand -base64 32` 输出，长度 44），但服务端直接当作原始字节使用。

**解决**：升级到 v0.2.1+，服务端自动识别 base64 格式。无需修改密钥值。

```bash
# 确认当前密钥格式
echo -n "$DEPLOYPILOT_ENCRYPTION_KEY" | wc -c
# 44 = base64 编码（正确，服务端会自动解码）
# 32 = 原始字节（也支持，兼容历史方案）
```

### `database DSN must not be empty`

**原因**：未设置数据库路径，且 config 加载失败后使用了零值。

**解决**（任选一种）：
```bash
# 方式 1：使用默认路径（推荐）
./bin/mcp-server

# 方式 2：环境变量
export DEPLOYPILOT_DATABASE_DSN=./data/deploypilot.db
./bin/mcp-server

# 方式 3：命令行参数
./bin/mcp-server -db-dsn ./data/deploypilot.db
```

### SSH `connection refused`

**原因**：目标服务器 SSH 端口不可达。

**排查步骤**：
1. 确认服务器 IP 和端口正确（云厂商通常不是 22，如 23196、2222）
2. 检查云厂商安全组是否放行了对应端口的入站 TCP
3. 确认 sshd 正在监听：`ss -tlnp | grep sshd`
4. 手动测试：`ssh -p <端口> root@<IP>`
5. 使用 `test_server` MCP 工具获取详细建议

### SSH `auth failed`

**原因**：认证凭据不正确。

**排查步骤**：
1. 确认已通过 `add_credential` MCP 工具创建凭据
2. 确认凭据类型（password / ssh_key）与服务器配置匹配
3. 确认 `DEPLOYPILOT_ENCRYPTION_KEY` 与创建凭据时使用的密钥一致
4. 如果密钥丢失，已加密的凭据无法恢复，需要重新创建

### 凭据解密失败

**原因**：当前 `DEPLOYPILOT_ENCRYPTION_KEY` 与加密时使用的密钥不同。

**排查步骤**：
1. 确认环境变量一致：`echo $DEPLOYPILOT_ENCRYPTION_KEY`
2. 如果密钥被轮换，旧凭据需要重新创建
3. 建议将密钥持久化到安全位置（如 `.env` 文件或密钥管理服务）

### `DEPLOYPILOT_ENCRYPTION_KEY not set`

**原因**：未设置加密密钥，系统生成了临时密钥。

**影响**：重启后临时密钥丢失，已加密凭据无法解密。

**解决**：
```bash
# 生成持久化密钥（推荐 base64 格式）
export DEPLOYPILOT_ENCRYPTION_KEY=$(openssl rand -base64 32)
echo "保存此密钥到安全位置: $DEPLOYPILOT_ENCRYPTION_KEY"
```

### Docker `Cannot connect to the Docker daemon`

**原因**：Docker 服务未运行或当前用户无权限。

**解决**：
```bash
# 启动 Docker
sudo systemctl start docker

# 添加当前用户到 docker 组
sudo usermod -aG docker $USER
# 重新登录后生效
```

### `coverage below threshold`

**原因**：测试覆盖率低于 80%。

**解决**：运行 `go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1` 查看当前覆盖率，补充测试。

## 自检工具

运行 `doctor` MCP 工具可一键检查：
- Docker 是否可用
- 数据库连接是否正常
- SSH 执行器状态

```
→ 调用 doctor 工具
← 返回: {"status": "ok", "checks": [...], "tip": "..."}
```

## v0.2 最小可跑通示例

以下是通过 MCP 协议完成远端部署的完整流程：

```
1. add_server          → 注册服务器（host/port，默认端口 22）
2. add_credential      → 创建加密凭据（AES-256-GCM 存储）
3. update_server       → 关联凭据到服务器
4. deploy_app          → 传 server_id 触发远端 SSH + Docker 部署
5. get_deploy_status   → 查看部署状态
```

**CLI 录入凭据（不通过 Cursor 面板）**：
```bash
# 方式 1：管道输入
echo -n "my-secret-password" | deploypilot credential add --name prod-ssh --type password --value-stdin

# 方式 2：交互式隐藏输入
deploypilot credential add --name prod-ssh --type password

# 方式 3：从文件读取
deploypilot credential add --name prod-ssh --type password --value-file /path/to/secret
```

## 加密密钥格式

`DEPLOYPILOT_ENCRYPTION_KEY` 支持两种格式：

| 格式 | 长度 | 示例 | 推荐 |
|------|------|------|------|
| Base64（推荐） | 44 字符 | `openssl rand -base64 32` 输出 | ✅ |
| 原始 32 字节 | 32 字符 | 任意 32 字节 ASCII 字符串 | 兼容 |

```bash
# 推荐生成方式
export DEPLOYPILOT_ENCRYPTION_KEY=$(openssl rand -base64 32)
```

## 国内环境

如果 `go mod download` 或 `make build` 很慢，设置 Go 代理：
```bash
export GOPROXY=https://goproxy.cn,direct
```

# 故障排查

## 常见错误与解决方案

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
5. 使用 `test_server` 工具获取详细建议

### SSH `auth failed`

**原因**：认证凭据不正确。

**排查步骤**：
1. 确认已通过 `add_credential` 创建凭据
2. 确认凭据类型（password / ssh_key）与服务器配置匹配
3. 确认 `DEPLOYPILOT_ENCRYPTION_KEY` 与创建凭据时使用的密钥一致
4. 如果密钥丢失，已加密的凭据无法恢复，需要重新创建

### `DEPLOYPILOT_ENCRYPTION_KEY not set`

**原因**：未设置加密密钥，系统生成了临时密钥。

**影响**：重启后临时密钥丢失，已加密凭据无法解密。

**解决**：
```bash
# 生成持久化密钥
export DEPLOYPILOT_ENCRYPTION_KEY=$(python3 -c "import os,base64; print(base64.b64encode(os.urandom(32)).decode())")
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

## 国内环境

如果 `go mod download` 或 `make build` 很慢，设置 Go 代理：
```bash
export GOPROXY=https://goproxy.cn,direct
```

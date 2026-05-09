# Troubleshooting

## Common Issues

### MCP Server Not Connecting

**Symptoms:** AI IDE cannot connect to DeployPilot MCP server.

**Solutions:**
1. Verify the binary path is correct
2. Check `config.yaml` exists and is valid
3. Test independently: `./mcp-server --config config.yaml`
4. For remote connections, verify firewall allows port 8080
5. Check API token is valid (for HTTP transport)

### SSH Connection Refused

**Symptoms:** `connection refused` when deploying to a server.

**Solutions:**
1. Verify server IP and SSH port (cloud providers often use non-standard ports)
2. Check cloud security group allows inbound TCP on SSH port
3. Confirm sshd is running: `ss -tlnp | grep sshd`
4. Test manually: `ssh -p <port> root@<ip>`
5. Use `test_server` MCP tool for diagnostics

### OAuth Login Fails

**Symptoms:** GitHub/Gitee OAuth callback returns an error.

**Solutions:**
1. Verify OAuth provider is configured in `config.yaml`
2. Check callback URL matches DeployPilot address
3. Verify network can reach GitHub/Gitea API (may need proxy in enterprise)
4. Check server logs for OAuth error details

### Credential Decryption Fails

**Symptoms:** `decryption failed` error when using stored credentials.

**Solutions:**
1. Verify `DEPLOYPILOT_ENCRYPTION_KEY` matches the key used during encryption
2. If key was rotated, old credentials must be recreated
3. Persist the key to a secure location (`.env` file or secret manager)

### Docker Cannot Connect

**Symptoms:** `Cannot connect to the Docker daemon`.

**Solutions:**
```bash
sudo systemctl start docker
sudo usermod -aG docker $USER
# Log out and back in
```

### Token Revocation Not Working

**Symptoms:** Revoked JWT token still grants access.

**Solutions:**
1. Verify Redis is running and accessible
2. Check Redis configuration in `config.yaml`
3. Without Redis, memory-based blacklist is used (lost on restart)

### AES Invalid Key Size

**Symptoms:** `crypto/aes: invalid key size 44`.

**Solutions:**
- This is auto-detected since v0.2.1+
- Base64 keys (44 chars) are automatically decoded
- Raw 32-byte keys are also supported

## Self-Diagnosis

Run the `doctor` MCP tool to check system health:
- Docker availability
- Database connection
- SSH executor status

## Getting Help

- [GitHub Issues](https://github.com/Yogdunana/deploypilot/issues) — Bug reports and feature requests
- [Discussions](https://github.com/Yogdunana/deploypilot/discussions) — Questions and community support

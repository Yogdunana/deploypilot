# MCP Integration

## What is MCP?

The Model Context Protocol (MCP) is a standardized protocol that allows AI assistants to interact with external tools and services. DeployPilot exposes 52+ MCP tools for server deployment management.

## Supported AI IDEs

| IDE | Transport | Status |
|-----|-----------|--------|
| Claude Desktop | stdio | ✅ Supported |
| Cursor | stdio | ✅ Supported |
| TRAE | stdio | ✅ Supported |
| TRAE Solo | HTTP (remote) | ✅ Supported |
| Coze (扣子) | HTTP (remote) | ✅ Supported |
| SOLO | stdio | ✅ Supported |

## Local Setup (stdio transport)

For IDEs running on the same machine or with direct SSH access:

### 1. Install DeployPilot

```bash
# Download the MCP server binary
wget https://github.com/Yogdunana/deploypilot/releases/latest/download/mcp-server-linux-amd64
chmod +x mcp-server-linux-amd64
sudo mv mcp-server-linux-amd64 /usr/local/bin/deploypilot-mcp
```

### 2. Create Configuration

```yaml
# /etc/deploypilot/config.yaml
server:
  host: 0.0.0.0
  port: 8080

database:
  dsn: ./data/deploypilot.db
```

### 3. Configure IDE

Add to your IDE's MCP configuration:

```json
{
  "mcpServers": {
    "deploypilot": {
      "command": "/usr/local/bin/deploypilot-mcp",
      "args": ["--config", "/etc/deploypilot/config.yaml"]
    }
  }
}
```

## Remote Setup (HTTP transport)

For AI IDEs running in cloud sandboxes (TRAE Solo, Coze) that cannot SSH to your server:

### 1. Deploy DeployPilot on Your Server

```bash
docker compose up -d
```

### 2. Create API Token

1. Login to the web dashboard at `http://your-server:8080`
2. Go to Settings → API Tokens
3. Create a new token with appropriate permissions

### 3. Configure IDE

```json
{
  "mcpServers": {
    "deploypilot": {
      "url": "https://your-server:8080/api/v1/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_API_TOKEN"
      }
    }
  }
}
```

### Important Notes

- Use HTTPS in production
- Ensure firewall allows inbound connections on port 8080
- Create a dedicated API token with minimum required permissions

## Bastion/Jump Host Scenario

For enterprise servers behind a bastion host:

1. Deploy DeployPilot on the bastion host (or a server with network access to internal servers)
2. Configure SSH credentials that can reach internal servers through the bastion
3. AI IDEs connect to DeployPilot via MCP, and DeployPilot handles SSH tunneling internally

This eliminates the need for AI IDEs to directly connect to internal servers.

## Available Tools

### Deployment
- `deploy_app` — Deploy an application
- `get_deploy_status` — Check deployment status
- `rollback_app` — Rollback to previous version
- `batch_deploy` — Deploy multiple apps

### Application Management
- `list_apps` / `create_app` / `update_app` / `delete_app` / `get_app_detail`

### Server Management
- `list_servers` / `add_server` / `update_server` / `delete_server` / `test_server`

### DNS Management
- `list_dns_records` / `add_dns_record` / `update_dns_record` / `delete_dns_record`

### Monitoring
- `heal_container` / `get_container_metrics` / `get_system_metrics` / `list_alerts`

### And more...

See the full tool reference in the [repository docs](https://github.com/Yogdunana/deploypilot/blob/main/docs/mcp-tools.md).

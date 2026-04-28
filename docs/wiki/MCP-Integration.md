# MCP Integration

## What is MCP?

The Model Context Protocol (MCP) is a standardized protocol that allows AI assistants to interact with external tools and services. DeployPilot exposes 37 MCP tools for server deployment management.

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

### Deployment (5 tools)
| Tool | Description | RBAC |
|------|-------------|------|
| `deploy_app` | Deploy an application | dev |
| `get_deploy_status` | Check deployment status | viewer |
| `rollback_app` | Rollback to previous version | dev |
| `batch_deploy` | Deploy multiple apps at once | admin |
| `check_deploy_readiness` | Check if deployment prerequisites are met | viewer |

### Application Management (5 tools)
| Tool | Description | RBAC |
|------|-------------|------|
| `list_apps` | List all applications | viewer |
| `create_app` | Register a new application | dev |
| `update_app` | Update application configuration | dev |
| `delete_app` | Delete an application | admin |
| `get_app_detail` | Get detailed application info | viewer |

### Server Management (5 tools)
| Tool | Description | RBAC |
|------|-------------|------|
| `list_servers` | List all registered servers | viewer |
| `add_server` | Register a new server | dev |
| `update_server` | Update server configuration | dev |
| `delete_server` | Delete a server | admin |
| `test_server` | Test server SSH connectivity | dev |

### DNS Management (4 tools)
| Tool | Description | RBAC |
|------|-------------|------|
| `list_dns_records` | List DNS records | viewer |
| `add_dns_record` | Create a DNS record | dev |
| `update_dns_record` | Update a DNS record | dev |
| `delete_dns_record` | Delete a DNS record | admin |
| `batch_dns` | Batch DNS operations | admin |

### Credential Management (4 tools)
| Tool | Description | RBAC |
|------|-------------|------|
| `list_credentials` | List all credentials | viewer |
| `add_credential` | Add a new credential | dev |
| `update_credential` | Update a credential | dev |
| `delete_credential` | Delete a credential | admin |

### Logging (2 tools)
| Tool | Description | RBAC |
|------|-------------|------|
| `get_app_logs` | Get application logs | viewer |
| `search_app_logs` | Search application logs | viewer |

### Backup & Restore (3 tools)
| Tool | Description | RBAC |
|------|-------------|------|
| `backup_database` | Create a database backup | dev |
| `restore_database` | Restore from a backup | dev |
| `batch_backup` | Batch backup operations | admin |

### Task Management (2 tools)
| Tool | Description | RBAC |
|------|-------------|------|
| `get_task_status` | Get task execution status | viewer |
| `list_tasks` | List all tasks | viewer |

### System & Monitoring (4 tools)
| Tool | Description | RBAC |
|------|-------------|------|
| `check_system_update` | Check for system updates | viewer |
| `detect_environment` | Detect server environment info | viewer |
| `health_check` | HTTP/TCP health probe | viewer |
| `send_notification` | Send deployment notifications | dev |

### Templates (2 tools)
| Tool | Description | RBAC |
|------|-------------|------|
| `list_templates` | List available app templates (9 stacks) | viewer |
| `get_template` | Get template details | viewer |

See the full tool specification in [docs/mcp-tools.md](https://github.com/Yogdunana/deploypilot/blob/main/docs/mcp-tools.md).

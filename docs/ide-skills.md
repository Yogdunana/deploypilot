# IDE Skill Templates for DeployPilot

This document describes how to set up DeployPilot MCP integration with various IDEs that support the Model Context Protocol (MCP).

## What is DeployPilot MCP?

DeployPilot exposes 52 MCP tools that enable AI assistants to manage:
- Application deployment (Docker containers)
- DNS record management (Cloudflare, Alibaba Cloud, Tencent Cloud, WestDNS)
- Server infrastructure management
- Monitoring and self-healing
- CI/CD pipeline integration
- SSL certificate management
- Credential management (encrypted)
- Backup and restore operations

## Prerequisites

1. DeployPilot MCP server binary or source code
2. A valid `config.yaml` configuration file
3. An IDE that supports MCP (Claude Desktop, Cursor, TRAE, etc.)

## Claude Desktop Setup

### Configuration File
Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

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

### Skill File
Place the skill file at `.claude/skills/deploypilot.md` in your project root. Claude Desktop will automatically load it as context for the project.

## Cursor Setup

### Configuration File
In Cursor, open Settings > MCP and add:

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

### Rules File
Place the rules file at `.cursor/rules/deploypilot.mdc` in your project root. Cursor will apply these rules when working with DeployPilot-related code.

## TRAE Setup

### Configuration File
In TRAE IDE, configure MCP servers in the settings:

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

### Rules File
Place the rules file at `.trae/rules/deploypilot.md` in your project root.

## TRAE Solo Setup (Cloud Sandbox)

TRAE Solo runs in a cloud sandbox environment with no direct SSH access to servers. DeployPilot is specifically designed to solve this problem.

### How It Works

TRAE Solo's sandbox cannot SSH to your servers directly. DeployPilot bridges this gap:

1. DeployPilot runs on your server (self-hosted)
2. TRAE Solo connects to DeployPilot via MCP protocol over the network
3. DeployPilot executes deployment operations on the server locally

### Configuration

Since TRAE Solo runs in a sandbox, you need to use the network MCP transport instead of local stdio:

1. Deploy DeployPilot on your server first (see [Quick Start](../README.md))
2. Enable MCP remote access in DeployPilot config
3. In TRAE Solo, configure the MCP server endpoint

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

- Ensure your server's firewall allows inbound connections on port 8080 (or your configured port)
- Use HTTPS in production for secure communication
- Create a dedicated API token with appropriate permissions in DeployPilot's admin panel

## Coze (扣子) Setup

Coze (扣子) is ByteDance's AI bot platform. DeployPilot can be integrated as an MCP tool provider.

### Configuration

1. Deploy DeployPilot on your server
2. In Coze's plugin/MCP configuration, add the DeployPilot MCP server endpoint:

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

### Bastion/Jump Host Scenario

For enterprise servers behind a bastion host (跳板机):

1. Deploy DeployPilot on the bastion host or a server with network access to internal servers
2. Configure DeployPilot with SSH credentials that can reach internal servers through the bastion
3. Coze/TRAE Solo connects to DeployPilot via MCP, and DeployPilot handles the SSH tunneling internally

This eliminates the need for AI IDEs to directly connect to internal servers.

## Available Tools Reference

### Core Operations
| Tool | Description |
|------|-------------|
| `deploy_app` | Deploy an application to a server |
| `list_apps` | List all applications |
| `get_app_detail` | Get detailed app information |
| `create_app` | Create a new application |
| `delete_app` | Delete an application |
| `stop` | Stop a running container |
| `remove` | Remove a container |
| `get_container_logs` | Get container logs |
| `rollback` | Rollback to previous version |

### Server Management
| Tool | Description |
|------|-------------|
| `list_servers` | List all servers |
| `add_server` | Register a new server |
| `test_server` | Test connectivity |
| `detect_environment` | Detect server environment |

### DNS Management
| Tool | Description |
|------|-------------|
| `list_dns_records` | List DNS records |
| `add_dns_record` | Add a DNS record |
| `update_dns_record` | Update a DNS record |
| `delete_dns_record` | Delete a DNS record |

Supported DNS providers:
- **Cloudflare** (`dns-cloudflare`)
- **Alibaba Cloud DNS** (`dns-aliyun`)
- **Tencent Cloud DNSPod** (`dns-tencent`)
- **WestDNS / west.cn** (`dns-west-dns`)

### Monitoring
| Tool | Description |
|------|-------------|
| `heal_container` | Self-heal a container |
| `get_container_metrics` | Container resource metrics |
| `get_system_metrics` | System resource metrics |
| `list_alerts` | List active alerts |

### CI/CD
| Tool | Description |
|------|-------------|
| `trigger_ci_build` | Trigger a CI/CD build |
| `get_ci_build_status` | Get build status |

## Common Workflows

### Deploy a new application
1. `create_app` - Register the application
2. `deploy_app` - Deploy to a server
3. `check_deploy_status` - Monitor deployment progress
4. `get_container_logs` - Verify logs

### Set up DNS for a domain
1. `add_dns_record` - Create DNS record
2. `list_dns_records` - Verify the record was created
3. `health_check` - Verify the domain resolves correctly

### Monitor server health
1. `detect_environment` - Check server environment
2. `get_system_metrics` - Get resource usage
3. `list_alerts` - Check for active alerts
4. `heal_container` - Auto-fix issues if needed

### Backup and restore
1. `backup` - Create a backup
2. `list_tasks` - Monitor backup progress
3. `restore` - Restore from backup if needed

## File Structure

```
project-root/
  .claude/
    skills/
      deploypilot.md      # Claude Desktop skill file
  .cursor/
    rules/
      deploypilot.mdc     # Cursor rules file
  .trae/
    rules/
      deploypilot.md      # TRAE rules file
  docs/
    ide-skills.md         # This documentation file
```

## Troubleshooting

### MCP server not connecting
- Verify the MCP server binary path is correct
- Check that the config.yaml file exists and is valid
- Ensure the MCP server is running independently before connecting

### DNS operations failing
- Verify the DNS provider is configured in the DeployPilot database
- Check that API credentials are correct
- Ensure the domain is registered with the provider

### Permission errors
- Verify the user has sufficient permissions in DeployPilot
- Check that the MCP server process has access to the config file

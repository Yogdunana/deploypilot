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

# DeployPilot MCP Integration

## Overview
DeployPilot provides MCP (Model Context Protocol) tools for AI-powered deployment management. It enables AI assistants to manage application deployments, DNS records, server infrastructure, monitoring, and CI/CD pipelines through a standardized tool interface.

## Connection
Configure in TRAE IDE MCP settings:
```json
{
  "mcpServers": {
    "deploypilot": {
      "command": "/path/to/mcp-server",
      "args": ["--config", "/path/to/config.yaml"]
    }
  }
}
```

## Available Tools (52 total)

### Core Operations
| Tool | Description |
|------|-------------|
| `deploy_app` | Deploy an application to a server |
| `list_apps` | List all applications |
| `get_app_detail` | Get detailed information about an application |
| `create_app` | Create a new application |
| `delete_app` | Delete an application |
| `update_app` | Update application configuration |
| `stop` | Stop a running container |
| `remove` | Remove a container |
| `get_container_logs` | Get container logs |
| `get_container_status` | Get container status |
| `rollback` | Rollback to a previous image version |

### Server Management
| Tool | Description |
|------|-------------|
| `list_servers` | List all registered servers |
| `add_server` | Register a new server |
| `remove_server` | Remove a server |
| `update_server` | Update server configuration |
| `test_server` | Test server connectivity |
| `detect_environment` | Detect server environment (OS, Docker, ports, services) |

### Credential Management
| Tool | Description |
|------|-------------|
| `list_credentials` | List encrypted credentials |
| `add_credential` | Add a new credential |
| `delete_credential` | Delete a credential |
| `update_credential` | Update a credential |
| `create_credential_with_expiry` | Create credential with expiration |
| `rotate_credential` | Rotate credential value |

### DNS Management
| Tool | Description |
|------|-------------|
| `list_dns_records` | List DNS records for a domain |
| `add_dns_record` | Add a DNS record |
| `update_dns_record` | Update a DNS record |
| `delete_dns_record` | Delete a DNS record |
| `batch_dns` | Batch DNS operations |

### Monitoring & Self-Healing
| Tool | Description |
|------|-------------|
| `heal_container` | Trigger self-healing for a container |
| `get_container_metrics` | Get container resource metrics |
| `get_system_metrics` | Get system resource metrics |
| `list_alerts` | List active alerts |
| `list_alert_rules` | List alert rules |
| `health_check` | Perform health check on a target |

### CI/CD
| Tool | Description |
|------|-------------|
| `trigger_ci_build` | Trigger a CI/CD build |
| `get_ci_build_status` | Get CI/CD build status |

### Backup & Restore
| Tool | Description |
|------|-------------|
| `backup` | Create application backup |
| `restore` | Restore from backup |
| `batch_backup` | Batch backup operations |

### SSL Certificates
| Tool | Description |
|------|-------------|
| `list_ssl_certificates` | List SSL certificates |
| `request_ssl_certificate` | Request a new SSL certificate |
| `renew_ssl_certificate` | Renew an SSL certificate |
| `delete_ssl_certificate` | Delete an SSL certificate |

### Task Management
| Tool | Description |
|------|-------------|
| `get_task_status` | Get async task status |
| `list_tasks` | List tasks with filtering |

### Other
| Tool | Description |
|------|-------------|
| `list_templates` | List deployment templates |
| `get_template` | Get a specific template |
| `send_notification` | Send notifications |
| `check_deploy_readiness` | Check deployment prerequisites |
| `batch_deploy` | Batch deploy multiple apps |
| `check_system_update` | Check for system updates |
| `build_and_deploy` | Full build-and-deploy pipeline |

## Supported DNS Providers
- **Cloudflare** (`dns-cloudflare`): API Token + Account Email
- **Alibaba Cloud DNS** (`dns-aliyun`): AccessKey ID + AccessKey Secret
- **Tencent Cloud DNSPod** (`dns-tencent`): Secret ID + Secret Key
- **WestDNS / west.cn** (`dns-west-dns`): API Token (userid) + API Secret (userpwd)

## Common Workflows

### 1. Deploy a new application
```
create_app -> deploy_app -> check_deploy_status -> get_container_logs
```

### 2. DNS setup for a new domain
```
add_dns_record -> list_dns_records (verify) -> health_check
```

### 3. Server monitoring
```
detect_environment -> get_system_metrics -> list_alerts
```

### 4. Backup and restore
```
backup -> list_tasks (monitor progress) -> restore
```

### 5. CI/CD pipeline
```
trigger_ci_build -> get_ci_build_status -> deploy_app
```

### 6. Incident response
```
list_alerts -> get_container_metrics -> heal_container -> get_container_logs
```

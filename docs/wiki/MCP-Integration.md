# MCP Integration

DeployPilot implements the Model Context Protocol (MCP) to enable AI assistants to interact with the platform programmatically.

## Overview

The MCP server exposes 91+ tools across multiple categories:
- Application Management
- Server Management
- Deployment Operations
- Monitoring & Alerting
- CI/CD Integration
- Kubernetes Management
- SSL Certificate Management
- And more...

## Quick Start

### Starting the MCP Server

```bash
# Start the MCP server (runs on port 9090 by default)
./deploypilot mcp-server

# Or using Docker
docker run -p 9090:9090 yogdunana/deploypilot mcp-server
```

### Connecting Claude Desktop

Add to your Claude Desktop config:

```json
{
  "mcpServers": {
    "deploypilot": {
      "command": "./deploypilot",
      "args": ["mcp-server"],
      "env": {
        "DEPLOYPILOT_MCP_TOKEN": "your-api-key"
      }
    }
  }
}
```

## Available Tools

For a complete list of all 91+ MCP tools, see [MCP Tools Reference](../mcp-tools.md).

## Common Use Cases

### Deploy an Application
```
Use the deploy_app tool with your application configuration.
```

### Check System Health
```
Use the detect_environment and health_check tools.
```

### Monitor Deployments
```
Use get_deploy_status and list_deployments tools.
```

## Authentication

All MCP tools require authentication via:
- API Key (DEPLOYPILOT_API_KEY)
- JWT Token (DEPLOYPILOT_JWT_TOKEN)

See [Authentication Guide](./Authentication.md) for details.

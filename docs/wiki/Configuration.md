# Configuration

DeployPilot is configured via `config.yaml`. You can also override settings with environment variables.

## Configuration File

```yaml
# Server settings
server:
  host: 0.0.0.0
  port: 8080
  cors_allowed_origins:
    - "*"

# Database
database:
  dsn: ./data/deploypilot.db
  # For PostgreSQL:
  # dsn: host=localhost port=5432 user=deploypilot password=secret dbname=deploypilot sslmode=disable

# Redis (optional, for caching and event bus)
redis:
  addr: localhost:6379
  password: ""
  db: 0

# Authentication
auth:
  jwt_secret: ""           # Auto-generated if empty
  token_expiry: 24h        # JWT token expiration
  oauth_providers:         # Optional OAuth login
    - name: github
      client_id: ""
      client_secret: ""
      redirect_uri: ""
    - name: gitee
      client_id: ""
      client_secret: ""
      redirect_uri: ""

# Security
security:
  rate_limit_default: 100    # Requests per minute
  rate_limit_owner: 1000
  rate_limit_admin: 500
  rate_limit_dev: 200
  rate_limit_viewer: 100

# Audit logging
audit:
  external_log_path: ""     # Optional: path for external audit log file

# Logging
log:
  level: info               # debug, info, warn, error
  format: json              # json or text
```

## Environment Variables

All configuration values can be overridden with environment variables using the `DEPLOYPILOT_` prefix:

| Variable | Description | Example |
|----------|-------------|---------|
| `DEPLOYPILOT_SERVER_PORT` | Server port | `8080` |
| `DEPLOYPILOT_DATABASE_DSN` | Database connection string | `./data/deploypilot.db` |
| `DEPLOYPILOT_REDIS_ADDR` | Redis address | `localhost:6379` |
| `DEPLOYPILOT_AUTH_JWT_SECRET` | JWT signing secret | (auto-generated) |
| `DEPLOYPILOT_ENCRYPTION_KEY` | Credential encryption key | (see below) |
| `DEPLOYPILOT_LOG_LEVEL` | Log level | `info` |

## Encryption Key

The encryption key is used to encrypt stored credentials (SSH keys, passwords, API tokens).

```bash
# Generate a key (recommended: base64 format)
export DEPLOYPILOT_ENCRYPTION_KEY=$(openssl rand -base64 32)
```

**Important:** Save this key securely! If lost, all encrypted credentials become unrecoverable.

## Docker Compose

```yaml
version: "3.8"
services:
  deploypilot:
    image: ghcr.io/yogdunana/deploypilot:latest
    ports:
      - "8080:8080"
    environment:
      - DEPLOYPILOT_ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - DEPLOYPILOT_REDIS_ADDR=redis:6379
    volumes:
      - ./data:/app/data
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data

volumes:
  redis-data:
```

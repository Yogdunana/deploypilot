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
  type: sqlite                # sqlite or postgres
  dsn: ./data/deploypilot.db
  # For PostgreSQL:
  # type: postgres
  # dsn: host=localhost port=5432 user=deploypilot password=secret dbname=deploypilot sslmode=disable

# Redis (optional, for caching and event bus)
redis:
  addr: localhost:6379
  password: ""
  db: 0

# Authentication
auth:
  jwt_secret: ""              # REQUIRED: set a strong random string (min 16 chars)
  token_expire: 24h           # JWT token expiration
  # OAuth providers (optional)
  # oauth_providers:
  #   - provider: github
  #     client_id: ""
  #     client_secret: ""
  #     redirect_url: "http://localhost:8080/api/v1/auth/oauth/github/callback"

# Security
security:
  rate_limit_default: 100     # Requests per minute
  rate_limit_owner: 200
  rate_limit_admin: 150
  rate_limit_dev: 100
  rate_limit_viewer: 50

# Audit logging
audit:
  external_log_path: ""       # Optional: path for external audit log (JSON Lines)

# Logging
log:
  level: info                 # debug, info, warn, error
  format: json                # json or text
  file: ./logs/deploypilot.log
  max_size: 100MB
  max_backups: 10
  enable_tracing: true

# Deployment settings
deploy:
  default_mode: api           # api or docker
  build_timeout: 10m
  health_check_interval: 30s
  health_check_retries: 3
  rollback_on_failure: true

# Monitoring
monitor:
  enabled: true
  metrics_port: 9091

# Backup settings
backup:
  enabled: true
  interval: "6h"
  retention_count: 10
  retention_days: 30
  backup_dir: "data/backups"

# Brute-force protection
bruteforce:
  max_attempts: 5
  lockout_duration: "15m"
  window_duration: "15m"
  progressive_delay: true
  base_delay: "1s"
  max_delay: "30s"
  ip_max_attempts: 20
  ip_lockout_duration: "30m"
```

## Environment Variables

All configuration values can be overridden with environment variables using the `DEPLOYPILOT_` prefix:

| Variable | Description | Example |
|----------|-------------|---------|
| `DEPLOYPILOT_SERVER_PORT` | Server port | `8080` |
| `DEPLOYPILOT_DATABASE_DSN` | Database connection string | `./data/deploypilot.db` |
| `DEPLOYPILOT_DATABASE_TYPE` | Database type | `sqlite` or `postgres` |
| `DEPLOYPILOT_REDIS_ADDR` | Redis address | `localhost:6379` |
| `DEPLOYPILOT_REDIS_PASSWORD` | Redis password | (empty by default) |
| `DEPLOYPILOT_AUTH_JWT_SECRET` | JWT signing secret | (auto-generated if empty) |
| `DEPLOYPILOT_AUTH_TOKEN_EXPIRE` | JWT token expiration | `24h` |
| `DEPLOYPILOT_ENCRYPTION_KEY` | Credential encryption key | (see below) |
| `DEPLOYPILOT_LOG_LEVEL` | Log level | `info` |
| `DEPLOYPILOT_LOG_FILE` | Log file path | `./logs/deploypilot.log` |
| `DEPLOYPILOT_LOG_ENABLE_TRACING` | Enable request tracing | `true` |
| `DEPLOYPILOT_MONITOR_ENABLED` | Enable monitoring | `true` |
| `DEPLOYPILOT_MONITOR_METRICS_PORT` | Metrics endpoint port | `9091` |
| `DEPLOYPILOT_BACKUP_ENABLED` | Enable automatic backups | `true` |
| `DEPLOYPILOT_BACKUP_INTERVAL` | Backup interval | `6h` |
| `DEPLOYPILOT_BACKUP_RETENTION_COUNT` | Max backup files | `10` |
| `DEPLOYPILOT_BACKUP_RETENTION_DAYS` | Max backup age | `30` |
| `DEPLOYPILOT_BACKUP_BACKUP_DIR` | Backup directory | `data/backups` |

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
      - DEPLOYPILOT_AUTH_JWT_SECRET=${JWT_SECRET}
      - DEPLOYPILOT_REDIS_ADDR=redis:6379
      - DEPLOYPILOT_REDIS_PASSWORD=${REDIS_PASSWORD}
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

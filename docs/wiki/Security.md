# Security

DeployPilot implements multiple layers of security to protect your infrastructure.

## Authentication

### Password Hashing
- **argon2id** (default) — PHC format, memory-hard, resistant to GPU/ASIC attacks
- **bcrypt** (fallback) — Backward compatible with existing password hashes
- Automatic format detection — no migration needed

### OAuth Login
- GitHub and Gitee OAuth2 integration
- CSRF protection via state parameter
- Automatic user creation on first login

### JWT Tokens
- JSON Web Tokens for API authentication
- Configurable expiration (default: 24h)
- Unique token ID (jti) for revocation support

## Token Revocation
- Redis-based token blacklist
- Fail-open strategy (allows request if Redis is unavailable)
- Memory fallback when Redis is not configured

## Authorization

### Role-Based Access Control (RBAC)
Four roles with hierarchical permissions:

| Role | Scope |
|------|-------|
| **Owner** | Full access to all resources |
| **Admin** | Manage users, servers, apps |
| **Developer** | Deploy and manage apps |
| **Viewer** | Read-only access |

### Resource-Level Access
- Per-resource ownership checks
- Users can only access resources they own (unless Owner/Admin)
- Applied to apps, servers, and credentials

## Credential Security

### Encryption
- AES-256-GCM encryption for all stored credentials
- Base64 and raw key format support
- Encryption key managed via environment variable

### Key Management
```bash
# Generate encryption key
export DEPLOYPILOT_ENCRYPTION_KEY=$(openssl rand -base64 32)

# Store securely (e.g., in .env file)
echo "DEPLOYPILOT_ENCRYPTION_KEY=$DEPLOYPILOT_ENCRYPTION_KEY" >> .env
```

## Audit Logging

### Database Audit
- All API operations logged to database
- Includes user ID, action, resource, IP address, timestamp

### External Audit (Optional)
- JSON Lines format for easy parsing
- Supports file-based external logging
- Composite writer pattern for multiple outputs

## Rate Limiting
- Per-role rate limiting
- Configurable requests per minute
- Default: 100 req/min (viewer) to 1000 req/min (owner)

## Security Headers
- Content-Security-Policy
- X-Content-Type-Options: nosniff
- X-Frame-Options: DENY
- X-XSS-Protection: 1; mode=block

## WebSocket Security
- JWT authentication required
- One-time ticket exchange (ws-ticket)
- JWT never exposed in URL parameters

## CI/CD Security
- golangci-lint with gosec, staticcheck, noctx, bodyclose
- gitleaks secret scanning
- npm audit for frontend dependencies
- govulncheck for Go vulnerability scanning

## Best Practices

1. **Always set `DEPLOYPILOT_ENCRYPTION_KEY`** — Without it, a temporary key is generated and lost on restart
2. **Use HTTPS in production** — Deploy behind a reverse proxy with SSL
3. **Restrict CORS origins** — Don't use `*` in production
4. **Use strong JWT secrets** — Auto-generated if not set, but persist it for restarts
5. **Enable Redis** — For token revocation and event bus
6. **Regularly update** — Dependabot is configured for automatic dependency updates
7. **Review audit logs** — Monitor for suspicious activity

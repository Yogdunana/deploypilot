# Security Policy

## Supported Versions

| Version | Supported | Notes |
|---------|:---------:|-------|
| v1.2.x  | Yes       | Current release series |
| v1.1.x  | Yes       | Maintenance only |
| v1.0.x  | No        | End of life |
| < v1.0  | No        | End of life |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in DeployPilot, please report it responsibly.

### How to Report

1. **Email**: Send a report to the project maintainers via [GitHub Security Advisories](https://github.com/Yogdunana/deploypilot/security/advisories/new).
2. **Do NOT** open a public issue for security vulnerabilities.

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested fixes (optional)

### Response Timeline

- **Acknowledgment**: Within 48 hours of receiving a report
- **Initial Assessment**: Within 72 hours
- **Patch Release**: As soon as a fix is available, typically within 7 days for critical issues

### Coordination

We will work with you to understand and resolve the issue. Security fixes will be backported to all supported versions.

## Security Features

DeployPilot includes the following security features:

| Feature | Description |
|---------|-------------|
| **AES-256-GCM Encryption** | All credentials (SSH keys, passwords, API tokens) are encrypted at rest |
| **Argon2id Hashing** | Passwords are hashed using Argon2id with configurable parameters |
| **JWT Authentication** | JSON Web Token-based authentication with configurable expiration |
| **RBAC** | Role-based access control with 4 roles: owner, admin, dev, viewer |
| **Brute-Force Protection** | Progressive delay, account lockout, and IP-based rate limiting |
| **Audit Logging** | Comprehensive audit trail for all sensitive operations |
| **Rate Limiting** | Per-role rate limiting on all API endpoints |
| **Request Tracing** | Distributed tracing support for request debugging |
| **CSRF Protection** | OAuth flows include state parameter validation |
| **Secret Scanning** | CI pipeline includes automated secret detection |

For detailed security architecture, see [docs/wiki/Security.md](docs/wiki/Security.md).

## Security Best Practices for Deployment

1. **Set a strong JWT secret**: Use `DEPLOYPILOT_AUTH_JWT_SECRET` with at least 16 random characters
2. **Set an encryption key**: Use `DEPLOYPILOT_ENCRYPTION_KEY` generated via `openssl rand -base64 32`
3. **Enable HTTPS**: Use a reverse proxy (Nginx/Caddy) with TLS in production
4. **Restrict CORS**: Set `server.cors_allowed_origins` to specific domains instead of `*`
5. **Use firewall rules**: Only expose necessary ports (8080 for API, 9091 for metrics)
6. **Rotate credentials regularly**: Update SSH keys and API tokens periodically
7. **Enable audit logging**: Configure `audit.external_log_path` for persistent audit records

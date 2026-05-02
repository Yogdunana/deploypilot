# Phase 7.4: API Open Platform Design

> Date: 2026-05-02
> Status: Approved

## Overview

Transform DeployPilot's API into an enterprise-grade open platform with fine-grained scope permissions, enhanced Swagger documentation, distributed rate limiting, and OAuth2 application registration supporting both Client Credentials and Authorization Code flows.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                  API Gateway Layer                   │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐  │
│  │ API Key  │  │  OAuth2  │  │   JWT (existing)  │  │
│  │ + Scopes │  │ CC+AuthC │  │   User auth       │  │
│  └────┬─────┘  └────┬─────┘  └────────┬──────────┘  │
│       └──────────────┼────────────────┘              │
│                      ▼                               │
│  ┌───────────────────────────────────────────────┐   │
│  │    Rate Limiter (Per-Key/Per-Endpoint)        │   │
│  │    Redis-backed + Memory fallback             │   │
│  └───────────────────────────────────────────────┘   │
│                      ▼                               │
│  ┌───────────────────────────────────────────────┐   │
│  │    Scope Middleware (RequireScope)             │   │
│  └───────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

## Feature Modules

### 1. Scope Permission System

**Standard Scope Enum:**

| Scope | Description |
|-------|-------------|
| `read` | Read all resources |
| `write` | Create/update resources |
| `delete` | Delete resources |
| `deploy` | Deploy/rollback operations |
| `admin` | Admin super-scope (has all permissions) |
| `monitor:read` | View monitoring data |
| `monitor:write` | Manage monitoring config |
| `server:read` | View server info |
| `server:exec` | Execute remote commands |
| `credential:read` | View credentials |
| `credential:write` | Manage credentials |
| `dns:write` | DNS record management |
| `ssl:write` | SSL certificate management |
| `backup:read` | View backups |
| `backup:write` | Execute backup/restore |
| `webhook:manage` | Manage webhooks |
| `grafana:manage` | Manage Grafana integration |

**RequireScope Middleware:**
- `auth.RequireScope(scopes ...string)` — checks if current API Key or OAuth2 token has any of the required scopes
- `admin` scope bypasses all checks
- Returns 403 with descriptive error if scope insufficient
- Sets scope info in context for audit logging

**API Key Scope Enhancement:**
- Validate scopes against the standard enum on creation
- Add scope descriptions to API responses
- Backward compatible: existing keys with custom scopes continue to work

### 2. Swagger Documentation Enhancement

**Security Definitions:**
```json
{
  "BearerAuth": { "type": "apiKey", "in": "header", "name": "Authorization" },
  "APIKeyAuth": { "type": "apiKey", "in": "header", "name": "X-API-Key" },
  "OAuth2AccessCode": {
    "type": "oauth2",
    "flow": "accessCode",
    "authorizationUrl": "/api/v1/oauth/authorize",
    "tokenUrl": "/api/v1/oauth/token",
    "scopes": { ... }
  }
}
```

**Annotations:** Add `@Summary`, `@Description`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Router`, `@Security` to all existing API handlers.

**Developer Docs Page:** Frontend page at `/docs` embedding Swagger UI + API usage guide.

### 3. Rate Limiting Enhancement

**Per-Key Rate Limiting:**
- Each API Key gets its own rate limit bucket
- Default: inherit from role-based limits
- Custom: user can set per-key rate limit (min 10/min, max 1000/min)
- Stored in `APIKey.RateLimit` field (new column)

**Per-Endpoint Rate Limiting:**
- High-cost endpoints get lower limits:
  - Deploy: 20/min
  - Backup: 10/min
  - SSL: 10/min
  - Default: role-based limit

**Redis Distributed:**
- Use Redis INCR + EXPIRE for distributed rate limiting
- Fallback to in-memory when Redis unavailable
- Key format: `ratelimit:{type}:{id}:{endpoint}`

**Burst Control:**
- Allow burst of 2x the rate limit for 5 seconds
- Use token bucket algorithm with burst parameter

### 4. OAuth2 Application Registration

**Grant Types:**

**Client Credentials (machine-to-machine):**
```
POST /api/v1/oauth/token
  grant_type=client_credentials
  client_id=xxx
  client_secret=xxx
  scope=read,monitor:read
→ { access_token, token_type, expires_in, scope }
```

**Authorization Code (user delegation):**
```
1. User visits: /api/v1/oauth/authorize?client_id=xxx&redirect_uri=xxx&scope=read&response_type=code
2. User sees consent page → approves
3. Redirect: {redirect_uri}?code=xxx
4. App exchanges code: POST /api/v1/oauth/token (grant_type=authorization_code, code=xxx, ...)
5. Response: { access_token, refresh_token, token_type, expires_in, scope }
```

**Data Models:**

```go
type OAuth2Client struct {
    ID           string    `json:"id"`
    TenantID     string    `json:"tenant_id"`
    UserID       string    `json:"user_id"`
    Name         string    `json:"name"`
    ClientID     string    `json:"client_id" gorm:"uniqueIndex"`
    ClientSecret string    `json:"-" gorm:"not null"`
    RedirectURIs string    `json:"redirect_uris"`  // JSON array
    Scopes       string    `json:"scopes"`         // JSON array
    GrantTypes   string    `json:"grant_types"`    // JSON array: ["client_credentials", "authorization_code"]
    Enabled      bool      `json:"enabled"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

type OAuth2Authorization struct {
    ID        string     `json:"id"`
    ClientID  string     `json:"client_id"`
    UserID    string     `json:"user_id"`
    Scopes    string     `json:"scopes"`
    Code      string     `json:"code" gorm:"uniqueIndex"`
    ExpiresAt time.Time  `json:"expires_at"`
    Used      bool       `json:"used"`
    CreatedAt time.Time  `json:"created_at"`
}

type OAuth2Token struct {
    ID           string     `json:"id"`
    ClientID     string     `json:"client_id"`
    UserID       string     `json:"user_id,omitempty"`
    AccessToken  string     `json:"access_token" gorm:"uniqueIndex"`
    RefreshToken string     `json:"refresh_token,omitempty" gorm:"uniqueIndex"`
    Scopes       string     `json:"scopes"`
    TokenType    string     `json:"token_type"`
    ExpiresAt    time.Time  `json:"expires_at"`
    CreatedAt    time.Time  `json:"created_at"`
}
```

**API Endpoints:**

```
# OAuth2
POST   /api/v1/oauth/authorize          # Start authorization (redirect user to consent page)
GET    /api/v1/oauth/consent            # Consent page (frontend)
POST   /api/v1/oauth/consent            # User approves/rejects authorization
POST   /api/v1/oauth/token              # Exchange code or client_credentials for token
POST   /api/v1/oauth/token/refresh       # Refresh access token
POST   /api/v1/oauth/token/revoke        # Revoke token

# OAuth2 Client Management
GET    /api/v1/oauth/clients            # List user's OAuth2 apps
POST   /api/v1/oauth/clients            # Register new OAuth2 app
GET    /api/v1/oauth/clients/:id        # Get app details
PUT    /api/v1/oauth/clients/:id        # Update app
DELETE /api/v1/oauth/clients/:id        # Delete app
POST   /api/v1/oauth/clients/:id/secret # Regenerate client secret
```

**Frontend Pages:**
- OAuth2 Apps management page (CRUD)
- OAuth2 consent page (user authorization)
- Developer documentation page

### 5. Config Changes

```go
type APIPlatformConfig struct {
    Enabled           bool `mapstructure:"enabled"`
    MaxClientsPerUser int  `mapstructure:"max_clients_per_user"` // default: 10
    TokenExpireHours  int  `mapstructure:"token_expire_hours"`   // default: 24
    CodeExpireMinutes int  `mapstructure:"code_expire_minutes"`  // default: 10
}
```

## File Structure

```
internal/
├── config/config.go                    # Add APIPlatformConfig
├── model/oauth2.go                     # OAuth2Client, OAuth2Authorization, OAuth2Token
├── auth/
│   ├── scopes.go                       # Scope constants + RequireScope middleware
│   └── oauth2_middleware.go            # OAuth2 Bearer token middleware
├── service/
│   ├── oauth2_service.go               # OAuth2 business logic
│   └── ratelimit_redis.go              # Redis-backed rate limiter
├── api/
│   ├── oauth2_api.go                   # OAuth2 handlers
│   ├── oauth2_clients_api.go           # Client management handlers
│   └── router.go                       # Add OAuth2 routes + scope middleware
web/src/
├── api/modules/oauth2.ts               # OAuth2 API module
├── views/
│   ├── OAuth2Apps.vue                  # App management page
│   └── OAuth2Consent.vue               # Consent page
└── router/index.ts                     # Add routes
```

## Dependencies

- No new Go dependencies (use crypto/rand, crypto/sha256 for OAuth2)
- Frontend: existing dependencies sufficient

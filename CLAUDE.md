# DeployPilot - AI Assistant Guide

This file provides context for AI coding assistants (Claude, Copilot, Cursor, etc.) working on the DeployPilot codebase.

## Project Overview

DeployPilot is an AI-native deployment platform for managing application deployments across servers. It provides a REST API, CLI tool, MCP server for AI integration, and a web dashboard.

## Tech Stack

### Backend
- **Language**: Go 1.23+
- **Web Framework**: Gin (`github.com/gin-gonic/gin`)
- **ORM**: GORM (`gorm.io/gorm`)
- **Database**: PostgreSQL (production), SQLite (development/testing)
- **Cache/PubSub**: Redis (`github.com/redis/go-redis/v9`)
- **CLI Framework**: Cobra (`github.com/spf13/cobra`)
- **MCP SDK**: mcp-go (`github.com/mark3labs/mcp-go`)
- **Configuration**: Viper (`github.com/spf13/viper`)
- **API Docs**: Swaggo (`github.com/swaggo/swag`)

### Frontend
- **Framework**: Vue 3 (Composition API with `<script setup>`)
- **Build Tool**: Vite
- **Language**: TypeScript
- **UI Components**: Custom components in `web/src/components/ui/`
- **State Management**: Pinia
- **Routing**: Vue Router
- **i18n**: vue-i18n (English + Chinese)

## Code Conventions

### Go
- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `log/slog` for structured logging (never `fmt.Println` for logging)
- All public functions must have doc comments
- Use `t.Helper()` in test helper functions
- Error handling: always check errors, use `fmt.Errorf("context: %w", err)` for wrapping
- Use table-driven tests where appropriate
- Database queries use GORM; raw SQL only when necessary (migrations)

### Frontend
- Use `<script setup lang="ts">` for all Vue components
- Use the shared UI components from `web/src/components/ui/`
- API calls go through `web/src/api/modules/`
- Add i18n keys to both `web/src/i18n/locales/en.ts` and `zh.ts`

### Commit Messages
Follow [Conventional Commits](https://www.conventionalcommits.org/):
```
<type>(<scope>): <description>
```
Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `security`

## Project Structure

```
cmd/api-server/     -> REST API entry point (main.go)
cmd/deploypilot/    -> CLI tool entry point
cmd/mcp-server/     -> MCP server entry point
internal/api/       -> HTTP handlers, routes (router.go defines all routes)
internal/auth/      -> JWT, OAuth, 2FA, middleware
internal/config/    -> Config loading (config.yaml, env vars)
internal/database/  -> DB connection, migrations (gormigrate), seed data
internal/engine/    -> Deployment engine (builder, deployer, healer)
internal/model/     -> GORM models (one file, all models)
internal/service/   -> Business logic layer
internal/provider/  -> External integrations (SSH, K8s, DNS, SSL, etc.)
internal/mcp/       -> MCP tool handlers
internal/middleware/ -> HTTP middleware (CORS, rate limit, security)
web/src/            -> Vue 3 frontend
```

## Common Commands

```bash
make dev-up          # Start PostgreSQL + Redis (Docker)
make dev-down        # Stop dev services
make dev-backend     # Run backend with hot reload (Air)
make dev-frontend    # Run frontend dev server (Vite)
make build-all       # Build all binaries
make test            # Run all tests
make lint            # Run golangci-lint
make check           # vet + lint + test
make swagger         # Regenerate Swagger docs
go run scripts/seed.go  # Seed demo data
```

## Database

- Migrations are in `internal/database/database.go` using gormigrate
- Each migration has a unique ID (format: `YYYYMMDDNNNN`)
- Migrations are idempotent (safe to run multiple times)
- Seed data creates default roles, tenant, and admin user
- Models are in `internal/model/model.go`

## API Structure

- All routes defined in `internal/api/router.go`
- API version prefix: `/api/v1`
- Authentication: JWT Bearer tokens
- Swagger docs available at `/swagger/index.html`
- WebSocket endpoints for real-time updates (monitoring, deployments)

## Testing

- Backend tests use Go's standard `testing` package
- Run with `make test` (includes race detector)
- Coverage report: `make coverage`
- E2E tests in `tests/e2e/`
- Frontend tests use Vitest (`web/vitest.config.ts`)

## Configuration

- Primary config: `configs/config.yaml` (loaded via Viper)
- Environment variables override config values (prefix: `DEPLOYPILOT_`)
- Example config: `.env.example`
- Dev config: `.env.dev`

## Key Patterns

### Adding a new API endpoint
1. Define handler in `internal/api/<feature>_api.go`
2. Register route in `internal/api/router.go`
3. Add business logic in `internal/service/<feature>_service.go`
4. Add/update model in `internal/model/model.go` if needed
5. Add migration in `internal/database/database.go` if schema changes
6. Update Swagger comments on handler
7. Add frontend API module in `web/src/api/modules/<feature>.ts`
8. Add i18n keys

### Adding a new MCP tool
1. Create handler in `internal/mcp/handler_<feature>.go`
2. Register in `internal/mcp/server.go`
3. Add permission check in `internal/mcp/permissions.go`

### Adding a new provider integration
1. Create provider in `internal/provider/<category>/<name>.go`
2. Implement the provider interface
3. Register in the service layer

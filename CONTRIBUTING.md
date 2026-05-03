# Contributing to DeployPilot

Thank you for your interest in contributing to DeployPilot! This guide will help you get started.

## Prerequisites

- **Go** 1.23+
- **Node.js** 20+ (with npm)
- **Docker** 20.10+ (for local PostgreSQL and Redis)
- **Make** (for build commands)
- **golangci-lint** (for linting)
- **Air** (optional, for Go hot reload: `go install github.com/air-verse/air@latest`)

## Development Setup

### 1. Clone the repository

```bash
git clone https://github.com/Yogdunana/deploypilot.git
cd deploypilot
```

### 2. Install dependencies

```bash
# Go dependencies
go mod download

# Frontend dependencies
cd web && npm ci && cd ..
```

### 3. Start development services

```bash
# Start PostgreSQL and Redis via Docker
make dev-up

# (Optional) Seed demo data
go run scripts/seed.go
```

### 4. Run the backend

```bash
# Option A: Build and run manually
make build-api
DEPLOYPILOT_ENCRYPTION_KEY=$(openssl rand -base64 32) ./bin/api-server \
  --db-driver postgres \
  --db-dsn "host=localhost port=5432 user=deploypilot password=deploypilot_dev dbname=deploypilot sslmode=disable"

# Option B: Hot reload with Air (recommended for development)
make dev-backend
```

### 5. Run the frontend

```bash
# Start Vite dev server with HMR
make dev-frontend
```

### 6. Quick start (all at once)

```bash
make dev
# Then in separate terminals:
#   make dev-backend
#   make dev-frontend
```

### VS Code Debugging

A launch configuration is provided at `.vscode/launch.json`. Open the project in VS Code, set breakpoints, and press F5 to debug the API server.

## Project Structure

```
deploypilot/
├── cmd/                    # Entry points
│   ├── api-server/         # REST API server (Gin)
│   ├── deploypilot/        # CLI tool (Cobra)
│   └── mcp-server/         # MCP server
├── internal/               # Private application code
│   ├── api/                # HTTP handlers and routes
│   ├── auth/               # Authentication (JWT, OAuth, 2FA)
│   ├── backup/             # Backup service
│   ├── config/             # Configuration management
│   ├── crypto/             # Encryption utilities
│   ├── database/           # Database connection and migrations
│   ├── engine/             # Deployment engine
│   │   ├── builder/        # Image builder
│   │   ├── deployer/       # Container deployer
│   │   ├── detector/       # Project type detector
│   │   └── healer/         # Self-healing
│   ├── i18n/               # Internationalization
│   ├── mcp/                # MCP tool handlers
│   ├── middleware/          # HTTP middleware
│   ├── model/              # Data models (GORM)
│   ├── monitor/            # Monitoring system
│   ├── plugin/             # Plugin system
│   ├── provider/           # External integrations
│   │   ├── cicd/           # CI/CD providers
│   │   ├── dns/            # DNS providers
│   │   ├── notify/         # Notification providers
│   │   ├── registry/       # Container registries
│   │   ├── server/         # Server providers (SSH, K8s, panels)
│   │   └── ssl/            # SSL providers
│   ├── service/            # Business logic layer
│   └── version/            # Version info
├── web/                    # Vue 3 frontend
│   └── src/
│       ├── api/            # API client modules
│       ├── components/     # Vue components
│       ├── composables/    # Vue composables
│       ├── i18n/           # Frontend i18n
│       ├── router/         # Vue Router
│       ├── stores/         # Pinia stores
│       ├── types/          # TypeScript types
│       └── views/          # Page components
├── scripts/                # Utility scripts
├── docs/                   # Documentation
│   ├── architecture/       # Architecture Decision Records
│   ├── swagger/            # API documentation
│   └── wiki/               # Wiki pages
├── configs/                # Configuration examples
├── .github/                # GitHub templates and workflows
├── docker-compose.dev.yml  # Development services
├── docker-compose.yml      # Production deployment
├── Makefile                # Build and dev commands
└── go.mod                  # Go module definition
```

## Code Standards

### Go

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Run `make lint` before committing (golangci-lint with gosec, staticcheck, etc.)
- All public functions must have doc comments
- Write tests for new code -- aim for >85% coverage
- Use `t.Helper()` in test helper functions
- Use structured logging via `log/slog`

### Frontend (Vue 3 + TypeScript)

- Use Composition API with `<script setup>`
- Follow existing component patterns (see `web/src/components/`)
- Use the shared UI components in `web/src/components/ui/`
- Support i18n -- add keys to both `en.ts` and `zh.ts`
- Use TypeScript strict mode

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types:
- `feat:` New features
- `fix:` Bug fixes
- `docs:` Documentation changes
- `refactor:` Code refactoring
- `test:` Test additions/changes
- `chore:` Build/config changes
- `security:` Security fixes

Examples:
```
feat(monitor): add CPU alert rule configuration
fix(deploy): resolve race condition in concurrent deployments
docs: update CONTRIBUTING.md with dev setup instructions
```

## Pull Request Process

1. **Fork** the repository
2. **Create a feature branch**: `git checkout -b feat/your-feature`
3. **Make your changes** with tests
4. **Run checks**: `make check` (runs vet, lint, and tests)
5. **Commit** with conventional commit messages
6. **Push** to your fork
7. **Open a Pull Request** using the provided template
8. **Ensure CI passes** on your PR
9. **Address review feedback** promptly

## Reporting Issues

Please use [GitHub Issues](https://github.com/Yogdunana/deploypilot/issues) to report bugs or request features. Use the provided issue templates for better triage.

### Bug Reports

Use the **Bug Report** template and include:
- Clear description of the bug
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs or error output
- DeployPilot version and environment details

### Feature Requests

Use the **Feature Request** template and include:
- Problem or motivation
- Proposed solution
- Area of the project affected
- Any alternatives considered

## Useful Commands

| Command | Description |
|---------|-------------|
| `make dev-up` | Start PostgreSQL and Redis |
| `make dev-down` | Stop dev services |
| `make dev-backend` | Run backend with hot reload |
| `make dev-frontend` | Run frontend dev server |
| `make build-all` | Build all binaries |
| `make test` | Run all tests |
| `make lint` | Run linter |
| `make check` | Run vet + lint + tests |
| `make swagger` | Regenerate API docs |
| `make dev-clean` | Remove dev volumes and data |

## License

By contributing, you agree that your contributions will be licensed under the [Business Source License 1.1 (BSL 1.1)](LICENSE). Non-commercial use (personal, internal organizational, testing, and non-commercial educational) is permitted. On **2029-04-28**, the license automatically converts to MIT.

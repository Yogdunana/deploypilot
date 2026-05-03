# ADR-001: Project Structure

## Title

Monorepo structure with Go backend and Vue 3 frontend

## Status

Accepted

## Context

DeployPilot is an AI-native deployment platform that requires:

1. A **REST API server** for the web dashboard and third-party integrations
2. A **CLI tool** for terminal-based server management
3. An **MCP server** for AI assistant integration
4. A **web dashboard** (Vue 3 SPA) for visual management
5. A **plugin system** for extensibility

We need a project structure that:
- Keeps all components in a single repository for easier development and release management
- Clearly separates concerns between backend and frontend
- Supports independent build pipelines for each component
- Allows shared documentation and configuration

## Decision

We adopt a **monorepo** structure with the following layout:

```
deploypilot/
├── cmd/                    # Entry points (one directory per binary)
│   ├── api-server/         # REST API server (Gin framework)
│   ├── deploypilot/        # CLI tool (Cobra framework)
│   └── mcp-server/         # MCP server (mcp-go SDK)
├── internal/               # Private Go packages (not importable by external modules)
│   ├── api/                # HTTP handlers, routes, middleware
│   ├── auth/               # JWT, OAuth, 2FA authentication
│   ├── config/             # Configuration loading (Viper)
│   ├── database/           # GORM database layer, migrations
│   ├── engine/             # Deployment engine (builder, deployer, healer)
│   ├── model/              # Data models
│   ├── provider/           # External service integrations
│   ├── service/            # Business logic
│   └── ...
├── web/                    # Vue 3 frontend (Vite + TypeScript)
│   └── src/
│       ├── api/            # API client
│       ├── components/     # Vue components
│       ├── views/          # Page components
│       └── ...
├── scripts/                # Utility scripts (build, install, seed)
├── docs/                   # Documentation (ADR, Swagger, Wiki)
├── configs/                # Configuration examples
├── pkg/                    # Public Go packages (importable by external modules)
├── Makefile                # Build, test, and dev commands
├── docker-compose.yml      # Production deployment
├── docker-compose.dev.yml  # Development services
└── go.mod                  # Go module definition
```

Key conventions:
- **`cmd/`** contains exactly one `main.go` per subdirectory, one per distributable binary
- **`internal/`** contains all private Go code, organized by domain
- **`web/`** is the self-contained Vue 3 frontend with its own `package.json`
- **`pkg/`** is reserved for public Go packages that may be imported by external projects
- **`scripts/`** contains operational scripts (build, install, seed data)
- **`docs/`** contains all documentation including ADRs and API specs

## Consequences

### Positive

- **Simplified development**: All code in one repo, single `git clone` to get started
- **Atomic commits**: Changes across backend and frontend can be committed together
- **Shared tooling**: Single Makefile, CI pipeline, and linting configuration
- **Clear separation**: `internal/` prevents accidental external imports; `cmd/` makes entry points obvious
- **Easy onboarding**: New contributors understand the structure from directory names

### Negative

- **Larger repository**: Frontend `node_modules` and Go module cache can be large
- **Slower CI**: All components are tested on every push (mitigated by path-based CI triggers)
- **Coupled releases**: Backend and frontend share the same versioning (acceptable for this project scale)

### Risks

- Repository size growth over time -- mitigated by `.gitignore` and regular cleanup
- Merge conflicts on shared files (Makefile, go.mod) -- mitigated by clear ownership conventions

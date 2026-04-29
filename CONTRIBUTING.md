# Contributing to DeployPilot

Thank you for your interest in contributing to DeployPilot! This guide will help you get started.

## Development Environment

### Prerequisites
- Go 1.23+
- Node.js 20+
- Docker 20.10+ (for testing)
- Make
- golangci-lint

### Setup

```bash
# Clone the repository
git clone https://github.com/Yogdunana/deploypilot.git
cd deploypilot

# Install Go dependencies
go mod download

# Install frontend dependencies
cd web &amp;&amp; npm ci &amp;&amp; cd ..

# Build all binaries
make build-all

# Run tests
make test
```

### Running Locally

```bash
# Start API server (with embedded web dashboard)
make build-api
DEPLOYPILOT_ENCRYPTION_KEY=$(openssl rand -base64 32) ./bin/api-server

# Start MCP server
make build-mcp
DEPLOYPILOT_ENCRYPTION_KEY=$(openssl rand -base64 32) ./bin/mcp-server
```

### Frontend Development

```bash
cd web
npm run dev  # Start Vite dev server with HMR
```

## Code Standards

### Go
- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Run `make lint` before committing (golangci-lint with gosec, staticcheck, etc.)
- All public functions must have doc comments
- Write tests for new code — aim for &gt;85% coverage
- Use `t.Helper()` in test helper functions

### Frontend (Vue 3 + TypeScript)
- Use Composition API with `&lt;script setup&gt;`
- Follow existing component patterns (see `web/src/components/`)
- Use the shared UI components in `web/src/components/ui/`
- Support i18n — add keys to both `en.ts` and `zh.ts`

### Commit Messages
Follow [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` New features
- `fix:` Bug fixes
- `docs:` Documentation changes
- `refactor:` Code refactoring
- `test:` Test additions/changes
- `chore:` Build/config changes
- `security:` Security fixes

## Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/your-feature`
3. Make your changes with tests
4. Run `make check` to verify linting and tests pass
5. Commit with conventional commit messages
6. Push and open a Pull Request
7. Ensure CI passes on your PR

## Reporting Issues

Please use [GitHub Issues](https://github.com/Yogdunana/deploypilot/issues) to report bugs or request features. Use the provided issue templates for better triage.

## License

By contributing, you agree that your contributions will be licensed under the [Business Source License 1.1 (BSL 1.1)](LICENSE). Non-commercial use (personal, internal organizational, testing, and non-commercial educational) is permitted. On **2029-04-28**, the license automatically converts to MIT.
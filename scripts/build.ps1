# scripts/build.ps1 — Windows 构建脚本
$APP_NAME = "deploypilot"
$VERSION = git describe --tags --always
$env:CGO_ENABLED = "1"

function Build {
    go build -ldflags "-X main.version=$VERSION" -o bin/$APP_NAME.exe ./cmd/deploypilot/
}

function Build-MCP {
    go build -ldflags "-X main.version=$VERSION" -o bin/mcp-server.exe ./cmd/mcp-server/
}

function Test {
    go test -race -coverprofile=c.out ./...
    go tool cover -func=c.out | Select-String "total"
    golangci-lint run ./...
}

function Clean {
    Remove-Item -Recurse -Force bin/ -ErrorAction SilentlyContinue
    Remove-Item -Force coverage.out -ErrorAction SilentlyContinue
}

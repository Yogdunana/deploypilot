# scripts/Verify-Deps.ps1 — 版本可用性校验 (Windows)
$ErrorActionPreference = "Stop"

$required = @{
  "github.com/gin-gonic/gin"              = "v1.12.0"
  "github.com/gin-contrib/cors"           = "v1.7.7"
  "gorm.io/gorm"                          = "v1.31.1"
  "gorm.io/driver/sqlite"                 = "v1.6.0"
  "gorm.io/driver/postgres"               = "v1.6.0"
  "github.com/go-gormigrate/gormigrate/v2"= "v2.1.5"
  "github.com/spf13/cobra"                = "v1.10.2"
  "github.com/spf13/viper"                = "v1.21.0"
  "go.uber.org/zap"                       = "v1.27.1"
  "github.com/gorilla/websocket"          = "v1.5.3"
  "github.com/golang-jwt/jwt/v5"          = "v5.3.1"
  "golang.org/x/crypto"                   = "v0.46.0"
  "github.com/prometheus/client_golang"   = "v1.23.2"
  "github.com/mark3labs/mcp-go"           = "v0.47.0"
}

Write-Host "=== DeployPilot 依赖版本校验 ==="
$fail = $false

foreach ($pkg in $required.Keys) {
  $ver = $required[$pkg]
  $output = go list -m -versions $pkg 2>$null
  if ($output -match [regex]::Escape($ver)) {
    Write-Host "  ✅ ${pkg}@${ver}"
  } else {
    Write-Host "  ❌ ${pkg}@${ver} — 版本不存在！"
    $fail = $true
  }
}

if ($fail) {
  Write-Host ""
  Write-Host "❌ 存在不可用版本，请检查 go.mod 后重试。"
  exit 1
}

Write-Host ""
Write-Host "=== 全部版本可用，可以开始编码 ==="

#!/usr/bin/env bash
# scripts/verify-deps.sh — 版本可用性校验
# 用法: bash scripts/verify-deps.sh
set -euo pipefail

MOD="github.com/Yogdunana/deploypilot"

declare -A REQUIRED=(
  ["github.com/gin-gonic/gin"]="v1.12.0"
  ["github.com/gin-contrib/cors"]="v1.7.7"
  ["gorm.io/gorm"]="v1.31.1"
  ["gorm.io/driver/sqlite"]="v1.6.0"
  ["gorm.io/driver/postgres"]="v1.6.0"
  ["github.com/go-gormigrate/gormigrate/v2"]="v2.1.5"
  ["github.com/spf13/cobra"]="v1.10.2"
  ["github.com/spf13/viper"]="v1.21.0"
  ["go.uber.org/zap"]="v1.27.1"
  ["github.com/gorilla/websocket"]="v1.5.3"
  ["github.com/golang-jwt/jwt/v5"]="v5.3.1"
  ["golang.org/x/crypto"]="v0.46.0"
  ["github.com/prometheus/client_golang"]="v1.23.2"
  ["github.com/mark3labs/mcp-go"]="v0.47.0"
)

echo "=== DeployPilot 依赖版本校验 ==="
FAIL=0

for pkg in "${!REQUIRED[@]}"; do
  ver="${REQUIRED[$pkg]}"
  if go list -m -versions "${pkg}" 2>/dev/null | grep -q "${ver}"; then
    echo "  ✅ ${pkg}@${ver}"
  else
    echo "  ❌ ${pkg}@${ver} — 版本不存在！"
    FAIL=1
  fi
done

if [ "$FAIL" -eq 1 ]; then
  echo ""
  echo "❌ 存在不可用版本，请检查 go.mod 后重试。"
  exit 1
fi

echo ""
echo "=== 全部版本可用，可以开始编码 ==="

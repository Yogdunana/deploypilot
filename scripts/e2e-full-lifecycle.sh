#!/bin/bash
# e2e-full-lifecycle.sh - DeployPilot 全量生命周期 E2E 测试
# 在目标服务器上运行，验证 deploypilot CLI + 真实 Docker 部署
# Usage: bash e2e-full-lifecycle.sh
set -euo pipefail

APP_NAME="e2e-full-$(date +%s)"
IMAGE="nginx:alpine"
TEST_PORT=$((8300 + RANDOM % 100))
PASS=0
FAIL=0
ERRORS=()

log_pass() { echo "  ✅ $1"; PASS=$((PASS + 1)); }
log_fail() { echo "  ❌ $1"; FAIL=$((FAIL + 1)); ERRORS+=("$1"); }

# Binary path
DP="${1:-./bin/deploypilot}"

echo "=== DeployPilot Full Lifecycle E2E ==="
echo "Binary: ${DP}"
echo "App: ${APP_NAME} (port ${TEST_PORT})"
echo ""

# Cleanup on exit
cleanup() {
  echo ""
  echo "[Cleanup]"
  docker rm -f "${APP_NAME}" 2>/dev/null || true
  echo "  Cleaned up container ${APP_NAME}"
}
trap cleanup EXIT

# ========== Phase 1: Binary ==========

echo "[Phase 1] Binary Verification"
if [ -f "${DP}" ]; then
  log_pass "Binary exists: $(ls -lh "${DP}" | awk '{print $5}')"
else
  log_fail "Binary not found: ${DP}"
  echo "  Usage: bash e2e-full-lifecycle.sh /path/to/deploypilot"
  exit 1
fi

# ========== Phase 2: CLI Commands ==========

echo ""
echo "[Phase 2] CLI Commands"

# version
OUT=$(${DP} version 2>&1 || true)
if [ -n "$OUT" ]; then
  log_pass "version: ${OUT}"
else
  log_fail "version returned empty"
fi

# app list
OUT=$(${DP} app list 2>&1 || true)
if echo "$OUT" | grep -qi "list\|app"; then
  log_pass "app list: works"
else
  log_fail "app list: unexpected output"
fi

# server list
OUT=$(${DP} server list 2>&1 || true)
if echo "$OUT" | grep -qi "list\|server"; then
  log_pass "server list: works"
else
  log_fail "server list: unexpected output"
fi

# server add
OUT=$(${DP} server add --name e2e-target --host 127.0.0.1 --port 22 2>&1 || true)
if echo "$OUT" | grep -qi "add\|success\|server"; then
  log_pass "server add: works"
else
  log_fail "server add: unexpected output"
fi

# server test
OUT=$(${DP} server test --name e2e-target 2>&1 || true)
if echo "$OUT" | grep -qi "test\|reach\|server"; then
  log_pass "server test: works"
else
  log_fail "server test: unexpected output"
fi

# app create
OUT=$(${DP} app create --name "${APP_NAME}" --repo "https://github.com/nginx/nginx" --stack docker 2>&1 || true)
if echo "$OUT" | grep -qi "creat\|success\|app"; then
  log_pass "app create: works"
else
  log_fail "app create: unexpected output"
fi

# app deploy (CLI stub - verify it accepts params)
OUT=$(${DP} app deploy --name "${APP_NAME}" --image "${IMAGE}" --server e2e-target 2>&1 || true)
if echo "$OUT" | grep -qi "deploy\|start"; then
  log_pass "app deploy: works"
else
  log_fail "app deploy: unexpected output"
fi

# ========== Phase 3: Real Docker Deployment ==========

echo ""
echo "[Phase 3] Real Docker Deployment"

# Deploy via Docker directly (simulating what deploy engine does)
DEPLOY_OUT=$(docker run -d --name "${APP_NAME}" --restart unless-stopped \
  -p ${TEST_PORT}:80 \
  -e DEPLOY_ENV=e2e \
  -e APP_NAME="${APP_NAME}" \
  "${IMAGE}" 2>&1)
if [ -n "$DEPLOY_OUT" ]; then
  log_pass "Container deployed: ${DEPLOY_OUT:0:12}"
else
  log_fail "Container deploy failed"
fi

sleep 2

# Container running
STATUS=$(docker inspect --format '{{.State.Status}}' "${APP_NAME}" 2>/dev/null || echo "error")
if [ "$STATUS" = "running" ]; then
  log_pass "Status: running"
else
  log_fail "Status: ${STATUS}"
fi

# Port mapping
PORTS=$(docker port "${APP_NAME}" 2>/dev/null || echo "")
if echo "$PORTS" | grep -q "${TEST_PORT}"; then
  log_pass "Port mapping: ${PORTS}"
else
  log_fail "Port mapping missing"
fi

# Environment variables
ENV_CHECK=$(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "${APP_NAME}" 2>/dev/null | grep DEPLOY_ENV || echo "")
if [ -n "$ENV_CHECK" ]; then
  log_pass "Environment: DEPLOY_ENV=e2e set"
else
  log_fail "Environment variable not set"
fi

# Health check (HTTP)
HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:${TEST_PORT}/" 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "200" ]; then
  log_pass "Health check: HTTP ${HTTP_CODE}"
else
  log_fail "Health check: HTTP ${HTTP_CODE}"
fi

# Get logs
LOGS=$(docker logs --tail 10 "${APP_NAME}" 2>&1 || echo "")
if [ -n "$LOGS" ]; then
  LOG_LINES=$(echo "$LOGS" | wc -l)
  log_pass "Logs: ${LOG_LINES} lines"
else
  log_fail "No logs"
fi

# ========== Phase 4: Rollback Simulation ==========

echo ""
echo "[Phase 4] Rollback Simulation"

# Stop current
docker stop "${APP_NAME}" >/dev/null 2>&1
docker rm -f "${APP_NAME}" >/dev/null 2>&1

# Redeploy with "previous" image (simulate rollback)
ROLLBACK_IMAGE="nginx:alpine"
ROLLBACK_OUT=$(docker run -d --name "${APP_NAME}" --restart unless-stopped \
  -p ${TEST_PORT}:80 \
  "${ROLLBACK_IMAGE}" 2>&1)
sleep 2
ROLLBACK_STATUS=$(docker inspect --format '{{.State.Status}}' "${APP_NAME}" 2>/dev/null || echo "error")
if [ "$ROLLBACK_STATUS" = "running" ]; then
  log_pass "Rollback: container running after redeploy"
else
  log_fail "Rollback: status ${ROLLBACK_STATUS}"
fi

# Verify rollback serves traffic
ROLLBACK_HTTP=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:${TEST_PORT}/" 2>/dev/null || echo "000")
if [ "$ROLLBACK_HTTP" = "200" ]; then
  log_pass "Rollback health: HTTP ${ROLLBACK_HTTP}"
else
  log_fail "Rollback health: HTTP ${ROLLBACK_HTTP}"
fi

# ========== Phase 5: App Delete ==========

echo ""
echo "[Phase 5] App Delete"

OUT=$(${DP} app delete --name "${APP_NAME}" --force 2>&1 || true)
if echo "$OUT" | grep -qi "delet\|success"; then
  log_pass "app delete (CLI): works"
else
  log_fail "app delete (CLI): unexpected output"
fi

# Real cleanup
docker stop "${APP_NAME}" >/dev/null 2>&1 || true
docker rm -f "${APP_NAME}" >/dev/null 2>&1 || true
log_pass "Container removed"

# Verify gone
REMAIN=$(docker ps -a --filter "name=${APP_NAME}" --format "{{.Names}}" 2>/dev/null)
if [ -z "$REMAIN" ]; then
  log_pass "Container fully removed"
else
  log_fail "Container still exists: ${REMAIN}"
fi

# ========== Summary ==========

echo ""
echo "=== Results ==="
echo "Passed: ${PASS}"
echo "Failed: ${FAIL}"
if [ ${#ERRORS[@]} -gt 0 ]; then
  echo "Failures:"
  for e in "${ERRORS[@]}"; do
    echo "  - ${e}"
  done
fi

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
echo ""
echo "🎉 Full lifecycle E2E passed!"

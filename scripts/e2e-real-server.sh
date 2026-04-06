#!/bin/bash
# e2e-real-server.sh - 真机 E2E 测试
# Usage: ./e2e-real-server.sh <server-host> <ssh-user> [ssh-port]
# Requires: SSH access + Docker on target server
set -euo pipefail

HOST="${1:?Usage: $0 <host> <user> [port]}"
USER="${2:?Usage: $0 <host> <user> [port]}"
PORT="${3:-22}"
APP_NAME="e2e-test-$(date +%s)"
IMAGE="nginx:alpine"
TEST_PORT=$((8100 + RANDOM % 100))

PASS=0
FAIL=0
ERRORS=()

log_pass() { echo "  ✅ $1"; ((PASS++)); }
log_fail() { echo "  ❌ $1"; ((FAIL++)); ERRORS+=("$1"); }

ssh_cmd() {
  ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -p "$PORT" "${USER}@${HOST}" "$@"
}

echo "=== DeployPilot Real Server E2E ==="
echo "Host: ${USER}@${HOST}:${PORT}"
echo "App: ${APP_NAME} (port ${TEST_PORT})"
echo ""

# Cleanup on exit
cleanup() {
  echo ""
  echo "[Cleanup] Removing test container..."
  ssh_cmd "docker rm -f ${APP_NAME} 2>/dev/null || true"
  echo "[Cleanup] Done"
}
trap cleanup EXIT

# Test 1: SSH Connectivity
echo "[Test 1] SSH Connectivity"
if ssh_cmd "echo ok" | grep -q "ok"; then
  log_pass "SSH connection works"
else
  log_fail "SSH connection failed"
fi

# Test 2: Docker Available
echo "[Test 2] Docker Available"
DOCKER_VER=$(ssh_cmd "docker --version" 2>/dev/null || echo "")
if [ -n "$DOCKER_VER" ]; then
  log_pass "Docker: ${DOCKER_VER}"
else
  log_fail "Docker not found"
  echo "  Run scripts/deploy-server.sh first"
  exit 1
fi

# Test 3: Deploy Container
echo "[Test 3] Deploy Container"
DEPLOY_OUT=$(ssh_cmd "docker run -d --name ${APP_NAME} --restart unless-stopped -p ${TEST_PORT}:80 ${IMAGE}" 2>&1)
if [ -n "$DEPLOY_OUT" ]; then
  log_pass "Container deployed: ${DEPLOY_OUT:0:12}"
else
  log_fail "Deploy failed: ${DEPLOY_OUT}"
fi

# Test 4: Container Running
echo "[Test 4] Container Running"
sleep 2
STATUS=$(ssh_cmd "docker inspect --format '{{.State.Status}}' ${APP_NAME}" 2>/dev/null || echo "error")
if [ "$STATUS" = "running" ]; then
  log_pass "Status: running"
else
  log_fail "Status: ${STATUS}"
fi

# Test 5: Health Check (HTTP)
echo "[Test 5] Health Check (HTTP)"
# Try to reach via SSH tunnel
HTTP_CODE=$(ssh_cmd "curl -s -o /dev/null -w '%{http_code}' http://localhost:${TEST_PORT}/ 2>/dev/null || echo 000")
if [ "$HTTP_CODE" = "200" ]; then
  log_pass "HTTP ${HTTP_CODE}"
else
  log_fail "HTTP ${HTTP_CODE} (expected 200)"
fi

# Test 6: Get Logs
echo "[Test 6] Get Container Logs"
LOGS=$(ssh_cmd "docker logs --tail 5 ${APP_NAME} 2>&1" || echo "")
if [ -n "$LOGS" ]; then
  log_pass "Logs retrieved ($(echo "$LOGS" | wc -l) lines)"
else
  log_fail "No logs"
fi

# Test 7: Stop Container
echo "[Test 7] Stop Container"
STOP_OUT=$(ssh_cmd "docker stop ${APP_NAME}" 2>&1 || echo "failed")
STATUS_AFTER=$(ssh_cmd "docker inspect --format '{{.State.Status}}' ${APP_NAME}" 2>/dev/null || echo "error")
if [ "$STATUS_AFTER" = "exited" ]; then
  log_pass "Container stopped"
else
  log_fail "Container not stopped: ${STATUS_AFTER}"
fi

# Test 8: Rollback (Redeploy)
echo "[Test 8] Rollback (Redeploy)"
ssh_cmd "docker rm -f ${APP_NAME} 2>/dev/null || true"
ROLLBACK_OUT=$(ssh_cmd "docker run -d --name ${APP_NAME} --restart unless-stopped -p ${TEST_PORT}:80 ${IMAGE}" 2>&1)
sleep 2
ROLLBACK_STATUS=$(ssh_cmd "docker inspect --format '{{.State.Status}}' ${APP_NAME}" 2>/dev/null || echo "error")
if [ "$ROLLBACK_STATUS" = "running" ]; then
  log_pass "Rollback successful"
else
  log_fail "Rollback failed: ${ROLLBACK_STATUS}"
fi

# Summary
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

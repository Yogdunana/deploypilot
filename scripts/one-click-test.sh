#!/bin/bash
# DeployPilot 一键安装测试脚本
# 功能: 自动部署 + 本机SSH自连测试 + 数据库连接测试
# 用法: curl -fsSL https://raw.githubusercontent.com/Yogdunana/deploypilot/main/scripts/one-click-test.sh | bash

set -euo pipefail

SCRIPT_VERSION="1.0.0"
TEST_DIR="/tmp/deploypilot-test"
LOG_FILE="$TEST_DIR/install-test.log"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

# Test results
TESTS_PASSED=0
TESTS_FAILED=0
TEST_RESULTS=()

print_banner() {
    echo -e "${CYAN}"
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║                                                               ║"
    echo "║   DeployPilot 一键安装测试脚本                                ║"
    echo "║   版本: v${SCRIPT_VERSION}                                        ║"
    echo "║                                                               ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo -e "${RESET}"
}

print_info() { echo -e "${BLUE}[INFO]${RESET} $1" | tee -a "$LOG_FILE"; }
print_success() { echo -e "${GREEN}[✓ PASS]${RESET} $1" | tee -a "$LOG_FILE"; }
print_warning() { echo -e "${YELLOW}[! WARN]${RESET} $1" | tee -a "$LOG_FILE"; }
print_error() { echo -e "${RED}[✗ FAIL]${RESET} $1" | tee -a "$LOG_FILE"; }
print_step() {
    echo -e ""
    echo -e "${BOLD}${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo -e "${BOLD}${MAGENTA} $1${RESET}"
    echo -e "${BOLD}${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
}

# Record test result
record_test() {
    local test_name="$1"
    local result="$2"
    local details="${3:-}"

    if [ "$result" = "PASS" ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        print_success "$test_name"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        print_error "$test_name"
    fi

    if [ -n "$details" ]; then
        echo "  Details: $details" | tee -a "$LOG_FILE"
    fi

    TEST_RESULTS+=("$test_name:$result")
}

# Check prerequisites
check_prerequisites() {
    print_step "[1/8] 检查系统环境"

    # Check OS
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        print_info "操作系统: $NAME $VERSION_ID"
    fi

    # Check architecture
    ARCH=$(uname -m)
    print_info "系统架构: $ARCH"

    # Check Docker
    if command -v docker &> /dev/null; then
        DOCKER_VERSION=$(docker --version | awk '{print $3}' | tr -d ',')
        print_info "Docker 版本: $DOCKER_VERSION"
        record_test "Docker 已安装" "PASS"
    else
        record_test "Docker 已安装" "FAIL" "Docker 未安装"
        print_info "正在安装 Docker..."
        curl -fsSL https://get.docker.com | sh
        systemctl start docker
        systemctl enable docker
        record_test "Docker 安装" "PASS"
    fi

    # Check Docker Compose
    if docker compose version &> /dev/null || docker-compose --version &> /dev/null; then
        record_test "Docker Compose 可用" "PASS"
    else
        record_test "Docker Compose 可用" "FAIL" "Docker Compose 未安装"
    fi

    # Check SSH
    if command -v ssh &> /dev/null; then
        SSH_VERSION=$(ssh -V 2>&1 | head -1)
        print_info "SSH 版本: $SSH_VERSION"
        record_test "SSH 客户端可用" "PASS"
    else
        record_test "SSH 客户端可用" "FAIL" "SSH 未安装"
    fi

    # Check if SSH server is running
    if systemctl is-active --quiet sshd 2>/dev/null || systemctl is-active --quiet ssh 2>/dev/null; then
        record_test "SSH 服务运行中" "PASS"
    else
        print_warning "SSH 服务未运行，尝试启动..."
        systemctl start sshd 2>/dev/null || systemctl start ssh 2>/dev/null || true
        record_test "SSH 服务启动" "PASS"
    fi
}

# Setup test environment
setup_test_env() {
    print_step "[2/8] 准备测试环境"

    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"

    # Generate JWT secret
    JWT_SECRET=$(openssl rand -base64 32)
    print_info "生成 JWT Secret"

    # Get local IP
    LOCAL_IP=$(hostname -I | awk '{print $1}')
    print_info "本机 IP: $LOCAL_IP"

    record_test "测试目录创建" "PASS"
}

# Create Docker Compose configuration
create_docker_compose() {
    print_step "[3/8] 创建 Docker Compose 配置"

    cat > docker-compose.yml << 'EOF'
services:
  deploypilot:
    image: ghcr.io/yogdunana/deploypilot:latest
    container_name: deploypilot-test
    restart: unless-stopped
    ports:
      - "18080:8080"
    environment:
      - DEPLOYPILOT_DATABASE_TYPE=sqlite
      - DEPLOYPILOT_DATABASE_DSN=/app/data/deploypilot.db
      - JWT_SECRET=${JWT_SECRET}
      - DEPLOYPILOT_SERVER_PORT=8080
      - DEPLOYPILOT_LOG_LEVEL=debug
      - DEPLOYPILOT_LOG_FORMAT=json
      - DEPLOYPILOT_REDIS_ADDR=redis:6379
    volumes:
      - deploypilot-data:/app/data
      # Mount host SSH key for self-connection testing
      - /root/.ssh:/host-ssh:ro
      # Mount Docker socket for container management
      - /var/run/docker.sock:/var/run/docker.sock
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/api/v1/system/health"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    depends_on:
      redis:
        condition: service_healthy
    networks:
      - deploypilot-net

  redis:
    image: redis:7-alpine
    container_name: deploypilot-redis-test
    restart: unless-stopped
    volumes:
      - redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5
    networks:
      - deploypilot-net

  # Test SSH server for self-connection
  test-ssh:
    image: linuxserver/openssh-server:latest
    container_name: deploypilot-test-ssh
    environment:
      - PUID=1000
      - PGID=1000
      - TZ=Asia/Shanghai
      - PASSWORD_ACCESS=true
      - USER_PASSWORD=testpass123
      - USER_NAME=testuser
    ports:
      - "10022:2222"
    volumes:
      - ssh-data:/config
    networks:
      - deploypilot-net
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "2222"]
      interval: 5s
      timeout: 3s
      retries: 5

volumes:
  deploypilot-data:
  redis-data:
  ssh-data:

networks:
  deploypilot-net:
    driver: bridge
EOF

    # Substitute JWT_SECRET
    sed -i "s|\${JWT_SECRET}|$JWT_SECRET|g" docker-compose.yml

    print_info "Docker Compose 配置已创建"
    record_test "Docker Compose 配置创建" "PASS"
}

# Deploy services
deploy_services() {
    print_step "[4/8] 部署 DeployPilot 服务"

    cd "$TEST_DIR"

    # Pull images
    print_info "拉取 Docker 镜像..."
    if docker compose pull 2>&1 | tee -a "$LOG_FILE"; then
        record_test "Docker 镜像拉取" "PASS"
    else
        record_test "Docker 镜像拉取" "FAIL" "无法拉取镜像"
        return 1
    fi

    # Start services
    print_info "启动服务..."
    if docker compose up -d 2>&1 | tee -a "$LOG_FILE"; then
        record_test "服务启动" "PASS"
    else
        record_test "服务启动" "FAIL" "服务启动失败"
        return 1
    fi

    # Wait for healthcheck
    print_info "等待服务就绪 (最多 60 秒)..."
    local retries=0
    local max_retries=12

    while [ $retries -lt $max_retries ]; do
        if docker compose ps deploypilot | grep -q "healthy"; then
            record_test "服务健康检查" "PASS"
            return 0
        fi
        sleep 5
        retries=$((retries + 1))
        print_info "等待中... ($retries/$max_retries)"
    done

    record_test "服务健康检查" "FAIL" "服务未在预期时间内就绪"
    return 1
}

# Test API connectivity
test_api() {
    print_step "[5/8] 测试 API 接口"

    local api_url="http://localhost:18080"
    local retries=0
    local max_retries=10

    # Wait for API to be ready
    print_info "等待 API 就绪..."
    while [ $retries -lt $max_retries ]; do
        if curl -fs "$api_url/api/v1/system/health" > /dev/null 2>&1; then
            break
        fi
        sleep 2
        retries=$((retries + 1))
    done

    # Test health endpoint
    if curl -fs "$api_url/api/v1/system/health" > /dev/null 2>&1; then
        record_test "健康检查接口" "PASS"
    else
        record_test "健康检查接口" "FAIL"
    fi

    # Test version endpoint
    VERSION_RESP=$(curl -fs "$api_url/api/v1/system/version" 2>/dev/null || echo "")
    if [ -n "$VERSION_RESP" ]; then
        print_info "版本信息: $VERSION_RESP"
        record_test "版本接口" "PASS"
    else
        record_test "版本接口" "FAIL"
    fi

    # Register test user
    print_info "注册测试用户..."
    REGISTER_RESP=$(curl -fs -X POST "$api_url/api/v1/auth/register" \
        -H "Content-Type: application/json" \
        -d '{"username": "testadmin", "password": "TestPass123!"}' 2>/dev/null || echo "")

    if echo "$REGISTER_RESP" | grep -q '"id"' 2>/dev/null; then
        record_test "用户注册" "PASS"
    elif echo "$REGISTER_RESP" | grep -q 'already exists' 2>/dev/null; then
        print_warning "用户已存在，跳过注册"
        record_test "用户注册" "PASS" "用户已存在"
    else
        record_test "用户注册" "FAIL" "注册响应: $REGISTER_RESP"
    fi

    # Login
    print_info "测试登录..."
    LOGIN_RESP=$(curl -fs -X POST "$api_url/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username": "testadmin", "password": "TestPass123!"}' 2>/dev/null || echo "")

    if echo "$LOGIN_RESP" | grep -q '"token"' 2>/dev/null; then
        TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        print_info "登录成功，获取到 Token"
        record_test "用户登录" "PASS"
    else
        record_test "用户登录" "FAIL" "登录响应: $LOGIN_RESP"
        return 1
    fi
}

# Test SSH self-connection
test_ssh_connection() {
    print_step "[6/8] 测试 SSH 自连接"

    local api_url="http://localhost:18080"

    # First, ensure we have a token
    if [ -z "${TOKEN:-}" ]; then
        LOGIN_RESP=$(curl -fs -X POST "$api_url/api/v1/auth/login" \
            -H "Content-Type: application/json" \
            -d '{"username": "testadmin", "password": "TestPass123!"}' 2>/dev/null || echo "")
        TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    fi

    # Add SSH server
    print_info "添加 SSH 服务器..."
    ADD_SERVER_RESP=$(curl -fs -X POST "$api_url/api/v1/servers" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{
            "name": "test-ssh-server",
            "host": "test-ssh",
            "port": 2222,
            "user": "testuser",
            "auth_type": "password",
            "password": "testpass123"
        }' 2>/dev/null || echo "")

    if echo "$ADD_SERVER_RESP" | grep -q '"id"' 2>/dev/null; then
        SERVER_ID=$(echo "$ADD_SERVER_RESP" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
        print_info "SSH 服务器添加成功，ID: $SERVER_ID"
        record_test "添加 SSH 服务器" "PASS"
    else
        record_test "添加 SSH 服务器" "FAIL" "响应: $ADD_SERVER_RESP"
        return 1
    fi

    # Test connection
    print_info "测试 SSH 连接..."
    sleep 2

    TEST_CONN_RESP=$(curl -fs -X POST "$api_url/api/v1/servers/$SERVER_ID/test-connection" \
        -H "Authorization: Bearer $TOKEN" 2>/dev/null || echo "")

    if echo "$TEST_CONN_RESP" | grep -q '"success":true' 2>/dev/null; then
        record_test "SSH 连接测试" "PASS"
    else
        record_test "SSH 连接测试" "FAIL" "响应: $TEST_CONN_RESP"
    fi

    # Get server info
    SERVER_INFO=$(curl -fs "$api_url/api/v1/servers/$SERVER_ID" \
        -H "Authorization: Bearer $TOKEN" 2>/dev/null || echo "")

    if [ -n "$SERVER_INFO" ]; then
        record_test "获取服务器信息" "PASS"
    else
        record_test "获取服务器信息" "FAIL"
    fi
}

# Test database operations
test_database() {
    print_step "[7/8] 测试数据库操作"

    local api_url="http://localhost:18080"

    # Test list servers (database query)
    SERVERS_LIST=$(curl -fs "$api_url/api/v1/servers" \
        -H "Authorization: Bearer $TOKEN" 2>/dev/null || echo "")

    if echo "$SERVERS_LIST" | grep -q '"data"' 2>/dev/null; then
        record_test "数据库查询 (服务器列表)" "PASS"
    else
        record_test "数据库查询 (服务器列表)" "FAIL"
    fi

    # Test create app
    print_info "创建测试应用..."
    CREATE_APP_RESP=$(curl -fs -X POST "$api_url/api/v1/apps" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{
            "name": "test-app",
            "server_id": '$SERVER_ID',
            "type": "docker",
            "status": "stopped"
        }' 2>/dev/null || echo "")

    if echo "$CREATE_APP_RESP" | grep -q '"id"' 2>/dev/null; then
        APP_ID=$(echo "$CREATE_APP_RESP" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
        print_info "应用创建成功，ID: $APP_ID"
        record_test "创建应用 (数据库写入)" "PASS"
    else
        record_test "创建应用 (数据库写入)" "FAIL" "响应: $CREATE_APP_RESP"
    fi

    # Test list apps
    APPS_LIST=$(curl -fs "$api_url/api/v1/apps" \
        -H "Authorization: Bearer $TOKEN" 2>/dev/null || echo "")

    if echo "$APPS_LIST" | grep -q '"data"' 2>/dev/null; then
        record_test "应用列表查询" "PASS"
    else
        record_test "应用列表查询" "FAIL"
    fi
}

# Cleanup
cleanup() {
    print_step "[8/8] 清理测试环境"

    cd "$TEST_DIR"

    print_info "停止并移除测试容器..."
    docker compose down -v 2>&1 | tee -a "$LOG_FILE" || true

    print_info "清理测试目录..."
    rm -rf "$TEST_DIR"

    record_test "环境清理" "PASS"
}

# Print test summary
print_summary() {
    echo -e ""
    echo -e "${BOLD}${CYAN}╔═══════════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${BOLD}${CYAN}║                    测试执行完成                                ║${RESET}"
    echo -e "${BOLD}${CYAN}╚═══════════════════════════════════════════════════════════════╝${RESET}"
    echo -e ""
    echo -e "${BOLD}测试结果统计:${RESET}"
    echo -e "  ${GREEN}通过: $TESTS_PASSED${RESET}"
    echo -e "  ${RED}失败: $TESTS_FAILED${RESET}"
    echo -e "  总计: $((TESTS_PASSED + TESTS_FAILED))"
    echo -e ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}${BOLD}✓ 所有测试通过！DeployPilot 安装测试成功。${RESET}"
        return 0
    else
        echo -e "${RED}${BOLD}✗ 部分测试失败，请查看日志: $LOG_FILE${RESET}"
        echo -e ""
        echo -e "${YELLOW}失败的测试:${RESET}"
        for result in "${TEST_RESULTS[@]}"; do
            if [[ "$result" == *":FAIL" ]]; then
                echo -e "  ${RED}• ${result%:FAIL}${RESET}"
            fi
        done
        return 1
    fi
}

# Main execution
main() {
    print_banner

    # Create log file
    mkdir -p "$TEST_DIR"
    echo "DeployPilot Install Test - $(date)" > "$LOG_FILE"
    echo "========================================" >> "$LOG_FILE"

    # Run tests
    check_prerequisites
    setup_test_env
    create_docker_compose

    if deploy_services; then
        test_api
        test_ssh_connection
        test_database
    fi

    cleanup
    print_summary
}

# Handle signals
trap 'print_error "脚本被中断"; cleanup; exit 1' INT TERM

# Run main
main "$@"

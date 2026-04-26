#!/bin/bash
set -euo pipefail
SCRIPT_VERSION="2.0.0"
DEFAULT_INSTALL_DIR="/opt/deploypilot"
DEFAULT_PORT="8080"
GITHUB_REPO="Yogdunana/deploypilot"
# Detect latest version from GitHub
get_latest_version() {
    local url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$url" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        echo ""
    fi
}
# Detect if in non-interactive mode
NON_INTERACTIVE=false
if [ "${1:-}" = "--non-interactive" ] || [ "${NON_INTERACTIVE_MODE:-false}" = "true" ]; then
    NON_INTERACTIVE=true
fi
if [ "${1:-}" = "--interactive" ] || [ "${FORCE_INTERACTIVE:-false}" = "true" ]; then
    NON_INTERACTIVE=false
fi
# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
BOLD='\033[1m'
RESET='\033[0m'
function print_banner() {
    echo -e "${CYAN}"
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║                                                               ║"
    echo "║   ██████╗ ███████╗██████╗ ██╗      ██████╗ ██╗   ██╗          ║"
    echo "║   ██╔══██╗██╔════╝██╔══██╗██║     ██╔═══██╗╚██╗ ██╔╝          ║"
    echo "║   ██║  ██║█████╗  ██████╔╝██║     ██║   ██║ ╚████╔╝           ║"
    echo "║   ██║  ██║██╔══╝  ██╔═══╝ ██║     ██║   ██║  ╚██╔╝            ║"
    echo "║   ██████╔╝███████╗██║     ███████╗╚██████╔╝   ██║             ║"
    echo "║   ╚═════╝ ╚══════╝╚═╝     ╚══════╝ ╚═════╝    ╚═╝             ║"
    echo "║                                                               ║"
    echo "║                     一键安装脚本 v${SCRIPT_VERSION}                      ║"
    echo "║                                                               ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo -e "${RESET}"
}
function print_info() {
    echo -e "${BLUE}[INFO]${RESET} $1"
}
function print_success() {
    echo -e "${GREEN}[SUCCESS]${RESET} $1"
}
function print_warning() {
    echo -e "${YELLOW}[WARNING]${RESET} $1"
}
function print_error() {
    echo -e "${RED}[ERROR]${RESET} $1"
}
function print_step() {
    echo -e ""
    echo -e "${BOLD}${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo -e "${BOLD}${MAGENTA} 步骤 $1: $2${RESET}"
    echo -e "${BOLD}${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
}
function random_string() {
    local length=${1:-12}
    tr -dc 'a-z0-9' < /dev/urandom | head -c "$length"
}
function generate_password() {
    local length=${1:-16}
    tr -dc 'a-zA-Z0-9!@#$%^&*' < /dev/urandom | head -c "$length"
}
function generate_username() {
    echo "deploy_$(random_string 8)"
}
function check_root() {
    if [ "$EUID" -ne 0 ]; then
        print_error "请使用 root 用户或 sudo 运行此脚本"
        exit 1
    fi
}
function detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            print_error "不支持的架构: $arch"
            exit 1
            ;;
    esac
}
function detect_os() {
    local os=""
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        os=$ID
    elif [ -f /etc/redhat-release ]; then
        os="centos"
    elif [ -f /etc/debian_version ]; then
        os="debian"
    else
        print_error "无法检测操作系统"
        exit 1
    fi
    echo "$os"
}
function detect_systemd() {
    if command -v systemctl >/dev/null 2>&1; then
        echo "true"
    else
        echo "false"
    fi
}
function get_ip_address() {
    local ip
    ip=$(hostname -I | awk '{print $1}')
    if [ -z "$ip" ]; then
        ip=$(curl -s ifconfig.me 2>/dev/null || echo "127.0.0.1")
    fi
    echo "$ip"
}
function prompt_input() {
    local prompt="$1"
    local default="$2"
    local result

    if [ "$NON_INTERACTIVE" = true ]; then
        echo "$default"
        return
    fi

    echo -ne "${CYAN}${prompt}${RESET}"
    if [ -n "$default" ]; then
        echo -ne " [${YELLOW}${default}${RESET}]"
    fi
    echo -ne ": "

    read -r result
    if [ -z "$result" ]; then
        result="$default"
    fi
    echo "$result"
}
function prompt_confirm() {
    local prompt="$1"
    local default="${2:-y}"
    local result

    if [ "$NON_INTERACTIVE" = true ]; then
        echo -e "${BLUE}[INFO]${RESET} ${prompt} (非交互式模式，使用默认值: ${YELLOW}${default}${RESET})"
        if [ "$default" = "y" ]; then
            return 0
        else
            return 1
        fi
    fi

    while true; do
        echo -ne "${CYAN}${prompt}${RESET} [${YELLOW}${default}${RESET}]: "
        read -r result
        result=${result:-$default}

        case "$result" in
            [Yy]*) return 0 ;;
            [Nn]*) return 1 ;;
            *) print_warning "请输入 y 或 n" ;;
        esac
    done
}
function download_file() {
    local url="$1"
    local output="$2"
    local desc="$3"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "$output"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$url" -O "$output"
    else
        print_error "需要 curl 或 wget 来下载文件"
        exit 1
    fi
}
function install_docker() {
    local os="$1"
    local use_mirror="$2"

    print_info "正在安装 Docker..."

    case "$os" in
        ubuntu|debian)
            apt-get update -y
            apt-get install -y apt-transport-https ca-certificates curl gnupg lsb-release

            if [ "$use_mirror" = "true" ]; then
                curl -fsSL https://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/trusted.gpg.d/docker.gpg
                echo "deb [arch=$(dpkg --print-architecture)] https://mirrors.aliyun.com/docker-ce/linux/ubuntu $(lsb_release -cs) stable" > /etc/apt/sources.list.d/docker.list
            else
                curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/trusted.gpg.d/docker.gpg
                echo "deb [arch=$(dpkg --print-architecture)] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" > /etc/apt/sources.list.d/docker.list
            fi

            apt-get update -y
            apt-get install -y docker-ce docker-ce-cli containerd.io
            ;;
        centos|rhel)
            yum install -y yum-utils

            if [ "$use_mirror" = "true" ]; then
                yum-config-manager --add-repo https://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo
            else
                yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
            fi

            yum install -y docker-ce docker-ce-cli containerd.io
            ;;
        *)
            print_warning "未知的操作系统，尝试使用官方脚本安装"
            if [ "$use_mirror" = "true" ]; then
                curl -fsSL https://get.daocloud.io/docker | sh
            else
                curl -fsSL https://get.docker.com | sh
            fi
            ;;
    esac

    systemctl start docker
    systemctl enable docker
    print_success "Docker 安装成功"
}
function configure_firewall() {
    local port="$1"

    print_info "正在配置防火墙..."

    if command -v ufw >/dev/null 2>&1; then
        if ufw status | grep -q "active"; then
            ufw allow "$port"/tcp >/dev/null 2>&1
            print_success "已在 UFW 中开放端口 $port"
        fi
    elif command -v firewall-cmd >/dev/null 2>&1; then
        if systemctl is-active --quiet firewalld; then
            firewall-cmd --permanent --add-port="$port"/tcp >/dev/null 2>&1
            firewall-cmd --reload >/dev/null 2>&1
            print_success "已在 firewalld 中开放端口 $port"
        fi
    fi
}
function download_binaries() {
    local version="$1"
    local arch="$2"
    local dest_dir="$3"
    local use_mirror="$4"

    local base_url
    if [ "$use_mirror" = "true" ]; then
        # Use GitHub mirror for users in China
        base_url="https://mirror.ghproxy.com/https://github.com/${GITHUB_REPO}/releases/download/${version}"
    else
        base_url="https://github.com/${GITHUB_REPO}/releases/download/${version}"
    fi

    print_info "正在下载 DeployPilot v${version} (${arch}) ..."

    local binaries=("api-server" "mcp-server" "deploypilot")
    local download_success=true

    for bin in "${binaries[@]}"; do
        local filename="${bin}-linux-${arch}"
        local url="${base_url}/${filename}"
        local output="${dest_dir}/${bin}"

        print_info "  下载 ${filename} ..."
        if download_file "$url" "$output" "${bin}"; then
            chmod +x "$output"
            print_success "  ${bin} 下载完成"
        else
            print_error "  ${bin} 下载失败"
            download_success=false
        fi
    done

    if [ "$download_success" = false ]; then
        print_error "二进制文件下载失败，请检查网络连接或版本号"
        print_info "可手动下载: https://github.com/${GITHUB_REPO}/releases"
        exit 1
    fi
}
# Input validation functions to prevent command injection
validate_port() {
    local port="$1"
    if ! [[ "$port" =~ ^[0-9]+$ ]] || [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
        print_error "端口必须是 1-65535 之间的数字"
        exit 1
    fi
}

validate_install_dir() {
    local dir="$1"
    if [[ "$dir" =~ [\$\`\|\;\&\>\<\(\)\{\}\[\]'"\!\\~
] ]]; then
        print_error "安装路径包含非法字符"
        exit 1
    fi
    if [[ "$dir" != /* ]]; then
        print_error "安装路径必须是绝对路径"
        exit 1
    fi
}

validate_username() {
    local name="$1"
    if ! [[ "$name" =~ ^[a-zA-Z_][a-zA-Z0-9_-]*$ ]]; then
        print_error "用户名只能包含字母、数字、下划线和连字符，且必须以字母或下划线开头"
        exit 1
    fi
    if [ ${#name} -gt 32 ]; then
        print_error "用户名长度不能超过 32 个字符"
        exit 1
    fi
}

function main() {
    print_banner

    check_root

    print_step "1/10" "系统环境检测"

    ARCH=$(detect_arch)
    OS=$(detect_os)
    HAS_SYSTEMD=$(detect_systemd)
    IP_ADDRESS=$(get_ip_address)

    print_info "架构: $ARCH"
    print_info "操作系统: $OS"
    print_info "Systemd: $HAS_SYSTEMD"
    print_info "IP 地址: $IP_ADDRESS"

    print_step "2/10" "获取最新版本"

    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
        print_warning "无法获取最新版本，将使用源码构建方式"
        INSTALL_MODE="source"
    else
        print_success "最新版本: ${VERSION}"
        INSTALL_MODE="binary"
    fi

    print_step "3/10" "交互式配置"

    INSTALL_DIR=$(prompt_input "请输入安装路径" "$DEFAULT_INSTALL_DIR")
    PORT=$(prompt_input "请输入端口" "$DEFAULT_PORT")
    USERNAME=$(prompt_input "请输入用户名" "$(generate_username)")
    PASSWORD=$(prompt_input "请输入密码" "$(generate_password)")

    # Validate user inputs
    validate_install_dir "$INSTALL_DIR"
    validate_port "$PORT"
    validate_username "$USERNAME"

    echo ""
    if prompt_confirm "是否使用国内镜像源？" "y"; then
        USE_MIRROR="true"
    else
        USE_MIRROR="false"
    fi

    print_step "4/10" "安装配置确认"

    echo ""
    echo -e "${BOLD}${WHITE}════════════════════════════════════════════════════════════════${RESET}"
    echo -e "${BOLD}${WHITE}  安装配置摘要${RESET}"
    echo -e "${BOLD}${WHITE}════════════════════════════════════════════════════════════════${RESET}"
    echo -e "  ${CYAN}安装路径:${RESET}    ${INSTALL_DIR}"
    echo -e "  ${CYAN}端口:${RESET}        ${PORT}"
    echo -e "  ${CYAN}用户名:${RESET}      ${USERNAME}"
    echo -e "  ${CYAN}密码:${RESET}        ${PASSWORD}"
    echo -e "  ${CYAN}版本:${RESET}        ${VERSION:-source}"
    echo -e "  ${CYAN}国内镜像:${RESET}    ${USE_MIRROR}"
    echo -e "${BOLD}${WHITE}════════════════════════════════════════════════════════════════${RESET}"
    echo ""

    if ! prompt_confirm "确认以上配置并开始安装？" "y"; then
        print_warning "安装已取消"
        exit 0
    fi

    print_step "5/10" "检查并安装 Docker"

    if ! command -v docker >/dev/null 2>&1; then
        install_docker "$OS" "$USE_MIRROR"
    else
        print_success "Docker 已安装"
        docker --version
    fi

    print_step "6/10" "创建安装目录"

    print_info "正在创建目录结构..."
    mkdir -p "$INSTALL_DIR"/{bin,data,logs,backups,config}
    print_success "目录创建成功"

    print_step "7/10" "下载 DeployPilot 二进制文件"

    if [ "$INSTALL_MODE" = "binary" ]; then
        download_binaries "$VERSION" "$ARCH" "$INSTALL_DIR/bin" "$USE_MIRROR"
    else
        print_warning "无法获取 Release 版本，请手动构建或从 GitHub Release 页面下载"
        print_info "GitHub Release: https://github.com/${GITHUB_REPO}/releases"
        print_info "构建指南: https://github.com/${GITHUB_REPO}#从源码构建"
        exit 1
    fi

    print_step "8/10" "创建配置文件"

    print_info "正在创建配置文件..."
    JWT_SECRET=$(random_string 32)
    cat > "$INSTALL_DIR/config/config.yaml" << EOF
server:
  port: ${PORT}
  host: 0.0.0.0
database:
  path: ${INSTALL_DIR}/data/deploypilot.db
storage:
  data_dir: ${INSTALL_DIR}/data
  logs_dir: ${INSTALL_DIR}/logs
  backups_dir: ${INSTALL_DIR}/backups
auth:
  jwt_secret: ${JWT_SECRET}
EOF

    chmod 600 "$INSTALL_DIR/config/config.yaml"
    print_success "配置文件创建成功"

    print_step "9/10" "创建用户和初始化"

    print_info "正在创建系统用户..."
    if ! id deploypilot >/dev/null 2>&1; then
        useradd -m -s /bin/bash -d "$INSTALL_DIR" deploypilot
    fi
    usermod -aG docker deploypilot
    chown -R deploypilot:deploypilot "$INSTALL_DIR"

    print_info "正在初始化用户账号..."
    cat > "$INSTALL_DIR/data/initial-users.json" << EOF
[
  {
    "username": "${USERNAME}",
    "password": "${PASSWORD}",
    "role": "admin"
  }
]
EOF
    chown deploypilot:deploypilot "$INSTALL_DIR/data/initial-users.json"
    chmod 600 "$INSTALL_DIR/data/initial-users.json"
    print_success "用户初始化完成"

    print_step "10/10" "配置系统服务"

    if [ "$HAS_SYSTEMD" = "true" ]; then
        print_info "正在创建 systemd 服务..."

        cat > /etc/systemd/system/deploypilot.service << EOF
[Unit]
Description=DeployPilot API Server
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=deploypilot
WorkingDirectory=${INSTALL_DIR}
Environment="CONFIG_PATH=${INSTALL_DIR}/config/config.yaml"
ExecStart=${INSTALL_DIR}/bin/api-server --config ${INSTALL_DIR}/config/config.yaml
Restart=always
RestartSec=5
StandardOutput=append:${INSTALL_DIR}/logs/deploypilot.log
StandardError=append:${INSTALL_DIR}/logs/deploypilot.err.log

[Install]
WantedBy=multi-user.target
EOF

        cat > /etc/systemd/system/deploypilot-mcp.service << EOF
[Unit]
Description=DeployPilot MCP Server
After=network.target deploypilot.service

[Service]
Type=simple
User=deploypilot
WorkingDirectory=${INSTALL_DIR}
Environment="CONFIG_PATH=${INSTALL_DIR}/config/config.yaml"
ExecStart=${INSTALL_DIR}/bin/mcp-server --config ${INSTALL_DIR}/config/config.yaml
Restart=always
RestartSec=5
StandardOutput=append:${INSTALL_DIR}/logs/deploypilot-mcp.log
StandardError=append:${INSTALL_DIR}/logs/deploypilot-mcp.err.log

[Install]
WantedBy=multi-user.target
EOF

        systemctl daemon-reload
        systemctl enable deploypilot
        systemctl enable deploypilot-mcp
        print_success "systemd 服务配置成功 (api-server + mcp-server)"
    else
        print_warning "检测到非 systemd 系统，请手动配置服务"
    fi

    configure_firewall "$PORT"

    # Start services
    print_info "正在启动 DeployPilot..."
    systemctl start deploypilot
    sleep 2
    if systemctl is-active --quiet deploypilot; then
        print_success "DeployPilot API Server 启动成功"
    else
        print_warning "DeployPilot API Server 启动失败，请检查日志: ${INSTALL_DIR}/logs/deploypilot.err.log"
    fi

    systemctl start deploypilot-mcp
    sleep 1
    if systemctl is-active --quiet deploypilot-mcp; then
        print_success "DeployPilot MCP Server 启动成功"
    else
        print_warning "DeployPilot MCP Server 启动失败，请检查日志: ${INSTALL_DIR}/logs/deploypilot-mcp.err.log"
    fi

    echo ""
    echo -e "${GREEN}${BOLD}╔═══════════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${GREEN}${BOLD}║                                                               ║${RESET}"
    echo -e "${GREEN}${BOLD}║                    🎉 安装成功完成！                           ║${RESET}"
    echo -e "${GREEN}${BOLD}║                                                               ║${RESET}"
    echo -e "${GREEN}${BOLD}╚═══════════════════════════════════════════════════════════════╝${RESET}"
    echo ""
    echo -e "${BOLD}${WHITE}  访问信息:${RESET}"
    echo -e "  ${CYAN}────────────────────────────────────────────────────────────────${RESET}"
    echo -e "  ${CYAN}面板地址:${RESET}  http://${IP_ADDRESS}:${PORT}"
    echo -e "  ${CYAN}用户名:${RESET}    ${USERNAME}"
    echo -e "  ${CYAN}密码:${RESET}      ${PASSWORD}"
    echo -e "  ${CYAN}版本:${RESET}      ${VERSION:-source}"
    echo ""
    echo -e "${BOLD}${WHITE}  MCP 配置 (AI IDE 集成):${RESET}"
    echo -e "  ${CYAN}────────────────────────────────────────────────────────────────${RESET}"
    echo -e "  ${CYAN}MCP Server 已作为 systemd 服务运行${RESET}"
    echo -e "  ${CYAN}在 AI IDE 中配置以下内容:${RESET}"
    echo ""
    echo '  {'
    echo '    "mcpServers": {'
    echo '      "deploypilot": {'
    echo '        "command": "/opt/deploypilot/bin/mcp-server",'
    echo '        "args": ["--config", "/opt/deploypilot/config/config.yaml"]'
    echo '      }'
    echo '    }'
    echo '  }'
    echo ""
    echo -e "${BOLD}${WHITE}  管理命令:${RESET}"
    echo -e "  ${CYAN}────────────────────────────────────────────────────────────────${RESET}"
    echo -e "  ${CYAN}启动 API:${RESET}    systemctl start deploypilot"
    echo -e "  ${CYAN}启动 MCP:${RESET}    systemctl start deploypilot-mcp"
    echo -e "  ${CYAN}停止服务:${RESET}    systemctl stop deploypilot deploypilot-mcp"
    echo -e "  ${CYAN}重启服务:${RESET}    systemctl restart deploypilot deploypilot-mcp"
    echo -e "  ${CYAN}查看状态:${RESET}    systemctl status deploypilot"
    echo -e "  ${CYAN}查看日志:${RESET}    tail -f ${INSTALL_DIR}/logs/deploypilot.log"
    echo ""
    echo -e "${YELLOW}${BOLD}  ⚠️  安全提示:${RESET}"
    echo -e "  ${YELLOW}────────────────────────────────────────────────────────────────${RESET}"
    echo -e "  ${YELLOW}• 请妥善保管以上登录信息${RESET}"
    echo -e "  ${YELLOW}• 首次登录后请立即修改密码${RESET}"
    echo -e "  ${YELLOW}• 建议配置防火墙仅允许特定 IP 访问${RESET}"
    echo ""
}
main "$@"

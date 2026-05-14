#!/bin/bash
set -euo pipefail
SCRIPT_VERSION="2.5.0"
DEFAULT_INSTALL_DIR="/opt/deploypilot"
DEFAULT_PORT="8080"
GITHUB_REPO="Yogdunana/deploypilot"

# Detect latest version from GitHub (includes prereleases like beta)
get_latest_version() {
    local url="https://api.github.com/repos/${GITHUB_REPO}/releases?per_page=5"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$url" 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"([^"]+)".*/\1/'
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
    echo "    ____             __"
    echo "   / __ \___  ____  / /___  __  __"
    echo "  / / / / _ \/ __ \/ / __ \/ / / /"
    echo " / /_/ /  __/ /_/ / / /_/ / /_/ /"
    echo "/_____/\___/ .___/_/\____/\__, /"
    echo "          /_/            /____/"
    echo ""
    echo "    ____  _ __      __"
    echo "   / __ \(_) /___  / /_"
    echo "  / /_/ / / / __ \/ __/"
    echo " / ____/ / / /_/ / /_"
    echo "/_/   /_/_/\____/\__/"
    echo ""
    echo -e "          ${WHITE}DeployPilot${RESET}"
    echo ""
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
    local result
    result=$(tr -dc 'a-z0-9' < /dev/urandom 2>/dev/null | head -c "$length") || true
    echo -n "$result"
}

function generate_password() {
    local length=${1:-16}
    local result
    result=$(tr -dc 'a-zA-Z0-9!@#$%^&*' < /dev/urandom 2>/dev/null | head -c "$length") || true
    echo -n "$result"
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
    elif [ -f /etc/debian-version ]; then
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

# Check if stdin is a terminal (not a pipe)
function is_tty() {
    [ -t 0 ] || [ -t 1 ]
}

# Auto-detect: if stdin is not a terminal (pipe mode), switch to non-interactive
# This handles: curl | bash, wget -O - | bash, etc.
if [ ! -t 0 ]; then
    NON_INTERACTIVE=true
fi

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

function prompt_password() {
    local prompt="$1"
    local default="$2"
    local result

    if [ "$NON_INTERACTIVE" = true ]; then
        echo "$default"
        return
    fi

    echo -ne "${CYAN}${prompt}${RESET}"
    if [ -n "$default" ]; then
        echo -ne " [${YELLOW}(generated)${RESET}]"
    fi
    echo -ne ": "
    read -s -r result
    echo ""
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
    local max_attempts="${4:-2}"
    
    for attempt in $(seq 1 $max_attempts); do
        if command -v curl >/dev/null 2>&1; then
            # Download with timeout (60s) and retry
            if curl -fsSL --connect-timeout 30 --max-time 300 "$url" -o "$output" 2>/dev/null; then
                return 0
            fi
        elif command -v wget >/dev/null 2>&1; then
            if wget --timeout=300 -q "$url" -O "$output" 2>/dev/null; then
                return 0
            fi
        else
            print_error "需要 curl 或 wget 来下载文件"
            return 1
        fi
        
        if [ $attempt -lt $max_attempts ]; then
            sleep 5
        fi
    done
    
    return 1
}

function install_docker() {
    local os="$1"
    local use_mirror="$2"
    print_info "正在安装 Docker..."
    # Suppress kernel upgrade prompts and needrestart
    export DEBIAN_FRONTEND=noninteractive
    export NEEDRESTART_MODE=a
    if [ -f /etc/needrestart/needrestart.conf ]; then
        sed -i 's/#\$nrconf{restart} = .*/\$nrconf{restart} = "a"/' /etc/needrestart/needrestart.conf 2>/dev/null || true
    fi
    case "$os" in
        ubuntu|debian)
            apt-get update -y
            apt-get install -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" apt-transport-https ca-certificates curl gnupg lsb-release
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

    # Build list of download sources to try
    local urls=()

    if [ "$use_mirror" = "true" ]; then
        # GitHub mirror proxies (most reliable for China)
        urls+=("https://ghfast.top/https://github.com/${GITHUB_REPO}/releases/download/${version}")
        # Gitee Release (backup)
        urls+=("https://gitee.com/${GITHUB_REPO}/releases/download/${version}")
    fi

    # GitHub direct (always try last)
    urls+=("https://github.com/${GITHUB_REPO}/releases/download/${version}")

    print_info "正在下载 DeployPilot v${version} (${arch}) ..."
    local binaries=("api-server" "mcp-server" "deploypilot")
    local download_success=true

    for bin in "${binaries[@]}"; do
        local filename="${bin}-linux-${arch}"
        local output="${dest_dir}/${bin}"
        local bin_downloaded=false

        for url_base in "${urls[@]}"; do
            local url="${url_base}/${filename}"
            print_info "  下载 ${filename} ..."
            print_info "  来源: ${url_base}"

            if download_file "$url" "$output" "${bin}"; then
                chmod +x "$output"
                print_success "  ${bin} 下载完成"
                bin_downloaded=true
                break
            else
                print_warning "  从 ${url_base} 下载失败，尝试下一个源"
            fi
        done

        if [ "$bin_downloaded" = false ]; then
            print_error "  ${bin} 下载失败（所有源均不可用）"
            download_success=false
        fi
    done

    if [ "$download_success" = false ]; then
        print_error "二进制文件下载失败"
        print_info "请检查网络连接或手动下载:"
        print_info "  GitHub: https://github.com/${GITHUB_REPO}/releases"
        print_info "  Gitee:  https://gitee.com/${GITHUB_REPO}/releases"
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
    local bad_chars='[\$\`\|\;\&\>\<\(\)\{\}\[\]\\!~]'
    if [[ "$dir" =~ $bad_chars ]]; then
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

# ─── Upgrade Function ────────────────────────────────────────────────
function do_upgrade() {
    local target_version="${1:-latest}"
    print_banner
    check_root
    echo -e "${BOLD}${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo -e "${BOLD}${MAGENTA} 🔄 DeployPilot 升级模式${RESET}"
    echo -e "${BOLD}${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"

    print_step "1/7" "检测现有安装"
    INSTALL_DIR="${DEFAULT_INSTALL_DIR}"
    if [ -f "${INSTALL_DIR}/config/config.yaml" ]; then
        print_success "检测到已有安装: ${INSTALL_DIR}"
    elif [ -d "${INSTALL_DIR}/bin" ]; then
        print_success "检测到安装目录: ${INSTALL_DIR}"
    else
        print_error "未检测到 DeployPilot 安装 (${INSTALL_DIR} 不存在)"
        print_info "请先运行安装脚本"
        exit 1
    fi

    local current_version="unknown"
    if [ -x "${INSTALL_DIR}/bin/api-server" ]; then
        current_version=$("${INSTALL_DIR}/bin/api-server" --version 2>/dev/null | awk '{print $2}' || echo "unknown")
    fi
    print_info "当前版本: ${current_version}"

    print_step "2/7" "获取目标版本"
    if [ "$target_version" = "latest" ]; then
        target_version=$(get_latest_version)
        if [ -z "$target_version" ]; then
            print_error "无法获取最新版本信息"
            exit 1
        fi
    fi
    print_success "目标版本: ${target_version}"

    if [ "$current_version" = "$target_version" ]; then
        print_warning "当前版本已是 ${target_version}，无需升级"
        exit 0
    fi

    print_step "3/7" "预检查"
    ARCH=$(detect_arch)
    print_info "系统架构: ${ARCH}"

    local available_mb
    available_mb=$(df -BM "${INSTALL_DIR}" | awk 'NR==2 {print $4}' | tr -d 'M')
    if [ "$available_mb" -lt 100 ]; then
        print_error "磁盘空间不足: 仅剩 ${available_mb}MB，至少需要 100MB"
        exit 1
    fi
    print_success "磁盘空间充足: ${available_mb}MB 可用"

    print_step "4/7" "备份当前二进制文件"
    local backup_timestamp
    backup_timestamp=$(date +%Y%m%d-%H%M%S)
    local backup_dir="${INSTALL_DIR}/backups/upgrade-${backup_timestamp}"
    mkdir -p "$backup_dir"
    local backup_count=0
    for binary in api-server mcp-server deploypilot; do
        if [ -f "${INSTALL_DIR}/bin/${binary}" ]; then
            cp -f "${INSTALL_DIR}/bin/${binary}" "${backup_dir}/${binary}"
            backup_count=$((backup_count + 1))
        fi
    done
    if [ -f "${INSTALL_DIR}/config/config.yaml" ]; then
        cp -f "${INSTALL_DIR}/config/config.yaml" "${backup_dir}/config.yaml"
    fi
    print_success "已备份 ${backup_count} 个二进制文件到 ${backup_dir}"

    print_step "5/7" "下载新版本二进制文件"
    local download_base="https://github.com/${GITHUB_REPO}/releases/download/${target_version}"
    local download_failed=false
    for binary in api-server mcp-server deploypilot; do
        local filename="${binary}-linux-${ARCH}"
        local download_url="${download_base}/${filename}"
        local tmp_file="/tmp/${binary}.new"
        print_info "正在下载 ${binary}..."
        if ! download_file "$download_url" "$tmp_file" "$binary"; then
            print_error "下载 ${binary} 失败"
            download_failed=true
            break
        fi
        chmod +x "$tmp_file"
        mv -f "$tmp_file" "${INSTALL_DIR}/bin/${binary}"
        print_success "${binary} 已更新"
    done
    if [ "$download_failed" = true ]; then
        print_error "下载失败，正在回滚..."
        for binary in api-server mcp-server deploypilot; do
            if [ -f "${backup_dir}/${binary}" ]; then
                cp -f "${backup_dir}/${binary}" "${INSTALL_DIR}/bin/${binary}"
            fi
        done
        print_warning "已回滚到之前版本"
        exit 1
    fi

    print_step "6/7" "验证新版本"
    local verify_failed=false
    for binary in api-server mcp-server deploypilot; do
        if [ -x "${INSTALL_DIR}/bin/${binary}" ]; then
            local new_ver
            new_ver=$("${INSTALL_DIR}/bin/${binary}" --version 2>/dev/null | awk '{print $2}' || echo "unknown")
            print_info "${binary}: ${new_ver}"
        else
            print_error "${binary} 不可执行"
            verify_failed=true
        fi
    done
    if [ "$verify_failed" = true ]; then
        print_error "验证失败，正在回滚..."
        for binary in api-server mcp-server deploypilot; do
            if [ -f "${backup_dir}/${binary}" ]; then
                cp -f "${backup_dir}/${binary}" "${INSTALL_DIR}/bin/${binary}"
            fi
        done
        systemctl restart deploypilot deploypilot-mcp 2>/dev/null || true
        print_warning "已回滚到之前版本"
        exit 1
    fi

    print_step "7/7" "重启服务"
    if systemctl is-active --quiet deploypilot-mcp 2>/dev/null; then
        systemctl restart deploypilot-mcp
        print_success "MCP 服务已重启"
    fi
    if systemctl is-active --quiet deploypilot 2>/dev/null; then
        systemctl restart deploypilot
        print_success "API 服务已重启"
    fi

    echo ""
    echo -e "${GREEN}${BOLD}  🎉 升级成功完成！${RESET}"
    echo ""
    echo -e "  ${CYAN}旧版本:${RESET}      ${current_version}"
    echo -e "  ${CYAN}新版本:${RESET}      ${target_version}"
    echo -e "  ${CYAN}备份位置:${RESET}    ${backup_dir}"
    echo ""
}

function main() {
    # Handle --upgrade flag
    if [ "${1:-}" = "--upgrade" ] || [ "${1:-}" = "-U" ]; then
        do_upgrade "${2:-latest}"
        return
    fi

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
    PASSWORD=$(prompt_password "请输入密码" "$(generate_password)")

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
    echo -e "${BOLD}${WHITE}  安装配置摘要${RESET}"
    echo -e "  ${CYAN}安装路径:${RESET}    ${INSTALL_DIR}"
    echo -e "  ${CYAN}端口:${RESET}        ${PORT}"
    echo -e "  ${CYAN}用户名:${RESET}      ${USERNAME}"
    echo -e "  ${CYAN}密码:${RESET}        ${PASSWORD}"
    echo -e "  ${CYAN}版本:${RESET}        ${VERSION:-source}"
    echo -e "  ${CYAN}国内镜像:${RESET}    ${USE_MIRROR}"
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
        exit 1
    fi

    print_step "8/10" "创建配置文件"
    print_info "正在创建配置文件..."

    JWT_SECRET=$(random_string 32)
    API_KEY_SALT=$(random_string 32)
    SSH_KNOWN_HOSTS="${INSTALL_DIR}/data/known_hosts"

    cat > "$INSTALL_DIR/config/config.yaml" << EOF
server:
  port: ${PORT}
  host: 0.0.0.0

database:
  type: sqlite
  dsn: ${INSTALL_DIR}/data/deploypilot.db

auth:
  jwt_secret: ${JWT_SECRET}
  token_expire: 24h
  api_key_salt: ${API_KEY_SALT}

security:
  ssh_known_hosts_path: "${SSH_KNOWN_HOSTS}"
  ssh_strict_host_key_checking: false
EOF

    touch "${SSH_KNOWN_HOSTS}"
    chmod 600 "${SSH_KNOWN_HOSTS}"
    chown deploypilot:deploypilot "${SSH_KNOWN_HOSTS}" 2>/dev/null || true
    chmod 600 "$INSTALL_DIR/config/config.yaml"
    print_success "配置文件创建成功"

    print_step "9/10" "创建用户和初始化"
    print_info "正在创建系统用户..."
    if ! id deploypilot >/dev/null 2>&1; then
        useradd -m -s /bin/bash -d "$INSTALL_DIR" deploypilot
    fi
    usermod -aG docker deploypilot
    chown -R deploypilot:deploypilot "$INSTALL_DIR"

    print_step "10/10" "配置系统服务"
    # Create CLI symlink
    ln -sf ${INSTALL_DIR}/bin/deploypilot /usr/local/bin/deploypilot
    if [ "$HAS_SYSTEMD" = "true" ]; then
        print_info "正在创建 systemd 服务..."

        cat > /etc/systemd/system/deploypilot.service << EOF
[Unit]
Description=DeployPilot API Server
After=network.target
Wants=docker.service

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
# MCP server uses stdio transport - it exits when idle, so don't auto-restart
Restart=no
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
    sleep 3

    if systemctl is-active --quiet deploypilot; then
        print_success "DeployPilot API Server 启动成功"

        DB_FILE="${INSTALL_DIR}/data/deploypilot.db"
        if [ -f "$DB_FILE" ]; then
            # Reinstall detected
            echo ""
            print_warning "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            print_warning "  检测到已有 DeployPilot 数据"
            print_warning "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            echo ""
            echo "  请选择操作："
            echo "    1) 重置管理员密码（保留所有数据）"
            echo "    2) 清空所有数据，重新开始"
            echo "    3) 保留现有数据，不修改"
            echo ""
            REINSTALL_CHOICE=$(prompt_input "请输入选项 (1/2/3)" "1")

            case "$REINSTALL_CHOICE" in
                1)
                    print_info "请输入要重置密码的用户名（直接回车使用默认: admin）"
                    RESET_USERNAME=$(prompt_input "用户名" "admin")
                    print_info "正在重置用户 '${RESET_USERNAME}' 的密码..."
                    RESET_RESP=$(curl -s -X POST "http://localhost:${PORT}/api/v1/auth/reset-password" \
                        -H "Content-Type: application/json" \
                        -d "{\"username\": \"${RESET_USERNAME}\", \"password\": \"${PASSWORD}\"}" 2>&1)
                    if echo "$RESET_RESP" | grep -q 'password_reset'; then
                        print_success "管理员密码重置成功"
                        SKIP_SERVER_REG=0
                    elif echo "$RESET_RESP" | grep -q 'user_not_found\|not found'; then
                        print_warning "用户 '${RESET_USERNAME}' 不存在"
                        print_info "请检查用户名后重试，或选择选项 2 清空数据重新开始"
                        SKIP_SERVER_REG=1
                    else
                        print_warning "密码重置失败: $(echo "$RESET_RESP" | grep -o '"message":"[^"]*"' | cut -d'"' -f4)"
                        print_info "请手动登录面板修改密码"
                        SKIP_SERVER_REG=1
                    fi
                    ;;
                2)
                    print_warning "正在清空所有数据..."
                    systemctl stop deploypilot
                    rm -f "$DB_FILE"
                    rm -f "${DB_FILE}-wal" "${DB_FILE}-shm"
                    systemctl start deploypilot
                    sleep 2
                    print_info "正在创建管理员账号..."
                    REGISTER_RESP=$(curl -s -X POST "http://localhost:${PORT}/api/v1/auth/register" \
                        -H "Content-Type: application/json" \
                        -d "{\"username\": \"${USERNAME}\", \"email\": \"admin@example.com\", \"password\": \"${PASSWORD}\"}" 2>&1)
                    if echo "$REGISTER_RESP" | grep -q '"id"'; then
                        print_success "管理员账号创建成功"
                        SKIP_SERVER_REG=0
                    else
                        print_warning "管理员创建失败: $(echo "$REGISTER_RESP" | grep -o '"message":"[^"]*"' | cut -d'"' -f4)"
                        SKIP_SERVER_REG=1
                    fi
                    ;;
                3)
                    print_info "保留现有数据，跳过管理员配置"
                    print_info "请使用已有账号登录"
                    SKIP_SERVER_REG=1
                    ;;
                *)
                    print_warning "无效选项，保留现有数据"
                    SKIP_SERVER_REG=1
                    ;;
            esac
        else
            # Fresh install
            print_info "正在创建管理员账号..."
            REGISTER_RESP=$(curl -s -X POST "http://localhost:${PORT}/api/v1/auth/register" \
                -H "Content-Type: application/json" \
                -d "{\"username\": \"${USERNAME}\", \"email\": \"admin@example.com\", \"password\": \"${PASSWORD}\"}" 2>&1)
            if echo "$REGISTER_RESP" | grep -q '"id"'; then
                print_success "管理员账号创建成功"
            else
                print_warning "管理员账号创建失败: $(echo "$REGISTER_RESP" | grep -o '"message":"[^"]*"' | cut -d'"' -f4)"
                print_info "请手动创建账号或使用已有账号: http://${IP_ADDRESS}:${PORT}"
                SKIP_SERVER_REG=1
            fi
        fi
    else
        print_warning "DeployPilot API Server 启动失败，请检查日志: ${INSTALL_DIR}/logs/deploypilot.err.log"
    fi

    # MCP server uses stdio transport - it exits when idle, so we don't start it as a service
    # Users should configure their AI IDE to run the MCP server directly
    print_info "MCP Server 配置: 请在 AI IDE 中配置 MCP server 路径"

    # Register this server as a DeployPilot node
    if systemctl is-active --quiet deploypilot && [ "${SKIP_SERVER_REG:-0}" != "1" ]; then
        print_info "正在注册当前服务器..."
        # Login to get JWT token
        LOGIN_RESP=$(curl -s -X POST "http://localhost:${PORT}/api/v1/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"username\": \"${USERNAME}\", \"password\": \"${PASSWORD}\"}" 2>&1)
        TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
        if [ -n "$TOKEN" ]; then
            SERVER_NAME=$(hostname -s 2>/dev/null || echo "deploy-server")
            SERVER_RESP=$(curl -s -X POST "http://localhost:${PORT}/api/v1/servers" \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer ${TOKEN}" \
                -d "{\"name\": \"${SERVER_NAME}\", \"host\": \"127.0.0.1\", \"port\": 22, \"user\": \"root\"}" 2>&1)
            if echo "$SERVER_RESP" | grep -q '"id"'; then
                print_success "当前服务器已注册为 DeployPilot 节点 (${SERVER_NAME})"
            else
                print_warning "服务器注册失败（可能已注册）"
            fi
        else
            print_warning "无法登录以注册服务器（请使用已有账号登录面板添加）"
        fi
    fi

    echo ""
    echo -e "${GREEN}${BOLD}  🎉 安装成功完成！${RESET}"
    echo ""
    echo -e "${BOLD}${WHITE}  访问信息:${RESET}"
    echo -e "  ${CYAN}面板地址:${RESET}  http://${IP_ADDRESS}:${PORT}"
    echo -e "  ${CYAN}用户名:${RESET}    ${USERNAME}"
    echo -e "  ${CYAN}密码:${RESET}      ${PASSWORD}"
    echo -e "  ${CYAN}版本:${RESET}      ${VERSION:-source}"
    echo ""
    echo -e "${BOLD}${WHITE}  MCP 配置 (AI IDE 集成):${RESET}"
    echo -e "  ${CYAN}MCP Server 通过 SSE 传输运行${RESET}"
    echo -e "  ${CYAN}SSE 地址:${RESET}  http://${IP_ADDRESS}:${PORT}/api/v1/mcp/sse"
    echo ""
    echo -e "${BOLD}${WHITE}  管理命令:${RESET}"
    echo -e "  ${CYAN}启动服务:${RESET}    systemctl start deploypilot"
    echo -e "  ${CYAN}停止服务:${RESET}    systemctl stop deploypilot"
    echo -e "  ${CYAN}重启服务:${RESET}    systemctl restart deploypilot"
    echo -e "  ${CYAN}查看状态:${RESET}    deploypilot status"
    echo -e "  ${CYAN}查看日志:${RESET}    tail -f ${INSTALL_DIR}/logs/deploypilot.log"
    echo ""
    echo -e "${YELLOW}${BOLD}  ⚠️  安全提示:${RESET}"
    echo -e "  ${YELLOW}• 请妥善保管以上登录信息${RESET}"
    echo -e "  ${YELLOW}• 首次登录后请立即修改密码${RESET}"
    echo -e "  ${YELLOW}• 建议配置防火墙仅允许特定 IP 访问${RESET}"
    echo ""
}

main "$@"

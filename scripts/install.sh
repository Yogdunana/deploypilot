#!/bin/bash
# DeployPilot 一键安装脚本
# Usage: curl -fsSL https://raw.githubusercontent.com/Yogdunana/deploypilot/main/scripts/install.sh | bash
set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== DeployPilot 一键安装脚本 ===${NC}"
echo -e "${BLUE}=================================${NC}"

# 检查系统
echo -e "${YELLOW}[1/8] 检查系统环境...${NC}"
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
    VERSION=$VERSION_ID
    echo -e "${GREEN}  检测到系统: ${OS} ${VERSION}${NC}"
else
    echo -e "${RED}  无法检测系统类型${NC}"
    exit 1
fi

# 检查架构
ARCH=$(uname -m)
echo -e "${GREEN}  系统架构: ${ARCH}${NC}"

# 安装依赖
echo -e "${YELLOW}[2/8] 安装依赖...${NC}"
case "$OS" in
    ubuntu|debian)
        sudo apt update && sudo apt install -y curl wget unzip systemd git
        ;;
    centos|rhel|rocky|almalinux)
        sudo yum install -y curl wget unzip systemd git
        ;;
    *)
        echo -e "${RED}  不支持的系统类型${NC}"
        exit 1
        ;;
esac

# 检查Docker
echo -e "${YELLOW}[3/8] 检查Docker...${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${GREEN}  安装Docker...${NC}"
    curl -fsSL https://get.docker.com | sh
    sudo usermod -aG docker $USER
    echo -e "${YELLOW}  Docker已安装，请重新登录以应用组更改${NC}"
else
    echo -e "${GREEN}  Docker已安装${NC}"
    docker --version
fi

# 创建用户和目录
echo -e "${YELLOW}[4/8] 创建用户和目录...${NC}"
sudo useradd -m -s /bin/bash deploypilot 2>/dev/null || true
sudo usermod -aG docker deploypilot
sudo mkdir -p /opt/deploypilot/{bin,data,logs,backups}
sudo chown -R deploypilot:deploypilot /opt/deploypilot

# 下载DeployPilot
echo -e "${YELLOW}[5/8] 下载DeployPilot...${NC}"
# 从GitHub仓库克隆代码
echo -e "${GREEN}  从GitHub克隆代码...${NC}"
cd /tmp
git clone https://github.com/Yogdunana/deploypilot.git
echo -e "${GREEN}  编译中...${NC}"
cd deploypilot
make build
sudo cp bin/* /opt/deploypilot/bin/
sudo chown deploypilot:deploypilot /opt/deploypilot/bin/*

# 生成配置
echo -e "${YELLOW}[6/8] 生成配置...${NC}"
# 生成随机密码
ADMIN_PASSWORD=$(openssl rand -base64 12)
JWT_SECRET=$(openssl rand -base64 32)

# 创建配置文件
sudo tee /opt/deploypilot/data/config.yaml > /dev/null << EOF
server:
  host: 0.0.0.0
  port: 8080
  ssl:
    enabled: false

auth:
  jwt_secret: "$JWT_SECRET"
  admin:
    username: "admin"
    password: "$ADMIN_PASSWORD"

database:
  type: sqlite
  path: /opt/deploypilot/data/deploypilot.db

logs:
  level: info
  path: /opt/deploypilot/logs
EOF

sudo chown deploypilot:deploypilot /opt/deploypilot/data/config.yaml

# 配置systemd服务
echo -e "${YELLOW}[7/8] 配置systemd服务...${NC}"
sudo tee /etc/systemd/system/deploypilot.service > /dev/null << 'EOF'
[Unit]
Description=DeployPilot Server
After=docker.service
Requires=docker.service

[Service]
Type=simple
User=deploypilot
WorkingDirectory=/opt/deploypilot
ExecStart=/opt/deploypilot/bin/api-server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable deploypilot
sudo systemctl start deploypilot

# 放行端口
echo -e "${YELLOW}[8/8] 放行端口...${NC}"
# 检查是否有防火墙
if command -v ufw &> /dev/null; then
    sudo ufw allow 8080/tcp
    echo -e "${GREEN}  已通过ufw放行8080端口${NC}"
elif command -v firewall-cmd &> /dev/null; then
    sudo firewall-cmd --permanent --add-port=8080/tcp
    sudo firewall-cmd --reload
    echo -e "${GREEN}  已通过firewall-cmd放行8080端口${NC}"
fi

# 检查1Panel
if command -v bt &> /dev/null; then
    echo -e "${GREEN}  检测到1Panel，正在同步防火墙规则...${NC}"
    # 这里可以添加1Panel API调用
fi

# 获取服务器IP
SERVER_IP=$(curl -s ifconfig.me || echo "127.0.0.1")

# 等待服务启动
echo -e "${YELLOW}  等待服务启动...${NC}"
sleep 5

# 检查服务状态
if sudo systemctl is-active --quiet deploypilot; then
    echo -e "${GREEN}  DeployPilot服务已成功启动${NC}"
else
    echo -e "${RED}  DeployPilot服务启动失败${NC}"
    echo -e "${YELLOW}  请查看日志: sudo journalctl -u deploypilot${NC}"
fi

# 输出安装结果
echo -e "${BLUE}=================================${NC}"
echo -e "${GREEN}=== 安装完成 ===${NC}"
echo -e "${BLUE}=================================${NC}"
echo -e "${GREEN}访问地址: http://${SERVER_IP}:8080${NC}"
echo -e "${GREEN}用户名: admin${NC}"
echo -e "${GREEN}密码: ${ADMIN_PASSWORD}${NC}"
echo -e "${BLUE}=================================${NC}"
echo -e "${YELLOW}注意事项:${NC}"
echo -e "  1. 首次登录后请修改密码"
echo -e "  2. 确保8080端口已在防火墙中放行"
echo -e "  3. 如需修改配置，请编辑 /opt/deploypilot/data/config.yaml"
echo -e "  4. 查看日志: sudo journalctl -u deploypilot"
echo -e "${BLUE}=================================${NC}"

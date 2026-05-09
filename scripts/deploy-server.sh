#!/bin/bash
# deploy-server.sh - 一键在目标服务器上安装 DeployPilot
# Usage: ./deploy-server.sh <server-host> <ssh-user> [ssh-port]
set -euo pipefail

HOST="${1:?Usage: $0 <host> <user> [port]}"
USER="${2:?Usage: $0 <host> <user> [port]}"
PORT="${3:-22}"

echo "=== DeployPilot Server Setup ==="
echo "Host: ${USER}@${HOST}:${PORT}"

# Step 1: Check prerequisites
ssh -p "$PORT" "${USER}@${HOST}" bash -s << 'REMOTE'
set -euo pipefail

echo "[1/5] Checking prerequisites..."
command -v docker >/dev/null 2>&1 || {
  echo "Installing Docker..."
  curl -fsSL https://get.docker.com | sh
  sudo usermod -aG docker "$USER"
  echo "Docker installed. Please log out and back in for group changes."
}
docker --version

command -v go >/dev/null 2>&1 || {
  echo "Installing Go 1.23..."
  curl -fsSL https://go.dev/dl/go1.23.0.linux-amd64.tar.gz | sudo tar -C /usr/local -xzf -
  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
  export PATH=$PATH:/usr/local/go/bin
}
go version

echo "[2/5] Creating deploy user..."
sudo useradd -m -s /bin/bash deploypilot 2>/dev/null || true
sudo usermod -aG docker deploypilot

echo "[3/5] Creating directories..."
sudo mkdir -p /opt/deploypilot/{data,logs,backups}
sudo chown -R deploypilot:deploypilot /opt/deploypilot

echo "[4/5] Setting up systemd service (placeholder)..."
sudo tee /etc/systemd/system/deploypilot.service > /dev/null << 'SERVICE'
[Unit]
Description=DeployPilot MCP Server
After=docker.service
Requires=docker.service

[Service]
Type=simple
User=deploypilot
WorkingDirectory=/opt/deploypilot
ExecStart=/opt/deploypilot/bin/mcp-server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
SERVICE

echo "[5/5] Server ready!"
echo "  Data dir: /opt/deploypilot/data"
echo "  Logs dir: /opt/deploypilot/logs"
echo "  Backup dir: /opt/deploypilot/backups"
REMOTE

echo ""
echo "=== Server setup complete ==="
echo "Next: scp bin/mcp-server ${USER}@${HOST}:/opt/deploypilot/bin/"

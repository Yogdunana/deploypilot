# Getting Started

## Prerequisites

- A server with Docker 20.10+ installed
- (Optional) A domain name for your applications

## Docker Deployment (Recommended)

```bash
# Clone the repository
git clone https://github.com/Yogdunana/deploypilot.git
cd deploypilot

# Start with Docker Compose
docker compose up -d

# View logs
docker compose logs -f
```

Access the web dashboard at `http://your-server-ip:8080`.

## First-Time Setup

1. **Register** — Create an admin account
2. **Login** — Access the dashboard
3. **Add Server** — Register your server with SSH credentials
4. **Add Credentials** — Store encrypted SSH keys or passwords
5. **Deploy** — Use the web UI or MCP tools to deploy applications

## Binary Installation

Download the latest release from [GitHub Releases](https://github.com/Yogdunana/deploypilot/releases):

```bash
# Download binary (Linux amd64)
wget https://github.com/Yogdunana/deploypilot/releases/latest/download/api-server-linux-amd64 -O api-server
chmod +x api-server

# Set encryption key
export DEPLOYPILOT_ENCRYPTION_KEY=$(openssl rand -base64 32)

# Run
./api-server
```

## One-Line Install Script

```bash
curl -fsSL https://raw.githubusercontent.com/Yogdunana/deploypilot/main/scripts/install.sh | bash
```

## Next Steps

- [[MCP-Integration]] — Connect your AI IDE
- [[Configuration]] — Customize your deployment

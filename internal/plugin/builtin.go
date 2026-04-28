package plugin

import (
	"fmt"

	"github.com/Yogdunana/deploypilot/internal/provider/cicd"
	"github.com/Yogdunana/deploypilot/internal/provider/dns"
	"github.com/Yogdunana/deploypilot/internal/provider/notify"
	"github.com/Yogdunana/deploypilot/internal/provider/registry"
	"github.com/Yogdunana/deploypilot/internal/provider/server"
)

// RegisterBuiltinPlugins registers all existing providers as built-in plugins.
// This wraps existing provider constructors in the plugin factory pattern.
func RegisterBuiltinPlugins(r *Registry) error {
	// DNS providers
	dnsPlugins := []*PluginDescriptor{
		{
			Name:        "cloudflare-dns",
			DisplayName: "Cloudflare DNS",
			Version:     "1.0.0",
			Description: "Cloudflare DNS provider for managing DNS records",
			Author:      "DeployPilot",
			Provider:    "dns",
			Type:        "cloudflare",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				apiToken, _ := cfg["api_token"].(string)
				accountEmail, _ := cfg["account_email"].(string)
				if apiToken == "" {
					return nil, fmt.Errorf("cloudflare dns: api_token is required")
				}
				return dns.NewCloudflareProvider(apiToken, accountEmail), nil
			},
		},
		{
			Name:        "alidns",
			DisplayName: "Aliyun DNS",
			Version:     "1.0.0",
			Description: "Alibaba Cloud DNS (Alidns) provider",
			Author:      "DeployPilot",
			Provider:    "dns",
			Type:        "alidns",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				accessKeyID, _ := cfg["access_key_id"].(string)
				accessKeySecret, _ := cfg["access_key_secret"].(string)
				if accessKeyID == "" || accessKeySecret == "" {
					return nil, fmt.Errorf("alidns: access_key_id and access_key_secret are required")
				}
				return dns.NewAliyunProvider(accessKeyID, accessKeySecret), nil
			},
		},
		{
			Name:        "tencentcloud-dns",
			DisplayName: "Tencent Cloud DNSPod",
			Version:     "1.0.0",
			Description: "Tencent Cloud DNSPod DNS provider",
			Author:      "DeployPilot",
			Provider:    "dns",
			Type:        "tencentcloud",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				secretID, _ := cfg["secret_id"].(string)
				secretKey, _ := cfg["secret_key"].(string)
				if secretID == "" || secretKey == "" {
					return nil, fmt.Errorf("tencentcloud dns: secret_id and secret_key are required")
				}
				return dns.NewTencentProvider(secretID, secretKey), nil
			},
		},
		{
			Name:        "westdns",
			DisplayName: "WestDNS",
			Version:     "1.0.0",
			Description: "WestDNS (西部数码) DNS provider",
			Author:      "DeployPilot",
			Provider:    "dns",
			Type:        "westdns",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				apiKey, _ := cfg["api_key"].(string)
				apiSecret, _ := cfg["api_secret"].(string)
				if apiKey == "" || apiSecret == "" {
					return nil, fmt.Errorf("westdns: api_key and api_secret are required")
				}
				return dns.NewWestDNSProvider(apiKey, apiSecret), nil
			},
		},
	}

	// Notify providers
	notifyPlugins := []*PluginDescriptor{
		{
			Name:        "webhook-notify",
			DisplayName: "Webhook",
			Version:     "1.0.0",
			Description: "Generic webhook notification provider",
			Author:      "DeployPilot",
			Provider:    "notify",
			Type:        "webhook",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				url, _ := cfg["url"].(string)
				if url == "" {
					return nil, fmt.Errorf("webhook notify: url is required")
				}
				var headers map[string]string
				if h, ok := cfg["headers"].(map[string]string); ok {
					headers = h
				}
				return notify.NewWebhookNotifier(url, headers), nil
			},
		},
		{
			Name:        "email-notify",
			DisplayName: "Email",
			Version:     "1.0.0",
			Description: "Email notification provider via SMTP",
			Author:      "DeployPilot",
			Provider:    "notify",
			Type:        "email",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				smtpHost, _ := cfg["smtp_host"].(string)
				smtpPort, _ := cfg["smtp_port"].(int)
				username, _ := cfg["username"].(string)
				password, _ := cfg["password"].(string)
				from, _ := cfg["from"].(string)
				if smtpHost == "" {
					return nil, fmt.Errorf("email notify: smtp_host is required")
				}
				if smtpPort == 0 {
					smtpPort = 587
				}
				return notify.NewEmailNotifier(notify.EmailConfig{
					SMTPHost: smtpHost,
					SMTPPort: smtpPort,
					Username: username,
					Password: password,
					From:     from,
				}), nil
			},
		},
		{
			Name:        "dingtalk-notify",
			DisplayName: "DingTalk",
			Version:     "1.0.0",
			Description: "DingTalk (钉钉) notification provider",
			Author:      "DeployPilot",
			Provider:    "notify",
			Type:        "dingtalk",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				webhookURL, _ := cfg["webhook_url"].(string)
				secret, _ := cfg["secret"].(string)
				if webhookURL == "" {
					return nil, fmt.Errorf("dingtalk notify: webhook_url is required")
				}
				return notify.NewDingTalkNotifier(webhookURL, secret), nil
			},
		},
		{
			Name:        "feishu-notify",
			DisplayName: "Feishu",
			Version:     "1.0.0",
			Description: "Feishu (飞书/Lark) notification provider",
			Author:      "DeployPilot",
			Provider:    "notify",
			Type:        "feishu",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				webhookURL, _ := cfg["webhook_url"].(string)
				if webhookURL == "" {
					return nil, fmt.Errorf("feishu notify: webhook_url is required")
				}
				return notify.NewFeishuNotifier(webhookURL), nil
			},
		},
		{
			Name:        "telegram-notify",
			DisplayName: "Telegram",
			Version:     "1.0.0",
			Description: "Telegram Bot notification provider",
			Author:      "DeployPilot",
			Provider:    "notify",
			Type:        "telegram",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				botToken, _ := cfg["bot_token"].(string)
				chatID, _ := cfg["chat_id"].(string)
				if botToken == "" || chatID == "" {
					return nil, fmt.Errorf("telegram notify: bot_token and chat_id are required")
				}
				return notify.NewTelegramNotifier(botToken, chatID), nil
			},
		},
		{
			Name:        "wecom-notify",
			DisplayName: "WeCom",
			Version:     "1.0.0",
			Description: "WeCom (企业微信) notification provider",
			Author:      "DeployPilot",
			Provider:    "notify",
			Type:        "wecom",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				webhookURL, _ := cfg["webhook_url"].(string)
				if webhookURL == "" {
					return nil, fmt.Errorf("wecom notify: webhook_url is required")
				}
				return notify.NewWeComNotifier(webhookURL), nil
			},
		},
	}

	// Registry providers
	registryPlugins := []*PluginDescriptor{
		{
			Name:        "docker-hub-registry",
			DisplayName: "Docker Hub",
			Version:     "1.0.0",
			Description: "Docker Hub container registry provider",
			Author:      "DeployPilot",
			Provider:    "registry",
			Type:        "docker_hub",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				url, _ := cfg["url"].(string)
				username, _ := cfg["username"].(string)
				password, _ := cfg["password"].(string)
				return registry.NewDockerHubProvider(url, username, password), nil
			},
		},
		{
			Name:        "ghcr-registry",
			DisplayName: "GitHub Container Registry",
			Version:     "1.0.0",
			Description: "GitHub Container Registry (GHCR) provider",
			Author:      "DeployPilot",
			Provider:    "registry",
			Type:        "ghcr",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				url, _ := cfg["url"].(string)
				username, _ := cfg["username"].(string)
				password, _ := cfg["password"].(string)
				return registry.NewGHCRProvider(url, username, password), nil
			},
		},
	}

	// CICD providers
	cicdPlugins := []*PluginDescriptor{
		{
			Name:        "github-actions-cicd",
			DisplayName: "GitHub Actions",
			Version:     "1.0.0",
			Description: "GitHub Actions CI/CD provider",
			Author:      "DeployPilot",
			Provider:    "cicd",
			Type:        "github_actions",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				token, _ := cfg["token"].(string)
				owner, _ := cfg["owner"].(string)
				if token == "" || owner == "" {
					return nil, fmt.Errorf("github actions cicd: token and owner are required")
				}
				return cicd.NewGitHubActionsProvider(token, owner), nil
			},
		},
		{
			Name:        "gitea-actions-cicd",
			DisplayName: "Gitea Actions",
			Version:     "1.0.0",
			Description: "Gitea Actions CI/CD provider",
			Author:      "DeployPilot",
			Provider:    "cicd",
			Type:        "gitea_actions",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				token, _ := cfg["token"].(string)
				owner, _ := cfg["owner"].(string)
				baseURL, _ := cfg["base_url"].(string)
				if token == "" || owner == "" {
					return nil, fmt.Errorf("gitea actions cicd: token and owner are required")
				}
				return cicd.NewGiteaActionsProvider(token, owner, baseURL), nil
			},
		},
	}

	// Server (Panel) providers
	serverPlugins := []*PluginDescriptor{
		{
			Name:        "1panel-server",
			DisplayName: "1Panel",
			Version:     "1.0.0",
			Description: "1Panel hosting panel provider",
			Author:      "DeployPilot",
			Provider:    "server",
			Type:        "1panel",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				baseURL, _ := cfg["base_url"].(string)
				apiKey, _ := cfg["api_key"].(string)
				if baseURL == "" || apiKey == "" {
					return nil, fmt.Errorf("1panel: base_url and api_key are required")
				}
				return server.NewPanel1Client(baseURL, apiKey), nil
			},
		},
		{
			Name:        "btpanel-server",
			DisplayName: "BT-Panel (宝塔)",
			Version:     "1.0.0",
			Description: "BT Panel hosting panel provider",
			Author:      "DeployPilot",
			Provider:    "server",
			Type:        "bt-panel",
			Factory: func(cfg map[string]interface{}) (interface{}, error) {
				baseURL, _ := cfg["base_url"].(string)
				apiKey, _ := cfg["api_key"].(string)
				if baseURL == "" || apiKey == "" {
					return nil, fmt.Errorf("btpanel: base_url and api_key are required")
				}
				return server.NewBTPanelClient(baseURL, "admin", apiKey), nil
			},
		},
	}

	// Register all plugins
	allPlugins := append(append(append(append(dnsPlugins, notifyPlugins...), registryPlugins...), cicdPlugins...), serverPlugins...)
	for _, desc := range allPlugins {
		if err := r.Register(desc); err != nil {
			return fmt.Errorf("failed to register built-in plugin %s: %w", desc.Name, err)
		}
	}

	return nil
}

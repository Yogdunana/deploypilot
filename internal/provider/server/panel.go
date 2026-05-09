package server

import (
	"context"
	"fmt"
	"log/slog"
)

// PanelType represents the type of hosting panel.
type PanelType string

// CommandExecutor abstracts command execution for panel detection.
type CommandExecutor interface {
	RunCommand(ctx context.Context, cmd string) (string, error)
}

// WebsiteInfo represents basic information about a website managed by a panel.
type WebsiteInfo struct {
	ID            interface{} `json:"id"`
	PrimaryDomain string      `json:"primaryDomain"`
	Type          string      `json:"type"`
	Status        bool        `json:"status"`
	Remark        string      `json:"remark"`
	SSL           bool        `json:"ssl"`
	CreatedAt     string      `json:"createdAt"`
}

// PanelClient defines the unified interface for panel providers.
type PanelClient interface {
	OpenFirewall(ctx context.Context, port int, protocol string) error
	CloseFirewall(ctx context.Context, port int, protocol string) error
	CreateReverseProxy(ctx context.Context, domain, targetURL string, port int) error
	DeleteReverseProxy(ctx context.Context, domain string) error
	CreateWebsite(ctx context.Context, domain, websiteType, remark string) (*WebsiteInfo, error)
	GetWebsiteList(ctx context.Context) ([]WebsiteInfo, error)
	GetInfo() map[string]interface{}
}

const (
	PanelNone    PanelType = "none"
	Panel1Panel  PanelType = "1panel"
	PanelBTPanel PanelType = "bt-panel"
)

// PanelProvider manages hosting panel integrations.
type PanelProvider struct {
	client PanelClient
}

// NewPanelProvider creates a panel provider with the given panel client.
func NewPanelProvider(client PanelClient) *PanelProvider {
	return &PanelProvider{client: client}
}

// DetectPanel attempts to detect which panel is installed and returns a PanelClient.
func DetectPanel(ctx context.Context, executor CommandExecutor) PanelClient {
	// Try 1Panel
	output, err := executor.RunCommand(ctx, "systemctl is-active 1panel 2>/dev/null || echo 'inactive'")
	if err == nil && len(output) > 0 && output != "inactive" {
		slog.Info("detected 1Panel", "status", output)
		return NewPanel1Client("", "")
	}
	// Try BT-Panel
	output, err = executor.RunCommand(ctx, "systemctl is-active bt 2>/dev/null || echo 'inactive'")
	if err == nil && len(output) > 0 && output != "inactive" {
		slog.Info("detected BT-Panel", "status", output)
		return NewBTPanelClient("", "admin", "")
	}
	// Fallback: check processes
	output, err = executor.RunCommand(ctx, "ps aux | grep -E '1panel|BT-Panel|bt' | grep -v grep || true")
	if err == nil && len(output) > 0 {
		if contains(output, "1panel") {
			return NewPanel1Client("", "")
		}
		if contains(output, "BT-Panel") || contains(output, "bt") {
			return NewBTPanelClient("", "admin", "")
		}
	}
	return nil
}

// GetPanelInfo returns information about the detected panel.
func (p *PanelProvider) GetPanelInfo() (map[string]interface{}, error) {
	if p.client == nil {
		return nil, fmt.Errorf("no panel detected")
	}
	info := p.client.GetInfo()
	info["type"] = "detected"
	return info, nil
}

// OpenFirewall opens a port on the panel's firewall.
func (p *PanelProvider) OpenFirewall(ctx context.Context, port int, protocol string) error {
	if p.client == nil {
		slog.Warn("no panel detected, skipping firewall operation (use SSH fallback)", "port", port, "protocol", protocol)
		return nil
	}
	return p.client.OpenFirewall(ctx, port, protocol)
}

// CloseFirewall closes a port on the panel's firewall.
func (p *PanelProvider) CloseFirewall(ctx context.Context, port int, protocol string) error {
	if p.client == nil {
		slog.Warn("no panel detected, skipping firewall operation (use SSH fallback)", "port", port, "protocol", protocol)
		return nil
	}
	return p.client.CloseFirewall(ctx, port, protocol)
}

// CreateReverseProxy creates a reverse proxy via the panel's API.
func (p *PanelProvider) CreateReverseProxy(ctx context.Context, domain, targetURL string, port int) error {
	if p.client == nil {
		slog.Warn("no panel detected, skipping reverse proxy creation (use SSH fallback)", "domain", domain)
		return nil
	}
	return p.client.CreateReverseProxy(ctx, domain, targetURL, port)
}

// DeleteReverseProxy deletes a reverse proxy via the panel's API.
func (p *PanelProvider) DeleteReverseProxy(ctx context.Context, domain string) error {
	if p.client == nil {
		slog.Warn("no panel detected, skipping reverse proxy deletion (use SSH fallback)", "domain", domain)
		return nil
	}
	return p.client.DeleteReverseProxy(ctx, domain)
}

// CreateWebsite creates a website via the panel's API.
func (p *PanelProvider) CreateWebsite(ctx context.Context, domain, websiteType, remark string) (*WebsiteInfo, error) {
	if p.client == nil {
		slog.Warn("no panel detected, skipping website creation", "domain", domain)
		return nil, fmt.Errorf("no panel detected")
	}
	return p.client.CreateWebsite(ctx, domain, websiteType, remark)
}

// GetWebsiteList returns a list of websites managed by the panel.
func (p *PanelProvider) GetWebsiteList(ctx context.Context) ([]WebsiteInfo, error) {
	if p.client == nil {
		slog.Warn("no panel detected, skipping website list")
		return nil, fmt.Errorf("no panel detected")
	}
	return p.client.GetWebsiteList(ctx)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && search(s, substr)
}

func search(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

package server

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// PanelType represents the type of hosting panel.
type PanelType string

// CommandExecutor abstracts command execution for panel detection.
type CommandExecutor interface {
	RunCommand(ctx context.Context, cmd string) (string, error)
}

const (
	PanelNone    PanelType = "none"
	Panel1Panel  PanelType = "1panel"
	PanelBTPanel PanelType = "bt-panel"
)

// PanelProvider manages hosting panel integrations.
type PanelProvider struct {
	panelType PanelType
	baseURL   string
	apiKey    string

	// Lazy-initialized panel clients
	panel1Once  sync.Once
	panel1      *Panel1Client
	panel1Err   error
	btPanelOnce sync.Once
	btPanel     *BTPanelClient
	btPanelErr  error
}

// NewPanelProvider creates a panel provider based on the detected panel type.
func NewPanelProvider(panelType PanelType, baseURL, apiKey string) *PanelProvider {
	return &PanelProvider{
		panelType: panelType,
		baseURL:   baseURL,
		apiKey:    apiKey,
	}
}

// getPanel1Client lazily initializes and returns the 1Panel client.
func (p *PanelProvider) getPanel1Client() (*Panel1Client, error) {
	p.panel1Once.Do(func() {
		if p.baseURL == "" || p.apiKey == "" {
			p.panel1Err = fmt.Errorf("1Panel base URL and API key are required")
			return
		}
		p.panel1 = NewPanel1Client(p.baseURL, p.apiKey)
	})
	return p.panel1, p.panel1Err
}

// getBTPanelClient lazily initializes and returns the BT-Panel client.
func (p *PanelProvider) getBTPanelClient() (*BTPanelClient, error) {
	p.btPanelOnce.Do(func() {
		if p.baseURL == "" || p.apiKey == "" {
			p.btPanelErr = fmt.Errorf("BT-Panel base URL and API key are required")
			return
		}
		// For BT Panel, apiKey is actually the password
		// We assume username is "admin" by default
		p.btPanel = NewBTPanelClient(p.baseURL, "admin", p.apiKey)
	})
	return p.btPanel, p.btPanelErr
}

// DetectPanel attempts to detect which panel is installed.
func DetectPanel(ctx context.Context, executor CommandExecutor) PanelType {
	// Try 1Panel
	output, err := executor.RunCommand(ctx, "systemctl is-active 1panel 2>/dev/null || echo 'inactive'")
	if err == nil && len(output) > 0 && output != "inactive" {
		slog.Info("detected 1Panel", "status", output)
		return Panel1Panel
	}

	// Try BT-Panel
	output, err = executor.RunCommand(ctx, "systemctl is-active bt 2>/dev/null || echo 'inactive'")
	if err == nil && len(output) > 0 && output != "inactive" {
		slog.Info("detected BT-Panel", "status", output)
		return PanelBTPanel
	}

	// Fallback: check processes
	output, err = executor.RunCommand(ctx, "ps aux | grep -E '1panel|BT-Panel|bt' | grep -v grep || true")
	if err == nil && len(output) > 0 {
		if contains(output, "1panel") {
			return Panel1Panel
		}
		if contains(output, "BT-Panel") || contains(output, "bt") {
			return PanelBTPanel
		}
	}

	return PanelNone
}

// GetPanelInfo returns information about the detected panel.
func (p *PanelProvider) GetPanelInfo(_ context.Context) (map[string]interface{}, error) {
	if p.panelType == PanelNone {
		return nil, fmt.Errorf("no panel detected")
	}

	info := map[string]interface{}{
		"type":     string(p.panelType),
		"base_url": p.baseURL,
	}

	switch p.panelType {
	case Panel1Panel:
		info["name"] = "1Panel"
		info["features"] = []string{"container_management", "website_management", "firewall", "database"}
	case PanelBTPanel:
		info["name"] = "BT-Panel (宝塔)"
		info["features"] = []string{"website_management", "database", "ftp", "ssl", "cron"}
	}

	return info, nil
}

// OpenFirewall opens a port on the panel's firewall.
// It uses the panel-specific API when available, or falls back to SSH.
func (p *PanelProvider) OpenFirewall(ctx context.Context, port int, protocol string) error {
	slog.Info("opening firewall port", "panel", p.panelType, "port", port, "protocol", protocol)

	switch p.panelType {
	case Panel1Panel:
		client, err := p.getPanel1Client()
		if err != nil {
			return fmt.Errorf("failed to initialize 1Panel client: %w", err)
		}
		return client.OpenFirewall(ctx, port, protocol)

	case PanelBTPanel:
		client, err := p.getBTPanelClient()
		if err != nil {
			return fmt.Errorf("failed to initialize BT-Panel client: %w", err)
		}
		return client.OpenFirewall(ctx, port, protocol)

	default:
		slog.Warn("no panel detected, skipping firewall operation (use SSH fallback)", "port", port, "protocol", protocol)
		return nil
	}
}

// CloseFirewall closes a port on the panel's firewall.
// It uses the panel-specific API when available, or falls back to SSH.
func (p *PanelProvider) CloseFirewall(ctx context.Context, port int, protocol string) error {
	slog.Info("closing firewall port", "panel", p.panelType, "port", port, "protocol", protocol)

	switch p.panelType {
	case Panel1Panel:
		client, err := p.getPanel1Client()
		if err != nil {
			return fmt.Errorf("failed to initialize 1Panel client: %w", err)
		}
		return client.CloseFirewall(ctx, port, protocol)

	case PanelBTPanel:
		client, err := p.getBTPanelClient()
		if err != nil {
			return fmt.Errorf("failed to initialize BT-Panel client: %w", err)
		}
		return client.CloseFirewall(ctx, port, protocol)

	default:
		slog.Warn("no panel detected, skipping firewall operation (use SSH fallback)", "port", port, "protocol", protocol)
		return nil
	}
}

// CreateReverseProxy creates a reverse proxy via the panel's API.
// It uses the panel-specific API when available, or falls back to SSH.
func (p *PanelProvider) CreateReverseProxy(ctx context.Context, domain, targetURL string, port int) error {
	slog.Info("creating reverse proxy", "panel", p.panelType, "domain", domain, "targetURL", targetURL, "port", port)

	switch p.panelType {
	case Panel1Panel:
		client, err := p.getPanel1Client()
		if err != nil {
			return fmt.Errorf("failed to initialize 1Panel client: %w", err)
		}
		return client.CreateReverseProxy(ctx, domain, targetURL, port)

	case PanelBTPanel:
		client, err := p.getBTPanelClient()
		if err != nil {
			return fmt.Errorf("failed to initialize BT-Panel client: %w", err)
		}
		return client.CreateReverseProxy(ctx, domain, targetURL, port)

	default:
		slog.Warn("no panel detected, skipping reverse proxy creation (use SSH fallback)", "domain", domain)
		return nil
	}
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

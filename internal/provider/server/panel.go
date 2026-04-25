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
}

// NewPanelProvider creates a panel provider based on the detected panel type.
func NewPanelProvider(panelType PanelType, baseURL, apiKey string) *PanelProvider {
	return &PanelProvider{
		panelType: panelType,
		baseURL:   baseURL,
		apiKey:    apiKey,
	}
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

	if p.panelType == Panel1Panel {
		info["name"] = "1Panel"
		info["features"] = []string{"container_management", "website_management", "firewall", "database"}
	} else if p.panelType == PanelBTPanel {
		info["name"] = "BT-Panel (宝塔)"
		info["features"] = []string{"website_management", "database", "ftp", "ssl", "cron"}
	}

	return info, nil
}

// OpenFirewall opens a port on the panel's firewall.
func (p *PanelProvider) OpenFirewall(_ context.Context, port int, protocol string) error {
	slog.Info("opening firewall port", "panel", p.panelType, "port", port, "protocol", protocol)
	// Placeholder -- in production, call panel API
	return nil
}

// CloseFirewall closes a port on the panel's firewall.
func (p *PanelProvider) CloseFirewall(_ context.Context, port int, protocol string) error {
	slog.Info("closing firewall port", "panel", p.panelType, "port", port, "protocol", protocol)
	// Placeholder -- in production, call panel API
	return nil
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

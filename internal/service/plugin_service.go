package service

import (
	"context"
	"fmt"

	"github.com/Yogdunana/deploypilot/internal/model"
)

// ---------- PluginOps ----------

// PluginOps handles plugin lifecycle operations (enable, disable, reload).
func (b *Bridge) PluginOps(pluginID string, action string) (interface{}, error) {
	if b.PluginMgr == nil {
		return nil, fmt.Errorf("plugin manager not available")
	}

	ctx := context.Background()

	switch action {
	case "enable":
		if err := b.PluginMgr.EnablePlugin(ctx, pluginID); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":    "success",
			"plugin_id": pluginID,
			"action":    "enabled",
		}, nil

	case "disable":
		if err := b.PluginMgr.DisablePlugin(ctx, pluginID); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":    "success",
			"plugin_id": pluginID,
			"action":    "disabled",
		}, nil

	case "reload":
		if err := b.PluginMgr.ReloadPlugin(ctx, pluginID); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":    "success",
			"plugin_id": pluginID,
			"action":    "reloaded",
		}, nil

	default:
		return nil, fmt.Errorf("unknown plugin action: %s (valid: enable, disable, reload)", action)
	}
}
// ---------- ListPlugins ----------

// ListPlugins returns all plugins from DB, optionally filtered by provider.
func (b *Bridge) ListPlugins(provider string) (interface{}, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}

	plugins, err := model.ListPlugins(model.DefaultTenantID, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}

	// Enrich with registry descriptor info
	result := make([]map[string]interface{}, 0, len(plugins))
	for _, p := range plugins {
		entry := map[string]interface{}{
			"id":           p.ID,
			"name":         p.Name,
			"display_name": p.DisplayName,
			"version":      p.Version,
			"description":  p.Description,
			"author":       p.Author,
			"provider":     p.Provider,
			"type":         p.Type,
			"enabled":      p.Enabled,
			"priority":     p.Priority,
			"status":       p.Status,
			"created_at":   p.CreatedAt,
			"updated_at":   p.UpdatedAt,
		}
		if p.ErrorMsg != "" {
			entry["error_msg"] = p.ErrorMsg
		}
		result = append(result, entry)
	}

	return map[string]interface{}{
		"status": "success",
		"total":  len(result),
		"plugins": result,
	}, nil
}
// ---------- GetPluginInfo ----------

// GetPluginInfo returns detailed information about a specific plugin.
func (b *Bridge) GetPluginInfo(pluginID string) (interface{}, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}

	p, err := model.GetPlugin(pluginID)
	if err != nil {
		return nil, fmt.Errorf("plugin not found: %w", err)
	}

	result := map[string]interface{}{
		"id":           p.ID,
		"name":         p.Name,
		"display_name": p.DisplayName,
		"version":      p.Version,
		"description":  p.Description,
		"author":       p.Author,
		"provider":     p.Provider,
		"type":         p.Type,
		"enabled":      p.Enabled,
		"priority":     p.Priority,
		"status":       p.Status,
		"created_at":   p.CreatedAt,
		"updated_at":   p.UpdatedAt,
	}

	if p.ErrorMsg != "" {
		result["error_msg"] = p.ErrorMsg
	}

	// Check if instance is loaded
	if b.PluginMgr != nil {
		instance, err := b.PluginMgr.GetPluginInstance(pluginID)
		if err == nil {
			result["instance_loaded"] = true
			result["instance_type"] = fmt.Sprintf("%T", instance)
		} else {
			result["instance_loaded"] = false
		}
	}

	return map[string]interface{}{
		"status": "success",
		"plugin": result,
	}, nil
}

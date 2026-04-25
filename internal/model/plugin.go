package model

import (
	"fmt"
	"time"
)

// Plugin represents a plugin configuration stored in the database.
type Plugin struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	TenantID    string    `gorm:"index" json:"tenant_id"`
	Name        string    `gorm:"not null;uniqueIndex:idx_tenant_plugin_name" json:"name"`
	DisplayName string    `json:"display_name"`
	Version     string    `gorm:"default:1.0.0" json:"version"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	Provider    string    `gorm:"not null" json:"provider"` // dns, notify, registry, cicd, server, ssl
	Type        string    `gorm:"not null" json:"type"`     // e.g., cloudflare, webhook, docker_hub
	Config      string    `gorm:"type:text" json:"config"`  // JSON config for the plugin
	Enabled     bool      `gorm:"default:true" json:"enabled"`
	Priority    int       `gorm:"default:0" json:"priority"` // higher = loaded first
	Status      string    `gorm:"default:active" json:"status"` // active, error, disabled
	ErrorMsg    string    `json:"error_msg,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (Plugin) TableName() string { return "plugins" }

// CreatePlugin creates a new plugin record.
func CreatePlugin(tenantID, name, displayName, version, description, author, provider, pluginType, config string, enabled bool, priority int) (*Plugin, error) {
	plugin := &Plugin{
		ID:          "plg-" + generateUUID(),
		TenantID:    tenantID,
		Name:        name,
		DisplayName: displayName,
		Version:     version,
		Description: description,
		Author:      author,
		Provider:    provider,
		Type:        pluginType,
		Config:      config,
		Enabled:     enabled,
		Priority:    priority,
		Status:      "active",
	}

	result := getDB().Create(plugin)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create plugin: %w", result.Error)
	}

	return plugin, nil
}

// GetPlugin retrieves a plugin by ID.
func GetPlugin(id string) (*Plugin, error) {
	var plugin Plugin
	result := getDB().First(&plugin, "id = ?", id)
	if result.Error != nil {
		return nil, fmt.Errorf("plugin not found: %w", result.Error)
	}
	return &plugin, nil
}

// ListPlugins returns all plugins for a tenant, optionally filtered by provider.
func ListPlugins(tenantID string, provider string) ([]Plugin, error) {
	var plugins []Plugin
	query := getDB().Where("tenant_id = ?", tenantID)
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}
	result := query.Order("priority DESC, created_at ASC").Find(&plugins)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", result.Error)
	}
	return plugins, nil
}

// UpdatePlugin updates a plugin's fields. Only non-empty values are updated.
func UpdatePlugin(id string, updates map[string]interface{}) (*Plugin, error) {
	if len(updates) == 0 {
		return GetPlugin(id)
	}

	result := getDB().Model(&Plugin{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update plugin: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("plugin not found")
	}

	return GetPlugin(id)
}

// DeletePlugin removes a plugin by ID.
func DeletePlugin(id string) error {
	result := getDB().Delete(&Plugin{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete plugin: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("plugin not found")
	}
	return nil
}

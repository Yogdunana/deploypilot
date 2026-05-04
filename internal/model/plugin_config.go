package model

import (
	"fmt"
	"time"
)

// PluginConfig stores event-driven plugin configuration in the database.
// This is separate from the provider-based Plugin model.
type PluginConfig struct {
	ID        string    `json:"id" gorm:"primaryKey;size:36"`
	TenantID  string    `json:"tenant_id" gorm:"index;size:36"`
	Name      string    `json:"name" gorm:"uniqueIndex;size:100"`
	Enabled   bool      `json:"enabled"`
	Config    string    `json:"config" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (PluginConfig) TableName() string { return "plugin_configs" }

// CreatePluginConfig creates a new plugin config record.
func CreatePluginConfig(tenantID, name string, enabled bool, config string) (*PluginConfig, error) {
	pc := &PluginConfig{
		ID:        "pcfg-" + generateUUID(),
		TenantID:  tenantID,
		Name:      name,
		Enabled:   enabled,
		Config:    config,
	}
	result := getDB().Create(pc)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create plugin config: %w", result.Error)
	}
	return pc, nil
}

// GetPluginConfig retrieves a plugin config by ID.
func GetPluginConfig(id string) (*PluginConfig, error) {
	var pc PluginConfig
	result := getDB().First(&pc, "id = ?", id)
	if result.Error != nil {
		return nil, fmt.Errorf("plugin config not found: %w", result.Error)
	}
	return &pc, nil
}

// ListPluginConfigs returns all plugin configs for a tenant.
func ListPluginConfigs(tenantID string) ([]PluginConfig, error) {
	var configs []PluginConfig
	result := getDB().Where("tenant_id = ?", tenantID).Find(&configs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list plugin configs: %w", result.Error)
	}
	return configs, nil
}

// UpdatePluginConfig updates a plugin config's fields.
func UpdatePluginConfig(id string, updates map[string]interface{}) (*PluginConfig, error) {
	if len(updates) == 0 {
		return GetPluginConfig(id)
	}
	result := getDB().Model(&PluginConfig{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update plugin config: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("plugin config not found")
	}
	return GetPluginConfig(id)
}

// UpdatePluginConfigByTenant updates a plugin config by tenant ID and plugin name.
// It creates the record if it does not exist (upsert).
func UpdatePluginConfigByTenant(tenantID, name, config string, enabled *bool) error {
	db := getDB()
	var existing PluginConfig
	result := db.Where("tenant_id = ? AND name = ?", tenantID, name).First(&existing)
	if result.Error != nil {
		// Record does not exist, create it
		isEnabled := true
		if enabled != nil {
			isEnabled = *enabled
		}
		_, err := CreatePluginConfig(tenantID, name, isEnabled, config)
		if err != nil {
			return fmt.Errorf("failed to create plugin config: %w", err)
		}
		return nil
	}
	// Record exists, update it
	updates := map[string]interface{}{"config": config}
	if enabled != nil {
		updates["enabled"] = *enabled
	}
	if err := db.Model(&PluginConfig{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update plugin config: %w", err)
	}
	return nil
}

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/Yogdunana/deploypilot/internal/crypto"
	"gorm.io/gorm"
)

// Manager handles plugin lifecycle: load, enable, disable, reload.
type Manager struct {
	registry *Registry
	db       *gorm.DB
	encKey   string
	mu       sync.RWMutex
}

// NewManager creates a new plugin lifecycle manager.
func NewManager(registry *Registry, db *gorm.DB, encKey string) *Manager {
	return &Manager{
		registry: registry,
		db:       db,
		encKey:   encKey,
	}
}

// LoadAll loads all enabled plugins from DB and creates instances.
func (m *Manager) LoadAll(ctx context.Context, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var plugins []struct {
		ID       string `gorm:"column:id"`
		Provider string `gorm:"column:provider"`
		Type     string `gorm:"column:type"`
		Config   string `gorm:"column:config"`
		Enabled  bool   `gorm:"column:enabled"`
	}
	if err := m.db.Table("plugins").
		Where("tenant_id = ? AND enabled = ?", tenantID, true).
		Order("priority DESC").
		Find(&plugins).Error; err != nil {
		return fmt.Errorf("failed to query enabled plugins: %w", err)
	}

	for _, p := range plugins {
		if err := m.loadPlugin(ctx, p.ID, p.Provider, p.Type, p.Config); err != nil {
			slog.Error("failed to load plugin", "id", p.ID, "error", err)
			// Update status to error
			m.db.Table("plugins").Where("id = ?", p.ID).Updates(map[string]interface{}{
				"status":    "error",
				"error_msg": err.Error(),
			})
		}
	}

	slog.Info("plugins loaded", "count", len(plugins), "tenant", tenantID)
	return nil
}

// LoadPlugin loads a single plugin by ID, creates its instance.
func (m *Manager) LoadPlugin(ctx context.Context, pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var p struct {
		ID       string `gorm:"column:id"`
		Provider string `gorm:"column:provider"`
		Type     string `gorm:"column:type"`
		Config   string `gorm:"column:config"`
		Enabled  bool   `gorm:"column:enabled"`
	}
	if err := m.db.Table("plugins").Where("id = ?", pluginID).Take(&p).Error; err != nil {
		return fmt.Errorf("plugin not found: %w", err)
	}

	if !p.Enabled {
		return fmt.Errorf("plugin %s is disabled", pluginID)
	}

	if err := m.loadPlugin(ctx, p.ID, p.Provider, p.Type, p.Config); err != nil {
		return err
	}

	return nil
}

// loadPlugin is the internal implementation that creates a plugin instance.
// Caller must hold m.mu.
func (m *Manager) loadPlugin(ctx context.Context, pluginID, provider, pluginType, configJSON string) error {
	desc, ok := m.registry.GetDescriptor(provider, pluginType)
	if !ok {
		return fmt.Errorf("no registered descriptor for %s:%s", provider, pluginType)
	}

	var config map[string]interface{}
	if configJSON != "" {
		if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
			return fmt.Errorf("failed to parse plugin config: %w", err)
		}
	}

	// Decrypt config values if encryption key is available
	if m.encKey != "" {
		var decErr error
		config, decErr = decryptPluginConfig(m.encKey, config)
		if decErr != nil {
			slog.Warn("failed to decrypt plugin config", "error", decErr)
		}
	}

	instance, err := m.registry.CreateInstance(pluginID, desc, config)
	if err != nil {
		return fmt.Errorf("failed to create instance for plugin %s: %w", pluginID, err)
	}

	// Update status to active
	m.db.Table("plugins").Where("id = ?", pluginID).Updates(map[string]interface{}{
		"status":    "active",
		"error_msg": "",
	})

	slog.Info("plugin loaded", "id", pluginID, "provider", provider, "type", pluginType, "instance_type", fmt.Sprintf("%T", instance))
	return nil
}

// UnloadPlugin removes a plugin instance.
func (m *Manager) UnloadPlugin(pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.registry.RemoveInstance(pluginID)

	// Update status to disabled
	m.db.Table("plugins").Where("id = ?", pluginID).Updates(map[string]interface{}{
		"status":    "disabled",
		"error_msg": "",
	})

	slog.Info("plugin unloaded", "id", pluginID)
	return nil
}

// ReloadPlugin reloads a plugin (unload + load).
func (m *Manager) ReloadPlugin(ctx context.Context, pluginID string) error {
	if err := m.UnloadPlugin(pluginID); err != nil {
		return err
	}
	return m.LoadPlugin(ctx, pluginID)
}

// EnablePlugin enables a plugin and loads it.
func (m *Manager) EnablePlugin(ctx context.Context, pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update enabled flag in DB
	result := m.db.Table("plugins").Where("id = ?", pluginID).Update("enabled", true)
	if result.Error != nil {
		return fmt.Errorf("failed to enable plugin: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("plugin not found: %s", pluginID)
	}

	// Load the plugin
	var p struct {
		ID       string `gorm:"column:id"`
		Provider string `gorm:"column:provider"`
		Type     string `gorm:"column:type"`
		Config   string `gorm:"column:config"`
	}
	if err := m.db.Table("plugins").Where("id = ?", pluginID).Take(&p).Error; err != nil {
		return fmt.Errorf("plugin not found after enable: %w", err)
	}

	if err := m.loadPlugin(ctx, p.ID, p.Provider, p.Type, p.Config); err != nil {
		return err
	}

	slog.Info("plugin enabled and loaded", "id", pluginID)
	return nil
}

// DisablePlugin disables a plugin and unloads it.
func (m *Manager) DisablePlugin(ctx context.Context, pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update enabled flag in DB
	result := m.db.Table("plugins").Where("id = ?", pluginID).Update("enabled", false)
	if result.Error != nil {
		return fmt.Errorf("failed to disable plugin: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("plugin not found: %s", pluginID)
	}

	// Unload the instance
	m.registry.RemoveInstance(pluginID)

	// Update status
	m.db.Table("plugins").Where("id = ?", pluginID).Updates(map[string]interface{}{
		"status":    "disabled",
		"error_msg": "",
	})

	slog.Info("plugin disabled and unloaded", "id", pluginID)
	return nil
}

// GetPluginInstance returns the provider instance for a plugin.
func (m *Manager) GetPluginInstance(pluginID string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance, ok := m.registry.GetInstance(pluginID)
	if !ok {
		return nil, fmt.Errorf("no instance found for plugin %s", pluginID)
	}
	return instance, nil
}

// GetPluginStatus returns the status of all plugins.
func (m *Manager) GetPluginStatus(tenantID string) ([]map[string]interface{}, error) {
	var plugins []struct {
		ID          string `gorm:"column:id"`
		Name        string `gorm:"column:name"`
		DisplayName string `gorm:"column:display_name"`
		Provider    string `gorm:"column:provider"`
		Type        string `gorm:"column:type"`
		Enabled     bool   `gorm:"column:enabled"`
		Status      string `gorm:"column:status"`
		ErrorMsg    string `gorm:"column:error_msg"`
		Priority    int    `gorm:"column:priority"`
		Version     string `gorm:"column:version"`
		CreatedAt   string `gorm:"column:created_at"`
		UpdatedAt   string `gorm:"column:updated_at"`
	}
	if err := m.db.Table("plugins").
		Where("tenant_id = ?", tenantID).
		Order("priority DESC, created_at ASC").
		Find(&plugins).Error; err != nil {
		return nil, fmt.Errorf("failed to query plugins: %w", err)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]map[string]interface{}, 0, len(plugins))
	for _, p := range plugins {
		_, instanceLoaded := m.registry.GetInstance(p.ID)

		entry := map[string]interface{}{
			"id":           p.ID,
			"name":         p.Name,
			"display_name": p.DisplayName,
			"provider":     p.Provider,
			"type":         p.Type,
			"enabled":      p.Enabled,
			"status":       p.Status,
			"priority":     p.Priority,
			"version":      p.Version,
			"instance_loaded": instanceLoaded,
			"created_at":   p.CreatedAt,
			"updated_at":   p.UpdatedAt,
		}
		if p.ErrorMsg != "" {
			entry["error_msg"] = p.ErrorMsg
		}
		result = append(result, entry)
	}

	return result, nil
}

// decryptPluginConfig decrypts sensitive values in plugin config.
// Values prefixed with "enc:" are treated as encrypted and decrypted using AES-256-GCM.
func decryptPluginConfig(encKey string, config map[string]interface{}) (map[string]interface{}, error) {
	if encKey == "" || config == nil {
		return config, nil
	}

	key := []byte(encKey)
	result := make(map[string]interface{}, len(config))

	for k, v := range config {
		strVal, ok := v.(string)
		if !ok {
			result[k] = v
			continue
		}
		if strings.HasPrefix(strVal, "enc:") {
			decoded, err := crypto.Decrypt(key, strings.TrimPrefix(strVal, "enc:"))
			if err != nil {
				// Log but don't fail — return original value
				slog.Warn("failed to decrypt plugin config value", "key", k, "error", err)
				result[k] = v
				continue
			}
			result[k] = decoded
		} else {
			result[k] = v
		}
	}

	return result, nil
}

package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// EventPluginManager manages event-driven plugins (separate from the provider-based Manager).
// It handles registration, lifecycle (init/start/stop), and event dispatching.
type EventPluginManager struct {
	plugins  map[string]EventPlugin
	configs  map[string]map[string]interface{}
	statuses map[string]PluginStatus
	errors   map[string]string
	enabled  map[string]bool
	mu       sync.RWMutex
}

// NewEventPluginManager creates a new EventPluginManager.
func NewEventPluginManager() *EventPluginManager {
	return &EventPluginManager{
		plugins:  make(map[string]EventPlugin),
		configs:  make(map[string]map[string]interface{}),
		statuses: make(map[string]PluginStatus),
		errors:   make(map[string]string),
		enabled:  make(map[string]bool),
	}
}

// Register registers an event plugin. The plugin starts in "registered" state and is enabled by default.
func (m *EventPluginManager) Register(p EventPlugin) {
	name := p.Name()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins[name] = p
	m.statuses[name] = PluginStatusRegistered
	m.enabled[name] = true
	slog.Info("event plugin registered", "name", name, "version", p.Version())
}

// InitAll initializes all enabled plugins that are in "registered" state.
func (m *EventPluginManager) InitAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, p := range m.plugins {
		if !m.enabled[name] {
			continue
		}
		config := m.configs[name]
		if err := p.Init(ctx, config); err != nil {
			m.statuses[name] = PluginStatusError
			m.errors[name] = err.Error()
			slog.Error("event plugin init failed", "name", name, "error", err)
			continue
		}
		m.statuses[name] = PluginStatusInitialized
	}
	return nil
}

// StartAll starts all initialized plugins.
func (m *EventPluginManager) StartAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, p := range m.plugins {
		if !m.enabled[name] || m.statuses[name] != PluginStatusInitialized {
			continue
		}
		if err := p.Start(); err != nil {
			m.statuses[name] = PluginStatusError
			m.errors[name] = err.Error()
			slog.Error("event plugin start failed", "name", name, "error", err)
			continue
		}
		m.statuses[name] = PluginStatusRunning
		slog.Info("event plugin started", "name", name)
	}
	return nil
}

// StopAll stops all running plugins.
func (m *EventPluginManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, p := range m.plugins {
		if m.statuses[name] != PluginStatusRunning {
			continue
		}
		if err := p.Stop(); err != nil {
			slog.Error("event plugin stop failed", "name", name, "error", err)
		}
		m.statuses[name] = PluginStatusStopped
	}
}

// ListPlugins returns info about all registered plugins.
func (m *EventPluginManager) ListPlugins() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]PluginInfo, 0, len(m.plugins))
	for name, p := range m.plugins {
		info := PluginInfo{
			Name:        p.Name(),
			Version:     p.Version(),
			Description: p.Description(),
			Status:      m.statuses[name],
			Enabled:     m.enabled[name],
			Error:       m.errors[name],
			Config:      m.configs[name],
		}
		result = append(result, info)
	}
	return result
}

// GetPlugin returns a plugin by name.
func (m *EventPluginManager) GetPlugin(name string) (EventPlugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[name]
	return p, ok
}

// EnablePlugin enables a previously disabled plugin.
func (m *EventPluginManager) EnablePlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.plugins[name]; !ok {
		return fmt.Errorf("plugin %s not found", name)
	}
	m.enabled[name] = true
	return nil
}

// DisablePlugin disables a plugin and stops it if running.
func (m *EventPluginManager) DisablePlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.plugins[name]; !ok {
		return fmt.Errorf("plugin %s not found", name)
	}
	if m.statuses[name] == PluginStatusRunning {
		p := m.plugins[name]
		_ = p.Stop()
		m.statuses[name] = PluginStatusStopped
	}
	m.enabled[name] = false
	return nil
}

// SetPluginConfig sets the configuration for a plugin.
func (m *EventPluginManager) SetPluginConfig(name string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.plugins[name]; !ok {
		return fmt.Errorf("plugin %s not found", name)
	}
	m.configs[name] = config
	return nil
}

// DispatchEvent sends an event to all running plugins.
func (m *EventPluginManager) DispatchEvent(event BusEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for name, p := range m.plugins {
		if m.statuses[name] != PluginStatusRunning {
			continue
		}
		go func(n string, pl EventPlugin) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("event plugin panic in OnEvent", "plugin", n, "panic", r)
				}
			}()
			pl.OnEvent(event)
		}(name, p)
	}
}

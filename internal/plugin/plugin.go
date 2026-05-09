package plugin

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// BusEvent is a local copy to avoid importing service package (prevent import cycle).
type BusEvent struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Topic     string      `json:"topic"`
	Source    string      `json:"source,omitempty"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

// EventPlugin is the interface that all event-driven plugins must implement.
// This is separate from the existing provider-based PluginDescriptor system.
type EventPlugin interface {
	// Name returns the unique identifier for this plugin.
	Name() string
	// Version returns the plugin version string.
	Version() string
	// Description returns a human-readable description.
	Description() string
	// Init initializes the plugin with the given configuration.
	Init(ctx context.Context, config map[string]interface{}) error
	// Start starts the plugin's background operations.
	Start() error
	// Stop gracefully stops the plugin.
	Stop() error
	// OnEvent handles an event from the typed event bus.
	OnEvent(event BusEvent)
	// RegisterAPIRoutes registers plugin-specific API routes under the given group.
	RegisterAPIRoutes(rg *gin.RouterGroup)
}

// PluginStatus represents the current status of a plugin.
type PluginStatus string

const (
	PluginStatusRegistered  PluginStatus = "registered"
	PluginStatusInitialized PluginStatus = "initialized"
	PluginStatusRunning     PluginStatus = "running"
	PluginStatusStopped     PluginStatus = "stopped"
	PluginStatusError       PluginStatus = "error"
)

// PluginInfo contains metadata about a plugin.
type PluginInfo struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Status      PluginStatus           `json:"status"`
	Enabled     bool                   `json:"enabled"`
	Error       string                 `json:"error,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

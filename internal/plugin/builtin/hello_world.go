package builtin

import (
	"context"
	"log/slog"

	"github.com/Yogdunana/deploypilot/internal/plugin"
	"github.com/gin-gonic/gin"
)

// HelloWorldPlugin is a simple plugin that logs all received events.
// It is useful for debugging and as a reference implementation.
type HelloWorldPlugin struct {
	config map[string]interface{}
}

// NewHelloWorldPlugin creates a new HelloWorldPlugin.
func NewHelloWorldPlugin() plugin.EventPlugin {
	return &HelloWorldPlugin{}
}

func (p *HelloWorldPlugin) Name() string {
	return "hello-world"
}

func (p *HelloWorldPlugin) Version() string {
	return "1.0.0"
}

func (p *HelloWorldPlugin) Description() string {
	return "A simple plugin that logs all received events for debugging"
}

func (p *HelloWorldPlugin) Init(_ context.Context, config map[string]interface{}) error {
	p.config = config
	slog.Info("hello-world plugin initialized")
	return nil
}

func (p *HelloWorldPlugin) Start() error {
	slog.Info("hello-world plugin started")
	return nil
}

func (p *HelloWorldPlugin) Stop() error {
	slog.Info("hello-world plugin stopped")
	return nil
}

func (p *HelloWorldPlugin) OnEvent(event plugin.BusEvent) {
	slog.Info("hello-world plugin received event",
		"type", event.Type,
		"topic", event.Topic,
		"source", event.Source,
	)
}

func (p *HelloWorldPlugin) RegisterAPIRoutes(_ *gin.RouterGroup) {
	// No custom API routes for this plugin
}

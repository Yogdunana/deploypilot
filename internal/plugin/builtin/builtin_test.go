package builtin

import (
	"context"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/plugin"
)

func TestHelloWorldPlugin(t *testing.T) {
	p := NewHelloWorldPlugin()

	if p.Name() != "hello-world" {
		t.Errorf("Name() = %q, want %q", p.Name(), "hello-world")
	}
	if p.Version() != "1.0.0" {
		t.Errorf("Version() = %q, want %q", p.Version(), "1.0.0")
	}
	if p.Description() == "" {
		t.Error("Description() should not be empty")
	}

	err := p.Init(context.Background(), map[string]interface{}{"key": "value"})
	if err != nil {
		t.Errorf("Init() error = %v", err)
	}

	err = p.Start()
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	p.OnEvent(plugin.BusEvent{Type: "deploy", Topic: "test-topic", Source: "test-source"})

	p.RegisterAPIRoutes(nil)

	err = p.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestHelloWorldPlugin_EmptyConfig(t *testing.T) {
	p := NewHelloWorldPlugin()

	err := p.Init(context.Background(), nil)
	if err != nil {
		t.Errorf("Init() with nil config error = %v", err)
	}

	err = p.Start()
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	err = p.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestDeployGatePlugin(t *testing.T) {
	p := NewDeployGatePlugin().(*DeployGatePlugin)

	if p.Name() != "deploy-gate" {
		t.Errorf("Name() = %q, want %q", p.Name(), "deploy-gate")
	}
	if p.Version() != "1.0.0" {
		t.Errorf("Version() = %q, want %q", p.Version(), "1.0.0")
	}

	err := p.Init(context.Background(), map[string]interface{}{"require_approval": true})
	if err != nil {
		t.Errorf("Init() error = %v", err)
	}

	err = p.Start()
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	event := plugin.BusEvent{ID: "evt-1", Type: "deploy", Topic: "test-deploy"}
	p.OnEvent(event)

	pending := p.PendingDeployments()
	if len(pending) != 1 {
		t.Errorf("PendingDeployments() = %d, want 1", len(pending))
	}
	if pending[0].ID != "evt-1" {
		t.Errorf("Pending event ID = %q, want %q", pending[0].ID, "evt-1")
	}

	err = p.ApproveDeployment("evt-1")
	if err != nil {
		t.Errorf("ApproveDeployment() error = %v", err)
	}

	pending = p.PendingDeployments()
	if len(pending) != 0 {
		t.Errorf("PendingDeployments() after approve = %d, want 0", len(pending))
	}

	err = p.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestDeployGatePlugin_Disabled(t *testing.T) {
	p := NewDeployGatePlugin().(*DeployGatePlugin)

	err := p.Init(context.Background(), map[string]interface{}{"require_approval": false})
	if err != nil {
		t.Errorf("Init() error = %v", err)
	}

	p.OnEvent(plugin.BusEvent{ID: "evt-2", Type: "deploy", Topic: "test-deploy"})

	pending := p.PendingDeployments()
	if len(pending) != 0 {
		t.Errorf("PendingDeployments() should be empty when approval disabled, got %d", len(pending))
	}
}

func TestDeployGatePlugin_EmptyConfig(t *testing.T) {
	p := NewDeployGatePlugin().(*DeployGatePlugin)

	err := p.Init(context.Background(), nil)
	if err != nil {
		t.Errorf("Init() with nil config error = %v", err)
	}

	p.OnEvent(plugin.BusEvent{ID: "evt-3", Type: "deploy", Topic: "test-deploy"})

	pending := p.PendingDeployments()
	if len(pending) != 0 {
		t.Errorf("PendingDeployments() should be empty when config is nil, got %d", len(pending))
	}
}

func TestDeployGatePlugin_NonDeployEvent(t *testing.T) {
	p := NewDeployGatePlugin().(*DeployGatePlugin)

	err := p.Init(context.Background(), map[string]interface{}{"require_approval": true})
	if err != nil {
		t.Errorf("Init() error = %v", err)
	}

	p.OnEvent(plugin.BusEvent{ID: "evt-4", Type: "other", Topic: "test-topic"})

	pending := p.PendingDeployments()
	if len(pending) != 0 {
		t.Errorf("PendingDeployments() should be empty for non-deploy event, got %d", len(pending))
	}
}

func TestDeployGatePlugin_ApproveNotFound(t *testing.T) {
	p := NewDeployGatePlugin().(*DeployGatePlugin)

	err := p.ApproveDeployment("nonexistent")
	if err == nil {
		t.Error("ApproveDeployment() should fail for nonexistent deployment")
	}
}

func TestDeployGatePlugin_RejectNotFound(t *testing.T) {
	p := NewDeployGatePlugin().(*DeployGatePlugin)

	err := p.RejectDeployment("nonexistent")
	if err == nil {
		t.Error("RejectDeployment() should fail for nonexistent deployment")
	}
}

func TestDeployGatePlugin_RejectDeployment(t *testing.T) {
	p := NewDeployGatePlugin().(*DeployGatePlugin)

	err := p.Init(context.Background(), map[string]interface{}{"require_approval": true})
	if err != nil {
		t.Errorf("Init() error = %v", err)
	}

	event := plugin.BusEvent{ID: "evt-5", Type: "deploy", Topic: "test-deploy"}
	p.OnEvent(event)

	err = p.RejectDeployment("evt-5")
	if err != nil {
		t.Errorf("RejectDeployment() error = %v", err)
	}

	pending := p.PendingDeployments()
	if len(pending) != 0 {
		t.Errorf("PendingDeployments() after reject = %d, want 0", len(pending))
	}
}

func TestSlackNotifyPlugin(t *testing.T) {
	p := NewSlackNotifyPlugin()

	if p.Name() != "slack-notify" {
		t.Errorf("Name() = %q, want %q", p.Name(), "slack-notify")
	}
	if p.Version() != "1.0.0" {
		t.Errorf("Version() = %q, want %q", p.Version(), "1.0.0")
	}

	err := p.Init(context.Background(), map[string]interface{}{"webhook_url": "https://hooks.slack.com/services/test"})
	if err != nil {
		t.Errorf("Init() error = %v", err)
	}

	err = p.Start()
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	p.OnEvent(plugin.BusEvent{Type: "deploy", Topic: "test-deploy", Payload: map[string]string{"app": "myapp"}})

	p.OnEvent(plugin.BusEvent{Type: "other", Topic: "test-topic"})

	p.RegisterAPIRoutes(nil)

	err = p.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestSlackNotifyPlugin_NilConfig(t *testing.T) {
	p := NewSlackNotifyPlugin()

	err := p.Init(context.Background(), nil)
	if err == nil {
		t.Error("Init() should fail with nil config")
	}
}

func TestSlackNotifyPlugin_EmptyWebhookURL(t *testing.T) {
	p := NewSlackNotifyPlugin()

	err := p.Init(context.Background(), map[string]interface{}{"webhook_url": ""})
	if err == nil {
		t.Error("Init() should fail with empty webhook_url")
	}
}

func TestSlackNotifyPlugin_MissingWebhookURL(t *testing.T) {
	p := NewSlackNotifyPlugin()

	err := p.Init(context.Background(), map[string]interface{}{"other_key": "value"})
	if err == nil {
		t.Error("Init() should fail with missing webhook_url")
	}
}
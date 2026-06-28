package builtin

import (
	"context"
	"sync"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/plugin"
)

func TestHelloWorldPlugin(t *testing.T) {
	p := &HelloWorldPlugin{}

	if p.Name() != "hello-world" {
		t.Errorf("expected name 'hello-world', got %q", p.Name())
	}
	if p.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", p.Version())
	}
	if p.Description() == "" {
		t.Error("expected non-empty description")
	}

	if err := p.Init(context.Background(), nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := p.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	p.OnEvent(plugin.BusEvent{Type: "test", Topic: "test-topic", Source: "test-source"})
}

func TestDeployGatePlugin_Init(t *testing.T) {
	p := newDeployGatePlugin()

	if err := p.Init(context.Background(), nil); err != nil {
		t.Fatalf("Init with nil config failed: %v", err)
	}

	config := map[string]interface{}{"require_approval": true}
	if err := p.Init(context.Background(), config); err != nil {
		t.Fatalf("Init with config failed: %v", err)
	}

	if err := p.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestDeployGatePlugin_Name(t *testing.T) {
	p := newDeployGatePlugin()
	if p.Name() != "deploy-gate" {
		t.Errorf("expected name 'deploy-gate', got %q", p.Name())
	}
	if p.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", p.Version())
	}
}

func TestDeployGatePlugin_OnEvent_NotDeploy(t *testing.T) {
	p := newDeployGatePlugin()
	if err := p.Init(context.Background(), map[string]interface{}{"require_approval": true}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	p.OnEvent(plugin.BusEvent{Type: "other", ID: "test-1"})

	pending := p.PendingDeployments()
	if len(pending) != 0 {
		t.Errorf("expected 0 pending deployments, got %d", len(pending))
	}
}

func TestDeployGatePlugin_OnEvent_ApprovalDisabled(t *testing.T) {
	p := newDeployGatePlugin()
	if err := p.Init(context.Background(), map[string]interface{}{"require_approval": false}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	p.OnEvent(plugin.BusEvent{Type: "deploy", ID: "test-1"})

	pending := p.PendingDeployments()
	if len(pending) != 0 {
		t.Errorf("expected 0 pending deployments when approval disabled, got %d", len(pending))
	}
}

func TestDeployGatePlugin_OnEvent_ApprovalEnabled(t *testing.T) {
	p := newDeployGatePlugin()
	if err := p.Init(context.Background(), map[string]interface{}{"require_approval": true}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	event := plugin.BusEvent{Type: "deploy", ID: "test-1", Topic: "app-1"}
	p.OnEvent(event)

	pending := p.PendingDeployments()
	if len(pending) != 1 {
		t.Errorf("expected 1 pending deployment, got %d", len(pending))
	}
	if pending[0].ID != "test-1" {
		t.Errorf("expected event ID 'test-1', got %q", pending[0].ID)
	}
}

func TestDeployGatePlugin_ApproveDeployment(t *testing.T) {
	p := newDeployGatePlugin()
	if err := p.Init(context.Background(), map[string]interface{}{"require_approval": true}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	p.OnEvent(plugin.BusEvent{Type: "deploy", ID: "test-1"})
	p.OnEvent(plugin.BusEvent{Type: "deploy", ID: "test-2"})

	if err := p.ApproveDeployment("test-1"); err != nil {
		t.Fatalf("ApproveDeployment failed: %v", err)
	}

	pending := p.PendingDeployments()
	if len(pending) != 1 {
		t.Errorf("expected 1 pending deployment after approval, got %d", len(pending))
	}
	if pending[0].ID != "test-2" {
		t.Errorf("expected remaining event ID 'test-2', got %q", pending[0].ID)
	}
}

func TestDeployGatePlugin_ApproveDeployment_NotFound(t *testing.T) {
	p := newDeployGatePlugin()
	if err := p.Init(context.Background(), map[string]interface{}{"require_approval": true}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	err := p.ApproveDeployment("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent deployment")
	}
}

func TestDeployGatePlugin_RejectDeployment(t *testing.T) {
	p := newDeployGatePlugin()
	if err := p.Init(context.Background(), map[string]interface{}{"require_approval": true}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	p.OnEvent(plugin.BusEvent{Type: "deploy", ID: "test-1"})

	if err := p.RejectDeployment("test-1"); err != nil {
		t.Fatalf("RejectDeployment failed: %v", err)
	}

	pending := p.PendingDeployments()
	if len(pending) != 0 {
		t.Errorf("expected 0 pending deployments after rejection, got %d", len(pending))
	}
}

func TestDeployGatePlugin_RejectDeployment_NotFound(t *testing.T) {
	p := newDeployGatePlugin()
	if err := p.Init(context.Background(), map[string]interface{}{"require_approval": true}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	err := p.RejectDeployment("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent deployment")
	}
}

func TestDeployGatePlugin_ConcurrentAccess(t *testing.T) {
	p := newDeployGatePlugin()
	if err := p.Init(context.Background(), map[string]interface{}{"require_approval": true}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			p.OnEvent(plugin.BusEvent{Type: "deploy", ID: "event-" + string(rune('a'+id))})
		}(i)
	}

	wg.Wait()

	pending := p.PendingDeployments()
	if len(pending) != numGoroutines {
		t.Errorf("expected %d pending deployments after concurrent access, got %d", numGoroutines, len(pending))
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = p.ApproveDeployment("event-" + string(rune('a'+id)))
		}(i)
	}

	wg.Wait()

	pending = p.PendingDeployments()
	if len(pending) != 0 {
		t.Errorf("expected 0 pending deployments after concurrent approval, got %d", len(pending))
	}
}

func TestSlackNotifyPlugin_Init(t *testing.T) {
	p := NewSlackNotifyPlugin()

	err := p.Init(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil config")
	}

	err = p.Init(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing webhook_url")
	}

	err = p.Init(context.Background(), map[string]interface{}{"webhook_url": "https://hooks.slack.com/services/test"})
	if err != nil {
		t.Fatalf("Init with valid config failed: %v", err)
	}

	if err := p.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestSlackNotifyPlugin_Name(t *testing.T) {
	p := NewSlackNotifyPlugin()
	if p.Name() != "slack-notify" {
		t.Errorf("expected name 'slack-notify', got %q", p.Name())
	}
	if p.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", p.Version())
	}
}

func TestSlackNotifyPlugin_OnEvent_NotDeploy(t *testing.T) {
	p := NewSlackNotifyPlugin()
	if err := p.Init(context.Background(), map[string]interface{}{"webhook_url": "https://hooks.slack.com/services/test"}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	p.OnEvent(plugin.BusEvent{Type: "other", Topic: "test"})
}

func TestSlackNotifyPlugin_OnEvent_Deploy(t *testing.T) {
	p := NewSlackNotifyPlugin()
	if err := p.Init(context.Background(), map[string]interface{}{"webhook_url": "https://hooks.slack.com/services/test"}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	p.OnEvent(plugin.BusEvent{Type: "deploy", Topic: "app-1", Payload: map[string]string{"key": "value"}})
}

func newDeployGatePlugin() *DeployGatePlugin {
	return &DeployGatePlugin{
		pending: make(map[string]plugin.BusEvent),
	}
}
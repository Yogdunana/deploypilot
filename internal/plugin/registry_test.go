package plugin

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if r.descriptors == nil {
		t.Error("descriptors map should be initialized")
	}
	if r.instances == nil {
		t.Error("instances map should be initialized")
	}
}

func TestGlobal(t *testing.T) {
	ResetGlobal()

	g1 := Global()
	g2 := Global()

	if g1 != g2 {
		t.Error("Global() should return the same instance")
	}
}

func TestRegister(t *testing.T) {
	r := NewRegistry()

	desc := &PluginDescriptor{
		Name:        "test-plugin",
		DisplayName: "Test Plugin",
		Version:     "1.0.0",
		Provider:    "dns",
		Type:        "cloudflare",
		Factory: func(cfg map[string]interface{}) (interface{}, error) {
			return "instance", nil
		},
	}

	err := r.Register(desc)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	got, ok := r.GetDescriptor("dns", "cloudflare")
	if !ok {
		t.Fatal("GetDescriptor() should find the registered plugin")
	}
	if got.Name != "test-plugin" {
		t.Errorf("Name = %q, want %q", got.Name, "test-plugin")
	}
}

func TestRegisterDuplicate(t *testing.T) {
	r := NewRegistry()

	desc := &PluginDescriptor{
		Name:     "dup",
		Provider: "dns",
		Type:     "cloudflare",
		Factory:  func(cfg map[string]interface{}) (interface{}, error) { return nil, nil },
	}

	err := r.Register(desc)
	if err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	err = r.Register(desc)
	if err == nil {
		t.Error("second Register() should fail for duplicate")
	}
}

func TestRegisterNilDescriptor(t *testing.T) {
	r := NewRegistry()

	err := r.Register(nil)
	if err == nil {
		t.Error("Register(nil) should fail")
	}
}

func TestRegisterEmptyProvider(t *testing.T) {
	r := NewRegistry()

	desc := &PluginDescriptor{
		Name:     "test",
		Provider: "",
		Type:     "cloudflare",
		Factory:  func(cfg map[string]interface{}) (interface{}, error) { return nil, nil },
	}

	err := r.Register(desc)
	if err == nil {
		t.Error("Register() with empty provider should fail")
	}
}

func TestRegisterEmptyType(t *testing.T) {
	r := NewRegistry()

	desc := &PluginDescriptor{
		Name:     "test",
		Provider: "dns",
		Type:     "",
		Factory:  func(cfg map[string]interface{}) (interface{}, error) { return nil, nil },
	}

	err := r.Register(desc)
	if err == nil {
		t.Error("Register() with empty type should fail")
	}
}

func TestRegisterNilFactory(t *testing.T) {
	r := NewRegistry()

	desc := &PluginDescriptor{
		Name:     "test",
		Provider: "dns",
		Type:     "cloudflare",
		Factory:  nil,
	}

	err := r.Register(desc)
	if err == nil {
		t.Error("Register() with nil factory should fail")
	}
}

func TestUnregister(t *testing.T) {
	r := NewRegistry()

	desc := &PluginDescriptor{
		Name:     "to-remove",
		Provider: "dns",
		Type:     "cloudflare",
		Factory:  func(cfg map[string]interface{}) (interface{}, error) { return nil, nil },
	}
	r.Register(desc)

	err := r.Unregister("dns", "cloudflare")
	if err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	_, ok := r.GetDescriptor("dns", "cloudflare")
	if ok {
		t.Error("GetDescriptor() should not find unregistered plugin")
	}
}

func TestUnregisterNotFound(t *testing.T) {
	r := NewRegistry()

	err := r.Unregister("dns", "nonexistent")
	if err == nil {
		t.Error("Unregister() should fail for nonexistent plugin")
	}
}

func TestGetDescriptor(t *testing.T) {
	r := NewRegistry()

	r.Register(&PluginDescriptor{
		Name:     "test-dns",
		Provider: "dns",
		Type:     "cloudflare",
		Factory:  func(cfg map[string]interface{}) (interface{}, error) { return nil, nil },
	})

	// Found
	desc, ok := r.GetDescriptor("dns", "cloudflare")
	if !ok {
		t.Fatal("expected to find descriptor")
	}
	if desc.Name != "test-dns" {
		t.Errorf("Name = %q, want %q", desc.Name, "test-dns")
	}

	// Not found
	_, ok = r.GetDescriptor("dns", "nonexistent")
	if ok {
		t.Error("should not find nonexistent descriptor")
	}
}

func TestListDescriptors(t *testing.T) {
	r := NewRegistry()

	r.Register(&PluginDescriptor{
		Name:     "dns-1",
		Provider: "dns",
		Type:     "cloudflare",
		Factory:  func(cfg map[string]interface{}) (interface{}, error) { return nil, nil },
	})
	r.Register(&PluginDescriptor{
		Name:     "dns-2",
		Provider: "dns",
		Type:     "alidns",
		Factory:  func(cfg map[string]interface{}) (interface{}, error) { return nil, nil },
	})
	r.Register(&PluginDescriptor{
		Name:     "notify-1",
		Provider: "notify",
		Type:     "webhook",
		Factory:  func(cfg map[string]interface{}) (interface{}, error) { return nil, nil },
	})

	// List all
	all := r.ListDescriptors("")
	if len(all) != 3 {
		t.Errorf("ListDescriptors('') count = %d, want 3", len(all))
	}

	// List filtered by provider
	dnsList := r.ListDescriptors("dns")
	if len(dnsList) != 2 {
		t.Errorf("ListDescriptors('dns') count = %d, want 2", len(dnsList))
	}

	notifyList := r.ListDescriptors("notify")
	if len(notifyList) != 1 {
		t.Errorf("ListDescriptors('notify') count = %d, want 1", len(notifyList))
	}

	// List with non-existent provider
	emptyList := r.ListDescriptors("nonexistent")
	if len(emptyList) != 0 {
		t.Errorf("ListDescriptors('nonexistent') count = %d, want 0", len(emptyList))
	}
}

func TestCreateInstance(t *testing.T) {
	r := NewRegistry()

	desc := &PluginDescriptor{
		Name:     "instance-test",
		Provider: "dns",
		Type:     "cloudflare",
		Factory: func(cfg map[string]interface{}) (interface{}, error) {
			name, _ := cfg["name"].(string)
			return fmt.Sprintf("provider-%s", name), nil
		},
	}

	instance, err := r.CreateInstance("plg-001", desc, map[string]interface{}{"name": "test"})
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}

	result, ok := instance.(string)
	if !ok {
		t.Fatal("instance should be a string")
	}
	if result != "provider-test" {
		t.Errorf("instance = %q, want %q", result, "provider-test")
	}

	// Verify it's stored
	stored, ok := r.GetInstance("plg-001")
	if !ok {
		t.Fatal("GetInstance() should find the created instance")
	}
	if stored.(string) != "provider-test" {
		t.Errorf("stored = %q, want %q", stored, "provider-test")
	}
}

func TestCreateInstanceFactoryError(t *testing.T) {
	r := NewRegistry()

	desc := &PluginDescriptor{
		Name:     "error-plugin",
		Provider: "dns",
		Type:     "cloudflare",
		Factory: func(cfg map[string]interface{}) (interface{}, error) {
			return nil, errors.New("factory error")
		},
	}

	_, err := r.CreateInstance("plg-err", desc, nil)
	if err == nil {
		t.Error("CreateInstance() should fail when factory returns error")
	}
}

func TestCreateInstanceNilDescriptor(t *testing.T) {
	r := NewRegistry()

	_, err := r.CreateInstance("plg-nil", nil, nil)
	if err == nil {
		t.Error("CreateInstance() should fail with nil descriptor")
	}
}

func TestCreateInstanceNilFactory(t *testing.T) {
	r := NewRegistry()

	desc := &PluginDescriptor{
		Name:     "nil-factory",
		Provider: "dns",
		Type:     "cloudflare",
		Factory:  nil,
	}

	_, err := r.CreateInstance("plg-nil-factory", desc, nil)
	if err == nil {
		t.Error("CreateInstance() should fail with nil factory")
	}
}

func TestGetInstance(t *testing.T) {
	r := NewRegistry()

	// Not found
	_, ok := r.GetInstance("nonexistent")
	if ok {
		t.Error("GetInstance() should not find nonexistent instance")
	}

	// Create and find
	desc := &PluginDescriptor{
		Name:     "get-test",
		Provider: "dns",
		Type:     "cloudflare",
		Factory:  func(cfg map[string]interface{}) (interface{}, error) { return "test-instance", nil },
	}
	r.CreateInstance("plg-get", desc, nil)

	instance, ok := r.GetInstance("plg-get")
	if !ok {
		t.Fatal("GetInstance() should find created instance")
	}
	if instance.(string) != "test-instance" {
		t.Errorf("instance = %q, want %q", instance, "test-instance")
	}
}

func TestRemoveInstance(t *testing.T) {
	r := NewRegistry()

	desc := &PluginDescriptor{
		Name:     "remove-test",
		Provider: "dns",
		Type:     "cloudflare",
		Factory:  func(cfg map[string]interface{}) (interface{}, error) { return "to-remove", nil },
	}
	r.CreateInstance("plg-remove", desc, nil)

	r.RemoveInstance("plg-remove")

	_, ok := r.GetInstance("plg-remove")
	if ok {
		t.Error("GetInstance() should not find removed instance")
	}
}

func TestRemoveInstanceNonexistent(t *testing.T) {
	r := NewRegistry()

	// Should not panic
	r.RemoveInstance("nonexistent")
}

func TestDescriptorKey(t *testing.T) {
	key := descriptorKey("dns", "cloudflare")
	if key != "dns:cloudflare" {
		t.Errorf("descriptorKey() = %q, want %q", key, "dns:cloudflare")
	}
}

func TestRegistryConcurrency(t *testing.T) {
	r := NewRegistry()

	desc := &PluginDescriptor{
		Name:     "concurrent",
		Provider: "dns",
		Type:     "cloudflare",
		Factory:  func(cfg map[string]interface{}) (interface{}, error) { return nil, nil },
	}
	r.Register(desc)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.GetDescriptor("dns", "cloudflare")
			r.ListDescriptors("dns")
			r.ListDescriptors("")
		}()
	}
	wg.Wait()
}

func TestResetGlobal(t *testing.T) {
	ResetGlobal()

	g := Global()
	g.Register(&PluginDescriptor{
		Name:     "reset-test",
		Provider: "dns",
		Type:     "cloudflare",
		Factory:  func(cfg map[string]interface{}) (interface{}, error) { return nil, nil },
	})

	// Verify it's there
	_, ok := g.GetDescriptor("dns", "cloudflare")
	if !ok {
		t.Fatal("plugin should be registered before reset")
	}

	// Reset
	ResetGlobal()

	g2 := Global()
	_, ok = g2.GetDescriptor("dns", "cloudflare")
	if ok {
		t.Error("plugin should not exist after reset")
	}
}

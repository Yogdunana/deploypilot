package plugin

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	db.Exec(`CREATE TABLE IF NOT EXISTS plugins (
		id TEXT PRIMARY KEY, tenant_id TEXT, name TEXT NOT NULL,
		display_name TEXT, version TEXT DEFAULT '1.0.0', description TEXT,
		author TEXT, provider TEXT NOT NULL, type TEXT NOT NULL,
		config TEXT, enabled INTEGER DEFAULT 1, priority INTEGER DEFAULT 0,
		status TEXT DEFAULT 'active', error_msg TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(tenant_id, name)
	)`)

	return db
}

func seedPlugin(t *testing.T, db *gorm.DB, id, tenantID, name, provider, pluginType, config string, enabled bool) {
	t.Helper()
	db.Exec(`INSERT INTO plugins (id, tenant_id, name, provider, type, config, enabled, status) VALUES (?, ?, ?, ?, ?, ?, ?, 'active')`,
		id, tenantID, name, provider, pluginType, config, enabled)
}

func TestNewManager(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	r := NewRegistry()
	m := NewManager(r, db, "")

	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if m.registry != r {
		t.Error("registry not set correctly")
	}
}

func TestLoadAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	r := NewRegistry()
	r.Register(&PluginDescriptor{
		Name:     "test-dns",
		Provider: "dns",
		Type:     "cloudflare",
		Factory: func(cfg map[string]interface{}) (interface{}, error) {
			return "dns-instance", nil
		},
	})

	m := NewManager(r, db, "")

	// Seed two enabled plugins and one disabled
	seedPlugin(t, db, "plg-1", "tenant-default", "dns-1", "dns", "cloudflare", `{"api_token":"tok1"}`, true)
	seedPlugin(t, db, "plg-2", "tenant-default", "dns-2", "dns", "cloudflare", `{"api_token":"tok2"}`, true)
	seedPlugin(t, db, "plg-3", "tenant-default", "dns-3", "dns", "cloudflare", `{"api_token":"tok3"}`, false)

	err := m.LoadAll(context.Background(), "tenant-default")
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	// Check that enabled plugins have instances
	for _, id := range []string{"plg-1", "plg-2"} {
		instance, ok := m.registry.GetInstance(id)
		if !ok {
			t.Errorf("instance not found for %s", id)
		}
		if instance.(string) != "dns-instance" {
			t.Errorf("instance = %v, want dns-instance", instance)
		}
	}

	// Disabled plugin should not have instance
	_, ok := m.registry.GetInstance("plg-3")
	if ok {
		t.Error("disabled plugin should not have instance")
	}
}

func TestLoadAllWithFactoryError(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	r := NewRegistry()
	r.Register(&PluginDescriptor{
		Name:     "error-plugin",
		Provider: "dns",
		Type:     "cloudflare",
		Factory: func(cfg map[string]interface{}) (interface{}, error) {
			return nil, errors.New("factory error")
		},
	})

	m := NewManager(r, db, "")
	seedPlugin(t, db, "plg-err", "tenant-default", "dns-err", "dns", "cloudflare", `{}`, true)

	// LoadAll should not fail even if individual plugin fails
	err := m.LoadAll(context.Background(), "tenant-default")
	if err != nil {
		t.Fatalf("LoadAll() should not fail: %v", err)
	}

	// Plugin status should be error
	var status string
	db.Table("plugins").Where("id = ?", "plg-err").Select("status").Scan(&status)
	if status != "error" {
		t.Errorf("plugin status = %q, want %q", status, "error")
	}
}

func TestLoadPlugin(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	r := NewRegistry()
	r.Register(&PluginDescriptor{
		Name:     "load-test",
		Provider: "dns",
		Type:     "cloudflare",
		Factory: func(cfg map[string]interface{}) (interface{}, error) {
			return "loaded-instance", nil
		},
	})

	m := NewManager(r, db, "")
	seedPlugin(t, db, "plg-load", "tenant-default", "dns-load", "dns", "cloudflare", `{}`, true)

	err := m.LoadPlugin(context.Background(), "plg-load")
	if err != nil {
		t.Fatalf("LoadPlugin() error = %v", err)
	}

	instance, ok := m.registry.GetInstance("plg-load")
	if !ok {
		t.Fatal("instance not found after LoadPlugin")
	}
	if instance.(string) != "loaded-instance" {
		t.Errorf("instance = %v, want loaded-instance", instance)
	}
}

func TestLoadPluginDisabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	r := NewRegistry()
	r.Register(&PluginDescriptor{
		Name:     "disabled-test",
		Provider: "dns",
		Type:     "cloudflare",
		Factory: func(cfg map[string]interface{}) (interface{}, error) {
			return "should-not-load", nil
		},
	})

	m := NewManager(r, db, "")
	seedPlugin(t, db, "plg-disabled", "tenant-default", "dns-disabled", "dns", "cloudflare", `{}`, false)

	err := m.LoadPlugin(context.Background(), "plg-disabled")
	if err == nil {
		t.Fatal("LoadPlugin() should fail for disabled plugin")
	}
}

func TestLoadPluginNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	r := NewRegistry()
	m := NewManager(r, db, "")

	err := m.LoadPlugin(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("LoadPlugin() should fail for nonexistent plugin")
	}
}

func TestLoadPluginNoDescriptor(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	r := NewRegistry()
	// No descriptors registered
	m := NewManager(r, db, "")
	seedPlugin(t, db, "plg-nodesc", "tenant-default", "dns-nodesc", "dns", "cloudflare", `{}`, true)

	err := m.LoadPlugin(context.Background(), "plg-nodesc")
	if err == nil {
		t.Fatal("LoadPlugin() should fail when no descriptor found")
	}
}

func TestUnloadPlugin(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	r := NewRegistry()
	r.Register(&PluginDescriptor{
		Name:     "unload-test",
		Provider: "dns",
		Type:     "cloudflare",
		Factory: func(cfg map[string]interface{}) (interface{}, error) {
			return "to-unload", nil
		},
	})

	m := NewManager(r, db, "")
	seedPlugin(t, db, "plg-unload", "tenant-default", "dns-unload", "dns", "cloudflare", `{}`, true)

	_ = m.LoadPlugin(context.Background(), "plg-unload")

	err := m.UnloadPlugin("plg-unload")
	if err != nil {
		t.Fatalf("UnloadPlugin() error = %v", err)
	}

	_, ok := m.registry.GetInstance("plg-unload")
	if ok {
		t.Error("instance should be removed after UnloadPlugin")
	}

	// Status should be disabled
	var status string
	db.Table("plugins").Where("id = ?", "plg-unload").Select("status").Scan(&status)
	if status != "disabled" {
		t.Errorf("status = %q, want %q", status, "disabled")
	}
}

func TestReloadPlugin(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	loadCount := 0
	r := NewRegistry()
	r.Register(&PluginDescriptor{
		Name:     "reload-test",
		Provider: "dns",
		Type:     "cloudflare",
		Factory: func(cfg map[string]interface{}) (interface{}, error) {
			loadCount++
			return fmt.Sprintf("instance-%d", loadCount), nil
		},
	})

	m := NewManager(r, db, "")
	seedPlugin(t, db, "plg-reload", "tenant-default", "dns-reload", "dns", "cloudflare", `{}`, true)

	_ = m.LoadPlugin(context.Background(), "plg-reload")
	if loadCount != 1 {
		t.Fatalf("expected 1 load, got %d", loadCount)
	}

	err := m.ReloadPlugin(context.Background(), "plg-reload")
	if err != nil {
		t.Fatalf("ReloadPlugin() error = %v", err)
	}
	if loadCount != 2 {
		t.Errorf("expected 2 loads after reload, got %d", loadCount)
	}

	instance, ok := m.registry.GetInstance("plg-reload")
	if !ok {
		t.Fatal("instance not found after reload")
	}
	if instance.(string) != "instance-2" {
		t.Errorf("instance = %v, want instance-2", instance)
	}
}

func TestEnablePlugin(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	r := NewRegistry()
	r.Register(&PluginDescriptor{
		Name:     "enable-test",
		Provider: "dns",
		Type:     "cloudflare",
		Factory: func(cfg map[string]interface{}) (interface{}, error) {
			return "enabled-instance", nil
		},
	})

	m := NewManager(r, db, "")
	seedPlugin(t, db, "plg-enable", "tenant-default", "dns-enable", "dns", "cloudflare", `{}`, false)

	err := m.EnablePlugin(context.Background(), "plg-enable")
	if err != nil {
		t.Fatalf("EnablePlugin() error = %v", err)
	}

	// Check enabled flag
	var enabled bool
	db.Table("plugins").Where("id = ?", "plg-enable").Select("enabled").Scan(&enabled)
	if !enabled {
		t.Error("plugin should be enabled in DB")
	}

	// Check instance loaded
	instance, ok := m.registry.GetInstance("plg-enable")
	if !ok {
		t.Fatal("instance should be loaded after EnablePlugin")
	}
	if instance.(string) != "enabled-instance" {
		t.Errorf("instance = %v, want enabled-instance", instance)
	}
}

func TestDisablePlugin(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	r := NewRegistry()
	r.Register(&PluginDescriptor{
		Name:     "disable-test",
		Provider: "dns",
		Type:     "cloudflare",
		Factory: func(cfg map[string]interface{}) (interface{}, error) {
			return "will-be-disabled", nil
		},
	})

	m := NewManager(r, db, "")
	seedPlugin(t, db, "plg-disable", "tenant-default", "dns-disable", "dns", "cloudflare", `{}`, true)

	// Load first
	_ = m.LoadPlugin(context.Background(), "plg-disable")

	err := m.DisablePlugin(context.Background(), "plg-disable")
	if err != nil {
		t.Fatalf("DisablePlugin() error = %v", err)
	}

	// Check disabled flag
	var enabled bool
	db.Table("plugins").Where("id = ?", "plg-disable").Select("enabled").Scan(&enabled)
	if enabled {
		t.Error("plugin should be disabled in DB")
	}

	// Check instance removed
	_, ok := m.registry.GetInstance("plg-disable")
	if ok {
		t.Error("instance should be removed after DisablePlugin")
	}
}

func TestGetPluginInstance(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	r := NewRegistry()
	r.Register(&PluginDescriptor{
		Name:     "get-test",
		Provider: "dns",
		Type:     "cloudflare",
		Factory: func(cfg map[string]interface{}) (interface{}, error) {
			return "my-instance", nil
		},
	})

	m := NewManager(r, db, "")
	seedPlugin(t, db, "plg-get", "tenant-default", "dns-get", "dns", "cloudflare", `{}`, true)
	_ = m.LoadPlugin(context.Background(), "plg-get")

	instance, err := m.GetPluginInstance("plg-get")
	if err != nil {
		t.Fatalf("GetPluginInstance() error = %v", err)
	}
	if instance.(string) != "my-instance" {
		t.Errorf("instance = %v, want my-instance", instance)
	}

	// Nonexistent
	_, err = m.GetPluginInstance("nonexistent")
	if err == nil {
		t.Error("GetPluginInstance() should fail for nonexistent plugin")
	}
}

func TestGetPluginStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	r := NewRegistry()
	r.Register(&PluginDescriptor{
		Name:     "status-test",
		Provider: "dns",
		Type:     "cloudflare",
		Factory: func(cfg map[string]interface{}) (interface{}, error) {
			return "status-instance", nil
		},
	})

	m := NewManager(r, db, "")
	seedPlugin(t, db, "plg-status", "tenant-default", "dns-status", "dns", "cloudflare", `{}`, true)

	status, err := m.GetPluginStatus("tenant-default")
	if err != nil {
		t.Fatalf("GetPluginStatus() error = %v", err)
	}
	if len(status) != 1 {
		t.Fatalf("expected 1 plugin status, got %d", len(status))
	}

	// Before loading, instance_loaded should be false
	if loaded, ok := status[0]["instance_loaded"].(bool); !ok || loaded {
		t.Error("instance_loaded should be false before loading")
	}

	// Load and check again
	_ = m.LoadPlugin(context.Background(), "plg-status")

	status, err = m.GetPluginStatus("tenant-default")
	if err != nil {
		t.Fatalf("GetPluginStatus() after load error = %v", err)
	}
	if loaded, ok := status[0]["instance_loaded"].(bool); !ok || !loaded {
		t.Error("instance_loaded should be true after loading")
	}
}

func TestEnablePluginNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	r := NewRegistry()
	m := NewManager(r, db, "")

	err := m.EnablePlugin(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("EnablePlugin() should fail for nonexistent plugin")
	}
}

func TestDisablePluginNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.DB.Close()

	r := NewRegistry()
	m := NewManager(r, db, "")

	err := m.DisablePlugin(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("DisablePlugin() should fail for nonexistent plugin")
	}
}

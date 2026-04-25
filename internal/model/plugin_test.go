package model

import (
	"testing"

	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/database"
)

func setupPluginDB(t *testing.T) func() {
	t.Helper()
	tmpDir := t.TempDir()
	db, err := database.Connect("sqlite", tmpDir+"/test.db")
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err := database.Seed(db); err != nil {
		t.Fatalf("Seed() error = %v", err)
	}
	encKey := crypto.NewEncryptionKey()
	InitDB(db, encKey)
	return func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}
}

func TestCreatePlugin(t *testing.T) {
	cleanup := setupPluginDB(t)
	defer cleanup()

	plugin, err := CreatePlugin(
		"tenant-default", "my-cloudflare", "Cloudflare DNS",
		"1.0.0", "Cloudflare DNS plugin", "DeployPilot",
		"dns", "cloudflare", `{"api_token": "test"}`,
		true, 10,
	)
	if err != nil {
		t.Fatalf("CreatePlugin() error = %v", err)
	}

	if plugin.ID == "" {
		t.Error("ID should not be empty")
	}
	if len(plugin.ID) < 4 || plugin.ID[:4] != "plg-" {
		t.Errorf("ID = %q, want prefix 'plg-'", plugin.ID)
	}
	if plugin.Name != "my-cloudflare" {
		t.Errorf("Name = %q, want %q", plugin.Name, "my-cloudflare")
	}
	if plugin.DisplayName != "Cloudflare DNS" {
		t.Errorf("DisplayName = %q, want %q", plugin.DisplayName, "Cloudflare DNS")
	}
	if plugin.Provider != "dns" {
		t.Errorf("Provider = %q, want %q", plugin.Provider, "dns")
	}
	if plugin.Type != "cloudflare" {
		t.Errorf("Type = %q, want %q", plugin.Type, "cloudflare")
	}
	if plugin.Status != "active" {
		t.Errorf("Status = %q, want %q", plugin.Status, "active")
	}
	if !plugin.Enabled {
		t.Error("Enabled should be true")
	}
	if plugin.Priority != 10 {
		t.Errorf("Priority = %d, want 10", plugin.Priority)
	}
}

func TestGetPlugin(t *testing.T) {
	cleanup := setupPluginDB(t)
	defer cleanup()

	created, _ := CreatePlugin(
		"tenant-default", "get-test", "Get Test",
		"1.0.0", "test plugin", "DeployPilot",
		"dns", "cloudflare", `{}`,
		true, 0,
	)

	got, err := GetPlugin(created.ID)
	if err != nil {
		t.Fatalf("GetPlugin() error = %v", err)
	}

	if got.Name != "get-test" {
		t.Errorf("Name = %q, want %q", got.Name, "get-test")
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
}

func TestGetPluginNotFound(t *testing.T) {
	cleanup := setupPluginDB(t)
	defer cleanup()

	_, err := GetPlugin("nonexistent-id")
	if err == nil {
		t.Error("GetPlugin() should fail for nonexistent ID")
	}
}

func TestListPlugins(t *testing.T) {
	cleanup := setupPluginDB(t)
	defer cleanup()

	CreatePlugin("tenant-default", "plugin-1", "Plugin 1", "1.0.0", "", "", "dns", "cloudflare", `{}`, true, 5)
	CreatePlugin("tenant-default", "plugin-2", "Plugin 2", "1.0.0", "", "", "notify", "webhook", `{}`, true, 10)
	CreatePlugin("tenant-default", "plugin-3", "Plugin 3", "1.0.0", "", "", "dns", "alidns", `{}`, true, 3)

	// List all plugins for tenant
	plugins, err := ListPlugins("tenant-default", "")
	if err != nil {
		t.Fatalf("ListPlugins() error = %v", err)
	}
	if len(plugins) != 3 {
		t.Errorf("count = %d, want 3", len(plugins))
	}

	// List filtered by provider
	dnsPlugins, err := ListPlugins("tenant-default", "dns")
	if err != nil {
		t.Fatalf("ListPlugins(dns) error = %v", err)
	}
	if len(dnsPlugins) != 2 {
		t.Errorf("dns count = %d, want 2", len(dnsPlugins))
	}

	// Verify priority ordering (plugin-2 has priority 10, should be first)
	if plugins[0].Name != "plugin-2" {
		t.Errorf("first plugin by priority = %q, want %q", plugins[0].Name, "plugin-2")
	}
}

func TestListPluginsByTenant(t *testing.T) {
	cleanup := setupPluginDB(t)
	defer cleanup()

	CreatePlugin("tenant-default", "plugin-a", "A", "1.0.0", "", "", "dns", "cloudflare", `{}`, true, 0)
	CreatePlugin("tenant-other", "plugin-b", "B", "1.0.0", "", "", "dns", "cloudflare", `{}`, true, 0)

	plugins, _ := ListPlugins("tenant-default", "")
	if len(plugins) != 1 {
		t.Errorf("count = %d, want 1 (filtered by tenant)", len(plugins))
	}
}

func TestListPluginsEmpty(t *testing.T) {
	cleanup := setupPluginDB(t)
	defer cleanup()

	plugins, err := ListPlugins("tenant-nonexistent", "")
	if err != nil {
		t.Fatalf("ListPlugins() error = %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("count = %d, want 0", len(plugins))
	}
}

func TestUpdatePlugin(t *testing.T) {
	cleanup := setupPluginDB(t)
	defer cleanup()

	created, _ := CreatePlugin(
		"tenant-default", "update-test", "Update Test",
		"1.0.0", "original description", "DeployPilot",
		"dns", "cloudflare", `{}`,
		true, 0,
	)

	updated, err := UpdatePlugin(created.ID, map[string]interface{}{
		"display_name": "Updated Name",
		"description":  "new description",
		"priority":     20,
	})
	if err != nil {
		t.Fatalf("UpdatePlugin() error = %v", err)
	}

	if updated.DisplayName != "Updated Name" {
		t.Errorf("DisplayName = %q, want %q", updated.DisplayName, "Updated Name")
	}
	if updated.Description != "new description" {
		t.Errorf("Description = %q, want %q", updated.Description, "new description")
	}
	if updated.Priority != 20 {
		t.Errorf("Priority = %d, want 20", updated.Priority)
	}
}

func TestUpdatePluginNotFound(t *testing.T) {
	cleanup := setupPluginDB(t)
	defer cleanup()

	_, err := UpdatePlugin("nonexistent-id", map[string]interface{}{
		"display_name": "test",
	})
	if err == nil {
		t.Error("UpdatePlugin() should fail for nonexistent ID")
	}
}

func TestUpdatePluginEmptyUpdates(t *testing.T) {
	cleanup := setupPluginDB(t)
	defer cleanup()

	created, _ := CreatePlugin(
		"tenant-default", "empty-update", "Empty Update",
		"1.0.0", "", "", "dns", "cloudflare", `{}`,
		true, 0,
	)

	got, err := UpdatePlugin(created.ID, map[string]interface{}{})
	if err != nil {
		t.Fatalf("UpdatePlugin() with empty updates error = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
}

func TestDeletePlugin(t *testing.T) {
	cleanup := setupPluginDB(t)
	defer cleanup()

	created, _ := CreatePlugin(
		"tenant-default", "delete-test", "Delete Test",
		"1.0.0", "", "", "dns", "cloudflare", `{}`,
		true, 0,
	)

	err := DeletePlugin(created.ID)
	if err != nil {
		t.Fatalf("DeletePlugin() error = %v", err)
	}

	_, err = GetPlugin(created.ID)
	if err == nil {
		t.Error("GetPlugin() should fail after delete")
	}
}

func TestDeletePluginNotFound(t *testing.T) {
	cleanup := setupPluginDB(t)
	defer cleanup()

	err := DeletePlugin("nonexistent-id")
	if err == nil {
		t.Error("DeletePlugin() should fail for nonexistent ID")
	}
}

func TestPluginTableName(t *testing.T) {
	p := Plugin{}
	if p.TableName() != "plugins" {
		t.Errorf("TableName() = %q, want %q", p.TableName(), "plugins")
	}
}

func TestPluginRoundTrip(t *testing.T) {
	cleanup := setupPluginDB(t)
	defer cleanup()

	// Create
	plugin, _ := CreatePlugin(
		"tenant-default", "round-trip", "Round Trip",
		"2.0.0", "round trip test", "TestAuthor",
		"notify", "webhook", `{"url": "https://example.com"}`,
		true, 5,
	)

	// Get and verify
	got, _ := GetPlugin(plugin.ID)
	if got.Name != "round-trip" {
		t.Errorf("round-trip get failed: Name = %q", got.Name)
	}
	if got.Config != `{"url": "https://example.com"}` {
		t.Errorf("round-trip get failed: Config = %q", got.Config)
	}

	// Update
	UpdatePlugin(plugin.ID, map[string]interface{}{
		"enabled": false,
		"status":  "disabled",
	})

	// Get again
	got2, _ := GetPlugin(plugin.ID)
	if got2.Enabled != false {
		t.Errorf("round-trip update failed: Enabled = %v", got2.Enabled)
	}
	if got2.Status != "disabled" {
		t.Errorf("round-trip update failed: Status = %q", got2.Status)
	}

	// Delete
	DeletePlugin(plugin.ID)

	// Verify deleted
	_, err := GetPlugin(plugin.ID)
	if err == nil {
		t.Error("should fail after delete in round-trip")
	}
}

func TestCreatePluginDefaults(t *testing.T) {
	cleanup := setupPluginDB(t)
	defer cleanup()

	plugin, _ := CreatePlugin(
		"tenant-default", "defaults-test", "",
		"", "", "",
		"dns", "cloudflare", `{}`,
		true, 0,
	)

	if plugin.Version != "1.0.0" {
		t.Errorf("Version = %q, want default '1.0.0'", plugin.Version)
	}
	if plugin.Status != "active" {
		t.Errorf("Status = %q, want default 'active'", plugin.Status)
	}
	if plugin.Enabled != true {
		t.Error("Enabled should be true")
	}
	if plugin.Priority != 0 {
		t.Errorf("Priority = %d, want default 0", plugin.Priority)
	}
}

package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// ========== Watcher (Hot Reload) ==========

func TestWatchConfigChange(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`server:
  port: 8080
`)
	os.WriteFile(configPath, content, 0644)

	mgr := NewManager(configPath)
	defer mgr.Close()

	// Initial load
	cfg := mgr.GetConfig()
	if cfg.Server.Port != 8080 {
		t.Fatalf("initial Port = %d, want 8080", cfg.Server.Port)
	}

	// Modify config file
	newContent := []byte(`server:
  port: 9999
  host: "0.0.0.0"
`)
	os.WriteFile(configPath, newContent, 0644)

	// Trigger manual reload (simulates what watchLoop does)
	err := mgr.Reload()
	if err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	cfg = mgr.GetConfig()
	if cfg.Server.Port != 9999 {
		t.Errorf("after reload Port = %d, want 9999", cfg.Server.Port)
	}
}

func TestWatchConfigCallback(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`server:
  port: 8080
`)
	os.WriteFile(configPath, content, 0644)

	mgr := NewManager(configPath)
	defer mgr.Close()

	var callbackCount int
	var mu sync.Mutex
	mgr.OnChange(func(oldCfg, newCfg *Config) {
		mu.Lock()
		callbackCount++
		mu.Unlock()
	})

	// Modify config and trigger reload
	newContent := []byte(`server:
  port: 7777
  host: "0.0.0.0"
`)
	os.WriteFile(configPath, newContent, 0644)
	mgr.Reload()

	mu.Lock()
	if callbackCount < 1 {
		t.Errorf("OnChange callback not called, count = %d", callbackCount)
	}
	mu.Unlock()
}

func TestManagerGetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`server:
  port: 5555
  host: "192.168.1.1"
`)
	os.WriteFile(configPath, content, 0644)

	mgr := NewManager(configPath)
	defer mgr.Close()

	cfg := mgr.GetConfig()
	if cfg == nil {
		t.Fatal("GetConfig() should not return nil")
	}
	if cfg.Server.Port != 5555 {
		t.Errorf("Port = %d, want 5555", cfg.Server.Port)
	}
	if cfg.Server.Host != "192.168.1.1" {
		t.Errorf("Host = %q", cfg.Server.Host)
	}
}

func TestManagerReload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`server:
  port: 8080
`)
	os.WriteFile(configPath, content, 0644)

	mgr := NewManager(configPath)
	defer mgr.Close()

	// Modify file
	newContent := []byte(`server:
  port: 4444
`)
	os.WriteFile(configPath, newContent, 0644)

	// Manual reload
	err := mgr.Reload()
	if err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	cfg := mgr.GetConfig()
	if cfg.Server.Port != 4444 {
		t.Errorf("after Reload() Port = %d, want 4444", cfg.Server.Port)
	}
}

func TestManagerReloadInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`server:
  port: 8080
`)
	os.WriteFile(configPath, content, 0644)

	mgr := NewManager(configPath)
	defer mgr.Close()

	// Write invalid YAML
	os.WriteFile(configPath, []byte(`{{invalid yaml`), 0644)

	err := mgr.Reload()
	if err == nil {
		t.Error("Reload() should fail for invalid YAML")
	}

	// Config should remain unchanged
	cfg := mgr.GetConfig()
	if cfg.Server.Port != 8080 {
		t.Errorf("Port should remain 8080 after failed reload, got %d", cfg.Server.Port)
	}
}

func TestManagerClose(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`server:
  port: 8080
`)
	os.WriteFile(configPath, content, 0644)

	mgr := NewManager(configPath)
	mgr.Close()

	// Should not panic
	cfg := mgr.GetConfig()
	if cfg == nil {
		t.Error("GetConfig() should still work after Close")
	}
}

func TestManagerGetConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`server:
  port: 8080
`)
	os.WriteFile(configPath, content, 0644)

	mgr := NewManager(configPath)
	defer mgr.Close()

	if mgr.GetConfigPath() != configPath {
		t.Errorf("GetConfigPath() = %q, want %q", mgr.GetConfigPath(), configPath)
	}
}

func TestManagerGetVersion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`server:
  port: 8080
`)
	os.WriteFile(configPath, content, 0644)

	mgr := NewManager(configPath)
	defer mgr.Close()

	v1 := mgr.GetVersion()
	mgr.Reload()
	v2 := mgr.GetVersion()

	if v2 <= v1 {
		t.Errorf("version should increment after reload: v1=%d, v2=%d", v1, v2)
	}
}

func TestOnChangeMultipleCallbacks(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`server:
  port: 8080
`)
	os.WriteFile(configPath, content, 0644)

	mgr := NewManager(configPath)
	defer mgr.Close()

	var count1, count2 int
	var mu sync.Mutex

	mgr.OnChange(func(_, _ *Config) {
		mu.Lock()
		count1++
		mu.Unlock()
	})
	mgr.OnChange(func(_, _ *Config) {
		mu.Lock()
		count2++
		mu.Unlock()
	})

	newContent := []byte(`server:
  port: 1111
`)
	os.WriteFile(configPath, newContent, 0644)
	mgr.Reload()

	mu.Lock()
	if count1 != 1 {
		t.Errorf("callback1 count = %d, want 1", count1)
	}
	if count2 != 1 {
		t.Errorf("callback2 count = %d, want 1", count2)
	}
	mu.Unlock()
}

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`
server:
  host: "0.0.0.0"
  port: 8080
  mcp_port: 9090
  web_port: 3000
database:
  type: "sqlite"
  dsn: "./data/test.db"
auth:
  jwt_secret: "test-secret"
  token_expire: "24h"
  ws_ticket_expire: "30s"
deploy:
  default_mode: "api"
  build_timeout: "10m"
  health_check_interval: "30s"
  health_check_retries: 3
  rollback_on_failure: true
  sandbox_cpu: 2
  sandbox_memory: "4GB"
  runtime_cpu: 2
  runtime_memory: "2GB"
cache:
  build_cache_ttl: "720h"
  build_cache_max: 50
security:
  rate_limit_default: 100
  rate_limit_owner: 200
  rate_limit_admin: 150
  rate_limit_dev: 100
  rate_limit_viewer: 50
log:
  level: "debug"
  format: "json"
  file: "./logs/test.log"
  max_size: "100MB"
  max_backups: 10
notify:
  default_channels:
    - "webhook"
monitor:
  enabled: true
  metrics_port: 9091
`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Server
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %q, want %q", cfg.Server.Host, "0.0.0.0")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 8080)
	}
	if cfg.Server.MCPPort != 9090 {
		t.Errorf("Server.MCPPort = %d, want %d", cfg.Server.MCPPort, 9090)
	}
	if cfg.Server.WebPort != 3000 {
		t.Errorf("Server.WebPort = %d, want %d", cfg.Server.WebPort, 3000)
	}

	// Database
	if cfg.Database.Type != "sqlite" {
		t.Errorf("Database.Type = %q, want %q", cfg.Database.Type, "sqlite")
	}
	if cfg.Database.DSN != "./data/test.db" {
		t.Errorf("Database.DSN = %q, want %q", cfg.Database.DSN, "./data/test.db")
	}

	// Auth
	if cfg.Auth.JWTSecret != "test-secret" {
		t.Errorf("Auth.JWTSecret = %q, want %q", cfg.Auth.JWTSecret, "test-secret")
	}
	if cfg.Auth.TokenExpire != "24h" {
		t.Errorf("Auth.TokenExpire = %q, want %q", cfg.Auth.TokenExpire, "24h")
	}

	// Deploy
	if cfg.Deploy.DefaultMode != "api" {
		t.Errorf("Deploy.DefaultMode = %q, want %q", cfg.Deploy.DefaultMode, "api")
	}
	if cfg.Deploy.RollbackOnFailure != true {
		t.Errorf("Deploy.RollbackOnFailure = %v, want %v", cfg.Deploy.RollbackOnFailure, true)
	}

	// Log
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "debug")
	}

	// Monitor
	if cfg.Monitor.Enabled != true {
		t.Errorf("Monitor.Enabled = %v, want %v", cfg.Monitor.Enabled, true)
	}
}

func TestLoadFromFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Load() should return error for missing file")
	}
}

func TestLoadDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Minimal config — only override one field
	content := []byte(`server:
  port: 9999
`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Overridden value
	if cfg.Server.Port != 9999 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 9999)
	}

	// Default values should still be present
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %q, want default %q", cfg.Server.Host, "0.0.0.0")
	}
	if cfg.Database.Type != "sqlite" {
		t.Errorf("Database.Type = %q, want default %q", cfg.Database.Type, "sqlite")
	}
}

func TestEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := []byte(`server:
  port: 8080
`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Set env var override
	t.Setenv("DEPLOYPILOT_SERVER_PORT", "7777")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 7777 {
		t.Errorf("Server.Port = %d, want %d (env override)", cfg.Server.Port, 7777)
	}
}

func TestDefaultConfig(t *testing.T) {
    cfg := DefaultConfig()
    if cfg.Database.Type != "sqlite" {
        t.Errorf("expected sqlite, got %s", cfg.Database.Type)
    }
    if cfg.Database.DSN != "./data/deploypilot.db" {
        t.Errorf("expected ./data/deploypilot.db, got %s", cfg.Database.DSN)
    }
}

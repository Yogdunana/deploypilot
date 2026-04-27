package main

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/config"
)

// ========== InitLogger tests ==========

func TestInitLoggerJSON(t *testing.T) {
	cfg := config.LogConfig{Level: "debug", Format: "json"}
	config.InitLogger(cfg)
	// Verify it doesn't panic — logger is now configured
}

func TestInitLoggerText(t *testing.T) {
	cfg := config.LogConfig{Level: "info", Format: "text"}
	config.InitLogger(cfg)
}

func TestInitLoggerDefault(t *testing.T) {
	cfg := config.LogConfig{Level: "unknown", Format: "text"}
	config.InitLogger(cfg)
}

func TestInitLoggerWarn(t *testing.T) {
	cfg := config.LogConfig{Level: "warn", Format: "json"}
	config.InitLogger(cfg)
}

func TestInitLoggerError(t *testing.T) {
	cfg := config.LogConfig{Level: "error", Format: "text"}
	config.InitLogger(cfg)
}

// ========== Version flag test ==========

func TestVersionFlag(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := tmpDir + "/api-server"

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, string(out))
	}

	out, err := exec.Command(binPath, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("--version exited with error: %v\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "api-server") {
		t.Errorf("--version output should contain 'api-server', got: %q", string(out))
	}
	if !strings.Contains(string(out), "dev") {
		t.Errorf("--version output should contain version 'dev', got: %q", string(out))
	}
}

// ========== Config loading via env vars ==========

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}
	if cfg.Database.Type != "sqlite" {
		t.Errorf("Database.Type = %q, want sqlite", cfg.Database.Type)
	}
	if cfg.Database.DSN == "" {
		t.Error("Database.DSN should not be empty")
	}
}

// ========== run() function tests ==========

func TestRunWithPostgresNoDSN(t *testing.T) {
	// Using postgres driver with empty DSN should fail at DB connect
	err := run("", "postgres", "", "")
	if err == nil {
		t.Error("expected error for postgres with no DSN")
	}
}

// ========== localExecutor tests ==========

func TestLocalExecutor(t *testing.T) {
	executor := &localExecutor{}
	out, err := executor.RunCommand(context.TODO(), "echo hello")
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("expected 'hello' in output, got: %q", out)
	}
}

func TestLocalExecutorFailure(t *testing.T) {
	executor := &localExecutor{}
	_, err := executor.RunCommand(context.TODO(), "false")
	if err == nil {
		t.Error("expected error for failing command")
	}
}

// ========== version variable ==========

func TestVersionDefault(t *testing.T) {
	if version != "dev" {
		t.Errorf("version = %q, want %q", version, "dev")
	}
}



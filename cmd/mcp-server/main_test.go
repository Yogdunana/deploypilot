package main

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
)

func TestMainBuilds(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := tmpDir + "/mcp-server"

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build: %v\n%s", err, string(output))
	}
}

func TestVersionFlag(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := tmpDir + "/mcp-server"

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, string(out))
	}

	out, err := exec.Command(binPath, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("--version exited with error: %v\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "mcp-server") {
		t.Errorf("--version output should contain 'mcp-server', got: %q", string(out))
	}
	if !strings.Contains(string(out), "dev") {
		t.Errorf("--version output should contain version 'dev', got: %q", string(out))
	}
}

func TestHelpFlag(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := tmpDir + "/mcp-server"

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, string(out))
	}

	out, err := exec.Command(binPath, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("--help exited with error: %v\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "Usage") {
		t.Errorf("--help output should contain 'Usage', got: %q", string(out))
	}
	if !strings.Contains(string(out), "-db-driver") {
		t.Errorf("--help output should contain '-db-driver', got: %q", string(out))
	}
	if !strings.Contains(string(out), "-db-dsn") {
		t.Errorf("--help output should contain '-db-dsn', got: %q", string(out))
	}
}

func TestMainStartsAndResponds(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := tmpDir + "/mcp-server"

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, string(out))
	}

	// Use --db-dsn to avoid relying on env vars
	runCmd := exec.Command(binPath, "--db-dsn", tmpDir+"/test.db")

	stdin, err := runCmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}

	if err := runCmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	stdin.Close()

	done := make(chan error, 1)
	go func() {
		done <- runCmd.Wait()
	}()

	select {
	case err := <-done:
		_ = err
	case <-time.After(5 * time.Second):
		runCmd.Process.Kill()
		t.Fatal("process did not exit within 5s after stdin close")
	}
}

func TestVersionDefault(t *testing.T) {
	if version != "dev" {
		t.Errorf("version = %q, want %q", version, "dev")
	}
}

func TestVersionVariableNotEmpty(t *testing.T) {
	if version == "" {
		t.Error("version variable should not be empty")
	}
}

// ========== InitLogger tests ==========

func TestInitLoggerJSON(t *testing.T) {
	cfg := config.LogConfig{Level: "debug", Format: "json"}
	config.InitLogger(cfg)
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

// ========== DefaultConfig test ==========

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
	err := run("postgres", "")
	if err == nil {
		t.Error("expected error for postgres with no DSN")
	}
}

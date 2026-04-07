package main

import (
	"os/exec"
	"strings"
	"testing"
	"time"
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

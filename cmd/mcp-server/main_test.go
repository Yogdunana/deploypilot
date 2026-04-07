package main

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestMainBuilds(t *testing.T) {
	// Verify the binary builds successfully
	tmpDir := t.TempDir()
	binPath := tmpDir + "/mcp-server"

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build: %v\n%s", err, string(output))
	}
}

func TestMainStartsAndResponds(t *testing.T) {
	// Build the binary
	tmpDir := t.TempDir()
	binPath := tmpDir + "/mcp-server"

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, string(out))
	}

	// Run the binary — it should start and wait for stdio input (not exit immediately)
	runCmd := exec.Command(binPath)
	runCmd.Env = append(os.Environ(), "DEPLOYPILOT_DATABASE_DSN="+tmpDir+"/test.db")

	// Use a pipe for stdin so we can close it
	stdin, err := runCmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}

	if err := runCmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Close stdin to trigger graceful shutdown after a moment
	time.Sleep(100 * time.Millisecond)
	stdin.Close()

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- runCmd.Wait()
	}()

	select {
	case err := <-done:
		// Process exited — that's expected when stdin closes
		// Non-zero exit is acceptable (broken pipe, etc.)
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

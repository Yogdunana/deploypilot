package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMainOutput(t *testing.T) {
	// Build and run the binary in a subprocess since main() calls os.Exit
	tmpDir := t.TempDir()
	binPath := tmpDir + "/mcp-server"

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build: %v\n%s", err, string(output))
	}

	runCmd := exec.Command(binPath)
	runCmd.Env = os.Environ()
	out, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	result := string(out)

	if !strings.Contains(result, "mcp-server") {
		t.Errorf("main() output should contain 'mcp-server', got: %q", result)
	}
	if !strings.Contains(result, "placeholder") {
		t.Errorf("main() output should contain 'placeholder', got: %q", result)
	}
	if !strings.Contains(result, "dev") {
		t.Errorf("main() output should contain version 'dev', got: %q", result)
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

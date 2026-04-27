package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// executeCommand runs a cobra command and returns its output.
func executeCommand(args ...string) (string, error) {
	rootCmd.SetArgs(args)

	// Capture both stdout and the cobra output writer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	err := rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var stdoutBuf bytes.Buffer
	io.Copy(&stdoutBuf, r)

	// Combine both outputs
	combined := buf.String() + stdoutBuf.String()
	return combined, err
}

// ========== Root Command ==========

func TestRootCommand(t *testing.T) {
	output, err := executeCommand("--help")
	if err != nil {
		t.Fatalf("root --help error = %v", err)
	}
	if !strings.Contains(output, "deploypilot") {
		t.Errorf("output missing 'deploypilot': %s", output)
	}
	if !strings.Contains(output, "MCP-based") {
		t.Errorf("output missing MCP description: %s", output)
	}
}

func TestRootPersistentFlags(t *testing.T) {
	output, _ := executeCommand("--help")
	if !strings.Contains(output, "--config") {
		t.Error("missing --config flag")
	}
	if !strings.Contains(output, "--format") {
		t.Error("missing --format flag")
	}
}

// ========== version ==========

func TestVersionCommand(t *testing.T) {
	output, err := executeCommand("version")
	if err != nil {
		t.Fatalf("version error = %v", err)
	}
	if !strings.Contains(output, "deploypilot version") {
		t.Errorf("output = %q", output)
	}
}

// ========== serve ==========

func TestServeCommand(t *testing.T) {
	output, err := executeCommand("serve", "--help")
	if err != nil {
		t.Fatalf("serve --help error = %v", err)
	}
	if !strings.Contains(output, "MCP server") {
		t.Errorf("output missing MCP server reference: %s", output)
	}
}

func TestServeFlags(t *testing.T) {
	output, _ := executeCommand("serve", "--help")
	if !strings.Contains(output, "--transport") {
		t.Error("missing --transport flag")
	}
	if !strings.Contains(output, "--port") {
		t.Error("missing --port flag")
	}
}

// ========== app ==========

func TestAppCommand(t *testing.T) {
	output, err := executeCommand("app", "--help")
	if err != nil {
		t.Fatalf("app --help error = %v", err)
	}
	if !strings.Contains(output, "applications") {
		t.Errorf("output = %s", output)
	}
}

func TestAppList(t *testing.T) {
	output, err := executeCommand("app", "list")
	if err != nil {
		t.Fatalf("app list error = %v", err)
	}
	if !strings.Contains(output, "Listing") {
		t.Errorf("output = %q", output)
	}
}

func TestAppCreateFlags(t *testing.T) {
	output, _ := executeCommand("app", "create", "--help")
	if !strings.Contains(output, "--name") {
		t.Error("missing --name flag")
	}
	if !strings.Contains(output, "--repo") {
		t.Error("missing --repo flag")
	}
	if !strings.Contains(output, "--stack") {
		t.Error("missing --stack flag")
	}
}

func TestAppDeployFlags(t *testing.T) {
	output, _ := executeCommand("app", "deploy", "--help")
	if !strings.Contains(output, "--name") {
		t.Error("missing --name flag")
	}
	if !strings.Contains(output, "--image") {
		t.Error("missing --image flag")
	}
	if !strings.Contains(output, "--server") {
		t.Error("missing --server flag")
	}
}

func TestAppDeleteFlags(t *testing.T) {
	output, _ := executeCommand("app", "delete", "--help")
	if !strings.Contains(output, "--name") {
		t.Error("missing --name flag")
	}
	if !strings.Contains(output, "--force") {
		t.Error("missing --force flag")
	}
}

// ========== server ==========

func TestServerCommand(t *testing.T) {
	output, err := executeCommand("server", "--help")
	if err != nil {
		t.Fatalf("server --help error = %v", err)
	}
	if !strings.Contains(output, "servers") {
		t.Errorf("output = %s", output)
	}
}

func TestServerAddFlags(t *testing.T) {
	output, _ := executeCommand("server", "add", "--help")
	if !strings.Contains(output, "--name") {
		t.Error("missing --name flag")
	}
	if !strings.Contains(output, "--host") {
		t.Error("missing --host flag")
	}
	if !strings.Contains(output, "--port") {
		t.Error("missing --port flag")
	}
}

func TestServerTestFlags(t *testing.T) {
	output, _ := executeCommand("server", "test", "--help")
	if !strings.Contains(output, "--name") {
		t.Error("missing --name flag")
	}
}

// ========== backup ==========

func TestBackupCommand(t *testing.T) {
	output, err := executeCommand("backup", "--help")
	if err != nil {
		t.Fatalf("backup --help error = %v", err)
	}
	if !strings.Contains(output, "backup") {
		t.Errorf("output = %s", output)
	}
}

func TestBackupCreateFlags(t *testing.T) {
	output, _ := executeCommand("backup", "create", "--help")
	if !strings.Contains(output, "--name") {
		t.Error("missing --name flag")
	}
	if !strings.Contains(output, "--include-db") {
		t.Error("missing --include-db flag")
	}
}

func TestBackupRestoreFlags(t *testing.T) {
	output, _ := executeCommand("backup", "restore", "--help")
	if !strings.Contains(output, "--name") {
		t.Error("missing --name flag")
	}
	if !strings.Contains(output, "--force") {
		t.Error("missing --force flag")
	}
}

// ========== All Commands Exist ==========

func TestAllCommandsRegistered(t *testing.T) {
	commands := [][]string{
		{"version"},
		{"serve"},
		{"status"}, {"start"}, {"stop"}, {"restart"}, {"reload"},
		{"logs"}, {"config"}, {"config", "show"}, {"config", "get"}, {"config", "set"},
		{"upgrade"}, {"uninstall"},
		{"reset"}, {"reset", "password"}, {"reset", "port"}, {"reset", "secret"},
		{"user-info"},
		{"app"}, {"app", "list"}, {"app", "create"}, {"app", "deploy"}, {"app", "delete"},
		{"server"}, {"server", "list"}, {"server", "add"}, {"server", "test"},
		{"backup"}, {"backup", "create"}, {"backup", "restore"}, {"backup", "list"},
		{"credential"}, {"credential", "add"},
	}

	for _, cmd := range commands {
		name := strings.Join(cmd, " ")
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(append(cmd, "--help")...)
			if err != nil {
				t.Errorf("command %q not registered: %v", name, err)
			}
		})
	}
}

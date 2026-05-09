package service

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
)

// preflightMockExecutor implements CommandExecutor for preflight tests.
// Named differently from mockExecutor in bridge_test.go to avoid collision.
type preflightMockExecutor struct {
	responses map[string]string // cmd -> output
	errors    map[string]error  // cmd -> error
}

func (m *preflightMockExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	if err, ok := m.errors[cmd]; ok {
		return "", err
	}
	if out, ok := m.responses[cmd]; ok {
		return out, nil
	}
	return "", fmt.Errorf("mock: unknown command %q", cmd)
}

func TestRunPreflight_AllPass(t *testing.T) {
	exec := &preflightMockExecutor{
		responses: map[string]string{
			"echo ok":                                        "ok",
			"docker version --format '{{.Server.Version}}'": "24.0.7",
		},
	}
	// Use empty host to skip TCP check (local deployment scenario)
	result := RunPreflight(context.Background(), PreflightConfig{
		Executor:     exec,
		PortMappings: "8080:80",
	})
	if !result.Passed {
		t.Fatalf("expected passed, got: %s", result.Message)
	}
	// Should have Docker + Port checks (no host = no TCP/SSH checks)
	if len(result.Checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(result.Checks))
	}
	for _, c := range result.Checks {
		if !c.Passed {
			t.Errorf("check %s failed: %s", c.Name, c.Message)
		}
	}
}

func TestRunPreflight_TCPFail(t *testing.T) {
	result := RunPreflight(context.Background(), PreflightConfig{
		Host:     "10.255.255.1",
		Port:     9999,
		Executor: &preflightMockExecutor{},
	})
	if result.Passed {
		t.Fatal("expected failure for unreachable host")
	}
	if result.Code != PreflightTCPUnreachable {
		t.Errorf("expected code %s, got %s", PreflightTCPUnreachable, result.Code)
	}
	if result.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestRunPreflight_SSHAuthFail(t *testing.T) {
	// We test SSH auth failure by providing an executor that fails on echo ok
	// but we need TCP to pass. Since we can't easily mock TCP, we test
	// the SSH check directly.
	check := checkSSHAuth(context.Background(), &preflightMockExecutor{
		errors: map[string]error{"echo ok": fmt.Errorf("permission denied")},
	})
	if check.Passed {
		t.Fatal("expected SSH auth failure")
	}
	if check.Suggestion == "" {
		t.Error("expected suggestion for SSH auth failure")
	}
}

func TestRunPreflight_DockerUnavailable(t *testing.T) {
	check := checkDocker(context.Background(), &preflightMockExecutor{
		errors: map[string]error{
			"docker version --format '{{.Server.Version}}'": fmt.Errorf("command not found"),
		},
	})
	if check.Passed {
		t.Fatal("expected Docker unavailability")
	}
	if check.Suggestion == "" {
		t.Error("expected suggestion for Docker unavailability")
	}
}

func TestRunPreflight_PortConflict(t *testing.T) {
	exec := &preflightMockExecutor{
		responses: map[string]string{
			"echo ok":                              "ok",
			"docker version --format '{{.Server.Version}}'": "24.0.7",
			"ss -tlnp 2>/dev/null | grep ':8080 ' || true": "LISTEN  0  128  *:8080  *:*",
		},
	}
	// TCP will fail since 127.0.0.1:22 may not be open in CI
	// So we just test the port conflict check directly
	check := checkPortConflict(context.Background(), exec, "8080:80")
	if check.Passed {
		t.Fatal("expected port conflict")
	}
	if !strings.Contains(check.Message, "8080") {
		t.Errorf("expected port 8080 in message, got: %s", check.Message)
	}
}

func TestRunPreflight_PortConflict_Free(t *testing.T) {
	exec := &preflightMockExecutor{
		responses: map[string]string{
			"ss -tlnp 2>/dev/null | grep ':9090 ' || true": "",
		},
	}
	check := checkPortConflict(context.Background(), exec, "9090:80")
	if !check.Passed {
		t.Fatalf("expected port to be free, got: %s", check.Message)
	}
}

func TestParseHostPorts(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"8080:80", []string{"8080"}},
		{"8080:80,3000:3000", []string{"8080", "3000"}},
		{"8080:80/tcp", []string{"8080"}},
		{"8080:80/tcp,3000:3000/tcp", []string{"8080", "3000"}},
		{"8080", []string{"8080"}},
		{"", nil},
	}
	for _, tt := range tests {
		result := parseHostPorts(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("parseHostPorts(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestPreflightError_Error(t *testing.T) {
	err := &PreflightError{
		Code:    PreflightTCPUnreachable,
		Message: "Cannot connect to 10.0.0.1:22",
		Checks:  []PreflightCheck{{Name: "TCP", Passed: false}},
	}
	msg := err.Error()
	if !strings.Contains(msg, "SSH_TCP_UNREACHABLE") {
		t.Errorf("expected code in error message, got: %s", msg)
	}
}

func TestRunPreflight_LocalOnly(t *testing.T) {
	exec := &preflightMockExecutor{
		responses: map[string]string{
			"docker version --format '{{.Server.Version}}'": "24.0.7",
		},
	}
	result := RunPreflight(context.Background(), PreflightConfig{
		Executor: exec,
	})
	if !result.Passed {
		t.Fatalf("local preflight should pass, got: %s", result.Message)
	}
	// Should only have Docker check (no host = no TCP/SSH checks)
	if len(result.Checks) != 1 {
		t.Fatalf("expected 1 check (Docker only), got %d", len(result.Checks))
	}
}

func TestRunPreflight_SSHAuthUnexpectedOutput(t *testing.T) {
	check := checkSSHAuth(context.Background(), &preflightMockExecutor{
		responses: map[string]string{"echo ok": "some banner\nok"},
	})
	// The trimmed output is "some banner\nok" which != "ok"
	if check.Passed {
		t.Fatal("expected failure for unexpected output")
	}
}

func TestCheckTCP_Success(t *testing.T) {
	// Start a TCP listener on a random port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("cannot bind TCP listener")
	}
	defer ln.Close()

	addr := ln.Addr().String()
	_, portStr, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portStr)

	check := checkTCP(context.Background(), "127.0.0.1", port)
	if !check.Passed {
		t.Fatalf("expected TCP success, got: %s", check.Message)
	}
}

// ========== Additional Coverage Tests ==========

func TestRunPreflight_RemoteWithExecutor(t *testing.T) {
	exec := &preflightMockExecutor{
		responses: map[string]string{
			"echo ok":                                        "ok",
			"docker version --format '{{.Server.Version}}'": "24.0.7",
		},
	}
	// Use a real TCP listener to pass the TCP check
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("cannot bind TCP listener")
	}
	defer ln.Close()

	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	result := RunPreflight(context.Background(), PreflightConfig{
		Host:         "127.0.0.1",
		Port:         port,
		Executor:     exec,
		PortMappings: "9090:80",
	})
	if !result.Passed {
		t.Fatalf("expected passed, got: %s", result.Message)
	}
	// Should have TCP + SSH + Docker + Port checks
	if len(result.Checks) != 4 {
		t.Fatalf("expected 4 checks, got %d", len(result.Checks))
	}
}

func TestRunPreflight_SSHAuthPass(t *testing.T) {
	check := checkSSHAuth(context.Background(), &preflightMockExecutor{
		responses: map[string]string{"echo ok": "ok"},
	})
	if !check.Passed {
		t.Fatalf("expected SSH auth success, got: %s", check.Message)
	}
}

func TestRunPreflight_DockerSuccess(t *testing.T) {
	check := checkDocker(context.Background(), &preflightMockExecutor{
		responses: map[string]string{
			"docker version --format '{{.Server.Version}}'": "24.0.7",
		},
	})
	if !check.Passed {
		t.Fatalf("expected Docker success, got: %s", check.Message)
	}
}

func TestRunPreflight_PortConflict_MultiplePorts(t *testing.T) {
	exec := &preflightMockExecutor{
		responses: map[string]string{
			"ss -tlnp 2>/dev/null | grep ':8080 ' || true": "",
			"ss -tlnp 2>/dev/null | grep ':3000 ' || true": "",
		},
	}
	check := checkPortConflict(context.Background(), exec, "8080:80,3000:3000")
	if !check.Passed {
		t.Fatalf("expected ports to be free, got: %s", check.Message)
	}
}

func TestRunPreflight_PortConflict_NoHostPorts(t *testing.T) {
	exec := &preflightMockExecutor{}
	check := checkPortConflict(context.Background(), exec, "")
	if !check.Passed {
		t.Fatalf("expected pass for empty port mappings, got: %s", check.Message)
	}
}

func TestParseHostPorts_JustPort(t *testing.T) {
	result := parseHostPorts("8080")
	if len(result) != 1 || result[0] != "8080" {
		t.Errorf("expected [8080], got %v", result)
	}
}

func TestParseHostPorts_WithProtocol(t *testing.T) {
	result := parseHostPorts("8080:80/tcp")
	if len(result) != 1 || result[0] != "8080" {
		t.Errorf("expected [8080], got %v", result)
	}
}

func TestCheckTCP_Fail(t *testing.T) {
	check := checkTCP(context.Background(), "10.255.255.1", 1)
	if check.Passed {
		t.Fatal("expected TCP failure for unreachable host")
	}
	if check.Suggestion == "" {
		t.Error("expected suggestion for TCP failure")
	}
}

package detector

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// ========== Mock Command Runner ==========

type mockRunner struct {
	mu       sync.Mutex
	commands map[string]string // "name arg1 arg2" -> output
	errors   map[string]error
	calls    []string
}

func newMockRunner() *mockRunner {
	return &mockRunner{
		commands: make(map[string]string),
		errors:   make(map[string]error),
	}
}

func (m *mockRunner) Run(name string, args ...string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := name + " " + strings.Join(args, " ")
	m.calls = append(m.calls, key)

	if err, ok := m.errors[key]; ok {
		return "", err
	}
	if out, ok := m.commands[key]; ok {
		return out, nil
	}
	return "", fmt.Errorf("mock: unknown command %s", key)
}

// ========== Level 1: OS Detection ==========

func TestDetectOS(t *testing.T) {
	d := New(newMockRunner())
	report := d.Detect(context.Background(), LevelOS, nil, nil)

	if report.OS == nil {
		t.Fatal("OS info should not be nil")
	}
	if report.OS.GOOS == "" {
		t.Error("OS.GOOS should not be empty")
	}
	if report.OS.Arch == "" {
		t.Error("OS.Arch should not be empty")
	}
	if report.Level != LevelOS {
		t.Errorf("Level = %d, want %d", report.Level, LevelOS)
	}
}

// ========== Level 2: Docker Detection ==========

func TestDetectDockerInstalled(t *testing.T) {
	mock := newMockRunner()
	mock.commands["docker --version"] = "Docker version 24.0.7, build afdd53b"
	mock.commands["docker info"] = "Containers: 5\nImages: 10"

	d := New(mock)
	report := d.Detect(context.Background(), LevelDocker, nil, nil)

	if report.Docker == nil {
		t.Fatal("Docker info should not be nil")
	}
	if !report.Docker.Installed {
		t.Error("Docker.Installed should be true")
	}
	if !report.Docker.Running {
		t.Error("Docker.Running should be true")
	}
	if !strings.Contains(report.Docker.Version, "24.0.7") {
		t.Errorf("Docker.Version = %q", report.Docker.Version)
	}
}

func TestDetectDockerNotInstalled(t *testing.T) {
	mock := newMockRunner()
	mock.errors["docker --version"] = fmt.Errorf("command not found")

	d := New(mock)
	report := d.Detect(context.Background(), LevelDocker, nil, nil)

	if report.Docker.Installed {
		t.Error("Docker.Installed should be false")
	}
	if report.Docker.Running {
		t.Error("Docker.Running should be false")
	}
}

func TestDetectDockerInstalledButNotRunning(t *testing.T) {
	mock := newMockRunner()
	mock.commands["docker --version"] = "Docker version 24.0.7"
	mock.errors["docker info"] = fmt.Errorf("Cannot connect to Docker daemon")

	d := New(mock)
	report := d.Detect(context.Background(), LevelDocker, nil, nil)

	if !report.Docker.Installed {
		t.Error("Docker.Installed should be true")
	}
	if report.Docker.Running {
		t.Error("Docker.Running should be false (daemon not running)")
	}
}

// ========== Level 3: Port Detection ==========

func TestDetectPortOpen(t *testing.T) {
	// Start a real TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	d := New(newMockRunner())
	report := d.Detect(context.Background(), LevelPort, []int{port}, nil)

	info, ok := report.Ports[port]
	if !ok {
		t.Fatalf("Port %d not in report", port)
	}
	if !info.Open {
		t.Error("Port should be open")
	}
	if info.Protocol != "tcp" {
		t.Errorf("Protocol = %q, want tcp", info.Protocol)
	}
}

func TestDetectPortClosed(t *testing.T) {
	d := New(newMockRunner())
	report := d.Detect(context.Background(), LevelPort, []int{19999}, nil)

	info := report.Ports[19999]
	if info.Open {
		t.Error("Port 19999 should be closed")
	}
}

func TestDetectMultiplePorts(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()
	openPort := listener.Addr().(*net.TCPAddr).Port

	d := New(newMockRunner())
	report := d.Detect(context.Background(), LevelPort, []int{openPort, 19998}, nil)

	if len(report.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(report.Ports))
	}
	if !report.Ports[openPort].Open {
		t.Errorf("Port %d should be open", openPort)
	}
	if report.Ports[19998].Open {
		t.Error("Port 19998 should be closed")
	}
}

// ========== Level 4: Service Detection ==========

func TestDetectServiceTCPHealthy(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	d := New(newMockRunner())
	report := d.Detect(context.Background(), LevelService, nil,
		[]string{fmt.Sprintf("tcp://127.0.0.1:%d", port)})

	svc := report.Services[fmt.Sprintf("tcp://127.0.0.1:%d", port)]
	if svc == nil {
		t.Fatal("Service should be in report")
	}
	if !svc.Healthy {
		t.Error("Service should be healthy")
	}
	if svc.Latency == "" {
		t.Error("Latency should not be empty")
	}
}

func TestDetectServiceTCPUnhealthy(t *testing.T) {
	d := New(newMockRunner())
	report := d.Detect(context.Background(), LevelService, nil,
		[]string{"tcp://127.0.0.1:19997"})

	svc := report.Services["tcp://127.0.0.1:19997"]
	if svc.Healthy {
		t.Error("Service should be unhealthy")
	}
	if svc.Error == "" {
		t.Error("Error should not be empty for unhealthy service")
	}
}

func TestDetectMultipleServices(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	d := New(newMockRunner())
	report := d.Detect(context.Background(), LevelService, nil,
		[]string{
			fmt.Sprintf("tcp://127.0.0.1:%d", port),
			"tcp://127.0.0.1:19996",
		})

	if len(report.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(report.Services))
	}
}

// ========== Full Detection ==========

func TestFullDetection(t *testing.T) {
	mock := newMockRunner()
	mock.commands["docker --version"] = "Docker version 25.0.0"
	mock.commands["docker info"] = "ok"

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	d := New(mock)
	report := d.Detect(context.Background(), LevelService,
		[]int{port},
		[]string{fmt.Sprintf("tcp://127.0.0.1:%d", port)})

	if report.Level != LevelService {
		t.Errorf("Level = %d, want %d", report.Level, LevelService)
	}
	if report.OS == nil {
		t.Error("OS should be detected")
	}
	if report.Docker == nil {
		t.Error("Docker should be detected")
	}
	if len(report.Ports) != 1 {
		t.Errorf("Expected 1 port, got %d", len(report.Ports))
	}
	if len(report.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(report.Services))
	}
}

// ========== Summary ==========

func TestSummary(t *testing.T) {
	mock := newMockRunner()
	mock.commands["docker --version"] = "Docker version 24.0.0"
	mock.commands["docker info"] = "ok"

	d := New(mock)
	report := d.Detect(context.Background(), LevelDocker, nil, nil)

	summary := report.Summary()
	if !strings.Contains(summary, "OS:") {
		t.Error("Summary should contain OS info")
	}
	if !strings.Contains(summary, "Docker:") {
		t.Error("Summary should contain Docker info")
	}
	if !strings.Contains(summary, "Level 2") {
		t.Error("Summary should contain level")
	}
}

// ========== Context Cancellation ==========

func TestDetectWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	d := New(newMockRunner())
	report := d.Detect(ctx, LevelService, nil, []string{"tcp://127.0.0.1:19995"})

	// Should still complete (port/service checks may fail but shouldn't hang)
	if report == nil {
		t.Error("Report should not be nil even with cancelled context")
	}
}

// Suppress unused import
var _ = time.Now

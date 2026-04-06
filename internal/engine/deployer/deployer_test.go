package deployer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockExecutor implements CommandExecutor for testing.
type mockExecutor struct {
	mu     sync.Mutex
	responses map[string]string // cmd -> output
	errors    map[string]error  // cmd -> error
	calls     []string          // recorded calls in order
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *mockExecutor) RunCommand(_ context.Context, cmd string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, cmd)

	// Check for prefix matches
	for pattern, output := range m.responses {
		if strings.Contains(cmd, pattern) {
			if err, ok := m.errors[pattern]; ok {
				return output, err
			}
			return output, nil
		}
	}

	return "", fmt.Errorf("mock: no response configured for command: %s", cmd)
}

func (m *mockExecutor) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *mockExecutor) getCall(n int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n < len(m.calls) {
		return m.calls[n]
	}
	return ""
}

// ========== Deploy Tests ==========

func TestDeploySuccess(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker pull"] = "nginx:latest: Pull complete"
	mock.responses["docker rm"] = ""
	mock.responses["docker run"] = "abc123def456"
	mock.responses["docker inspect"] = "abc123def456|/my-app|nginx:latest|running|2026-04-06T12:00:00Z"

	d := New(mock)
	cfg := DeployConfig{
		Image:         "nginx:latest",
		ContainerName: "my-app",
		Ports:         "8080:80",
	}

	status, err := d.Deploy(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	if status.Name != "my-app" {
		t.Errorf("status.Name = %q, want %q", status.Name, "my-app")
	}
	if status.Image != "nginx:latest" {
		t.Errorf("status.Image = %q, want %q", status.Image, "nginx:latest")
	}
	if status.Status != "running" {
		t.Errorf("status.Status = %q, want %q", status.Status, "running")
	}
}

func TestDeployMissingImage(t *testing.T) {
	mock := newMockExecutor()
	d := New(mock)

	_, err := d.Deploy(context.Background(), DeployConfig{
		ContainerName: "my-app",
	})
	if err == nil {
		t.Error("Deploy() should fail when image is empty")
	}
}

func TestDeployMissingContainerName(t *testing.T) {
	mock := newMockExecutor()
	d := New(mock)

	_, err := d.Deploy(context.Background(), DeployConfig{
		Image: "nginx:latest",
	})
	if err == nil {
		t.Error("Deploy() should fail when container_name is empty")
	}
}

func TestDeployPullFailure(t *testing.T) {
	mock := newMockExecutor()
	mock.errors["docker pull"] = fmt.Errorf("permission denied")

	d := New(mock)
	_, err := d.Deploy(context.Background(), DeployConfig{
		Image:         "private/image",
		ContainerName: "my-app",
	})
	if err == nil {
		t.Error("Deploy() should fail when pull fails")
	}
	if !strings.Contains(err.Error(), "failed to pull image") {
		t.Errorf("error = %q, want contain 'failed to pull image'", err.Error())
	}
}

func TestDeployRunFailure(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker pull"] = "ok"
	mock.responses["docker rm"] = ""
	mock.errors["docker run"] = fmt.Errorf("port already in use")

	d := New(mock)
	_, err := d.Deploy(context.Background(), DeployConfig{
		Image:         "nginx:latest",
		ContainerName: "my-app",
	})
	if err == nil {
		t.Error("Deploy() should fail when run fails")
	}
}

func TestDeployWithEnvVars(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker pull"] = "ok"
	mock.responses["docker rm"] = ""
	mock.responses["docker run"] = "container-id"
	mock.responses["docker inspect"] = "container-id|/my-app|nginx:latest|running|2026-04-06T12:00:00Z"

	d := New(mock)
	cfg := DeployConfig{
		Image:         "nginx:latest",
		ContainerName: "my-app",
		EnvVars:       map[string]string{"DB_HOST": "localhost", "DB_PORT": "5432"},
	}

	_, err := d.Deploy(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	// Verify env vars are in the run command
	runCmd := mock.getCall(2) // 0=pull, 1=rm, 2=run
	if !strings.Contains(runCmd, "-e DB_HOST=localhost") {
		t.Errorf("run command missing env var, got: %s", runCmd)
	}
	if !strings.Contains(runCmd, "-e DB_PORT=5432") {
		t.Errorf("run command missing env var, got: %s", runCmd)
	}
}

func TestDeployWithLabels(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker pull"] = "ok"
	mock.responses["docker rm"] = ""
	mock.responses["docker run"] = "container-id"
	mock.responses["docker inspect"] = "container-id|/my-app|nginx:latest|running|2026-04-06T12:00:00Z"

	d := New(mock)
	cfg := DeployConfig{
		Image:         "nginx:latest",
		ContainerName: "my-app",
		Labels:        map[string]string{"app": "myapp", "version": "1.0"},
	}

	_, err := d.Deploy(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	runCmd := mock.getCall(2)
	if !strings.Contains(runCmd, "-l app=myapp") {
		t.Errorf("run command missing label, got: %s", runCmd)
	}
}

func TestDeployWithResourceLimits(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker pull"] = "ok"
	mock.responses["docker rm"] = ""
	mock.responses["docker run"] = "container-id"
	mock.responses["docker inspect"] = "container-id|/my-app|nginx:latest|running|2026-04-06T12:00:00Z"

	d := New(mock)
	cfg := DeployConfig{
		Image:         "nginx:latest",
		ContainerName: "my-app",
		ResourceLimits: ResourceLimits{CPU: "2", Memory: "4GB"},
	}

	_, err := d.Deploy(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	runCmd := mock.getCall(2)
	if !strings.Contains(runCmd, "--cpus 2") {
		t.Errorf("run command missing CPU limit, got: %s", runCmd)
	}
	if !strings.Contains(runCmd, "--memory 4GB") {
		t.Errorf("run command missing memory limit, got: %s", runCmd)
	}
}

// ========== GetContainerStatus Tests ==========

func TestGetContainerStatus(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker inspect"] = "abc123|/my-app|nginx:latest|running|2026-04-06T12:00:00Z"

	d := New(mock)
	status, err := d.GetContainerStatus(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("GetContainerStatus() error = %v", err)
	}

	if status.ID != "abc123" {
		t.Errorf("ID = %q, want %q", status.ID, "abc123")
	}
	if status.Status != "running" {
		t.Errorf("Status = %q, want %q", status.Status, "running")
	}
}

func TestGetContainerStatusNotFound(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker inspect"] = ""

	d := New(mock)
	_, err := d.GetContainerStatus(context.Background(), "nonexistent")
	if err == nil {
		t.Error("GetContainerStatus() should fail for nonexistent container")
	}
}

// ========== Stop/Remove Tests ==========

func TestStop(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker stop"] = "my-app"

	d := New(mock)
	err := d.Stop(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if mock.callCount() != 1 {
		t.Errorf("expected 1 call, got %d", mock.callCount())
	}
}

func TestRemove(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker rm"] = "my-app"

	d := New(mock)
	err := d.Remove(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
}

// ========== GetContainerLogs Tests ==========

func TestGetContainerLogs(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker logs"] = "2026-04-06 12:00:00 Server started\n2026-04-06 12:00:01 Ready"

	d := New(mock)
	logs, err := d.GetContainerLogs(context.Background(), "my-app", 100)
	if err != nil {
		t.Fatalf("GetContainerLogs() error = %v", err)
	}
	if !strings.Contains(logs, "Server started") {
		t.Errorf("logs missing expected content, got: %s", logs)
	}
}

// ========== HealthCheck Tests ==========

func TestHealthCheckNone(t *testing.T) {
	mock := newMockExecutor()
	d := New(mock)

	err := d.HealthCheck(context.Background(), HealthCheckConfig{Type: "none"})
	if err != nil {
		t.Errorf("HealthCheck(none) should pass, got: %v", err)
	}
}

func TestHealthCheckHTTPSuccess(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["curl"] = "ok"

	d := New(mock)
	err := d.HealthCheck(context.Background(), HealthCheckConfig{
		Type:    "http",
		Target:  "http://localhost:8080/health",
		Timeout: 3 * time.Second,
		Retries: 1,
	})
	if err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}
}

func TestHealthCheckHTTPFailure(t *testing.T) {
	mock := newMockExecutor()
	mock.errors["curl"] = fmt.Errorf("connection refused")

	d := New(mock)
	err := d.HealthCheck(context.Background(), HealthCheckConfig{
		Type:    "http",
		Target:  "http://localhost:8080/health",
		Timeout: 1 * time.Second,
		Retries: 2,
		Interval: 10 * time.Millisecond,
	})
	if err == nil {
		t.Error("HealthCheck() should fail after retries")
	}
}

// ========== BuildRunCommand Tests ==========

func TestBuildRunCommandBasic(t *testing.T) {
	mock := newMockExecutor()
	d := New(mock)

	cfg := DeployConfig{
		Image:         "nginx:latest",
		ContainerName: "my-app",
		Ports:         "8080:80",
	}

	cmd := d.buildRunCommand(cfg)

	expected := []string{"docker run -d", "--name my-app", "--restart unless-stopped", "-p 8080:80", "nginx:latest"}
	for _, exp := range expected {
		if !strings.Contains(cmd, exp) {
			t.Errorf("buildRunCommand() missing %q, got: %s", exp, cmd)
		}
	}
}

func TestBuildRunCommandWithNetwork(t *testing.T) {
	mock := newMockExecutor()
	d := New(mock)

	cfg := DeployConfig{
		Image:         "nginx:latest",
		ContainerName: "my-app",
		Network:       "my-network",
	}

	cmd := d.buildRunCommand(cfg)
	if !strings.Contains(cmd, "--network my-network") {
		t.Errorf("buildRunCommand() missing network, got: %s", cmd)
	}
}

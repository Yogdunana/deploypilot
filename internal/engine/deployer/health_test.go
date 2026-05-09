package deployer

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ========== HTTP Health Check ==========

func TestCheckHTTPSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hc := NewHealthChecker(&noopExecutor{})
	result := hc.CheckHTTP(context.Background(), server.URL, 3, 100*time.Millisecond)

	if !result.Healthy {
		t.Errorf("expected healthy, got error: %s", result.Error)
	}
	if result.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", result.Attempts)
	}
}

func TestCheckHTTPFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	hc := NewHealthChecker(&noopExecutor{})
	result := hc.CheckHTTP(context.Background(), server.URL, 2, 50*time.Millisecond)

	if result.Healthy {
		t.Error("expected unhealthy for 503 response")
	}
}

func TestCheckHTTPConnectionRefused(t *testing.T) {
	hc := NewHealthChecker(&noopExecutor{})
	result := hc.CheckHTTP(context.Background(), "http://127.0.0.1:1", 2, 50*time.Millisecond)

	if result.Healthy {
		t.Error("expected unhealthy for connection refused")
	}
}

func TestCheckHTTPRetryThenSuccess(t *testing.T) {
	var attempts int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		a := attempts
		mu.Unlock()

		if a < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	hc := NewHealthChecker(&noopExecutor{})
	result := hc.CheckHTTP(context.Background(), server.URL, 5, 50*time.Millisecond)

	if !result.Healthy {
		t.Errorf("expected healthy after retries, error: %s", result.Error)
	}
	if result.Attempts < 3 {
		t.Errorf("Attempts = %d, want >= 3", result.Attempts)
	}
}

func TestCheckHTTPContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	hc := NewHealthChecker(&noopExecutor{})
	result := hc.CheckHTTP(ctx, "http://127.0.0.1:1", 5, 1*time.Second)

	if result.Healthy {
		t.Error("expected unhealthy when context cancelled")
	}
}

// ========== TCP Health Check ==========

func TestCheckTCPSuccess(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	hc := NewHealthChecker(&noopExecutor{})
	result := hc.CheckTCP(context.Background(), "127.0.0.1", port, 3, 100*time.Millisecond)

	if !result.Healthy {
		t.Errorf("expected healthy, error: %s", result.Error)
	}
}

func TestCheckTCPFail(t *testing.T) {
	hc := NewHealthChecker(&noopExecutor{})
	result := hc.CheckTCP(context.Background(), "127.0.0.1", 19998, 2, 50*time.Millisecond)

	if result.Healthy {
		t.Error("expected unhealthy for connection refused")
	}
}

// ========== Rollback ==========

func TestRollbackSuccess(t *testing.T) {
	mock := &trackingExecutor{
		mockExecutor: func() *mockExecutor {
			m := newMockExecutor()
			m.responses["docker inspect"] = "old-img|/my-app|old-img:latest|running|2026-04-06T12:00:00Z"
			m.responses["docker stop"] = ""
			m.responses["docker rm"] = ""
			m.responses["docker pull"] = "ok"
			m.responses["docker run"] = "container-id"
			return m
		}(),
	}

	hc := NewHealthChecker(mock)
	result := hc.Rollback(context.Background(), "my-app", "old-img:latest")

	if !result.Success {
		t.Errorf("expected success, message: %s", result.Message)
	}
	if !result.RolledBack {
		t.Error("expected RolledBack = true")
	}
	if result.OldImage != "old-img:latest" {
		t.Errorf("OldImage = %q", result.OldImage)
	}
}

func TestRollbackNoPreviousImage(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker inspect"] = ""

	hc := NewHealthChecker(mock)
	result := hc.Rollback(context.Background(), "my-app", "")

	if result.RolledBack {
		t.Error("should not rollback with no previous image")
	}
}

// ========== DeployWithHealthCheck ==========

func TestDeployWithHealthCheckHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mock := newMockExecutor()
	mock.responses["docker inspect"] = ""
	mock.responses["docker pull"] = "ok"
	mock.responses["docker rm"] = ""
	mock.responses["docker run"] = "abc123"
	mock.responses["docker stop"] = ""

	hc := NewHealthChecker(mock)
	status, healthResult, rollbackResult := hc.DeployWithHealthCheck(
		context.Background(),
		DeployConfig{Image: "nginx:latest", ContainerName: "my-app"},
		server.URL,
		"http",
	)

	if status == nil {
		t.Error("status should not be nil")
	}
	if !healthResult.Healthy {
		t.Errorf("expected healthy, error: %s", healthResult.Error)
	}
	if rollbackResult != nil {
		t.Error("rollbackResult should be nil when healthy")
	}
}

func TestDeployWithHealthCheckUnhealthyTriggersRollback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	mock := &trackingExecutor{
		mockExecutor: func() *mockExecutor {
			m := newMockExecutor()
			m.responses["docker inspect"] = "old-img|/my-app|old-img:latest|running|2026-04-06T12:00:00Z"
			m.responses["docker pull"] = "ok"
			m.responses["docker rm"] = ""
			m.responses["docker run"] = "abc123"
			m.responses["docker stop"] = ""
			return m
		}(),
	}

	hc := NewHealthChecker(mock)
	_, healthResult, rollbackResult := hc.DeployWithHealthCheck(
		context.Background(),
		DeployConfig{Image: "nginx:latest", ContainerName: "my-app"},
		server.URL,
		"http",
	)

	if healthResult.Healthy {
		t.Error("expected unhealthy")
	}
	if rollbackResult == nil {
		t.Fatal("rollbackResult should not be nil when unhealthy")
	}
	if !rollbackResult.RolledBack {
		t.Error("expected rollback to be performed")
	}
}

// ========== Helpers ==========

type noopExecutor struct{}

func (n *noopExecutor) RunCommand(_ context.Context, _ string) (string, error) { return "", nil }

type trackingExecutor struct {
	*mockExecutor
	onCall func(cmd string)
}

func (t *trackingExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	if t.onCall != nil {
		t.onCall(cmd)
	}
	return t.mockExecutor.RunCommand(ctx, cmd)
}

// Suppress unused import warnings
var _ = strings.TrimSpace
var _ = fmt.Sprintf

// ========== Additional Coverage Tests ==========

func TestDeployWithHealthCheck_DeployFail(t *testing.T) {
	mock := newMockExecutor()
	mock.errors["docker pull"] = fmt.Errorf("image not found")

	hc := NewHealthChecker(mock)
	status, healthResult, rollbackResult := hc.DeployWithHealthCheck(
		context.Background(),
		DeployConfig{Image: "nonexistent:v1", ContainerName: "fail-app"},
		"http://localhost:8080",
		"http",
	)

	if status != nil {
		t.Error("status should be nil when deploy fails")
	}
	if healthResult == nil {
		t.Fatal("healthResult should not be nil")
	}
	if healthResult.Healthy {
		t.Error("expected unhealthy when deploy fails")
	}
	if rollbackResult != nil {
		t.Error("rollbackResult should be nil when deploy fails")
	}
}

func TestDeployWithHealthCheck_TCPType(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker inspect"] = ""
	mock.responses["docker pull"] = "ok"
	mock.responses["docker rm"] = ""
	mock.responses["docker run"] = "abc123"
	mock.responses["docker stop"] = ""

	hc := NewHealthChecker(mock)
	status, healthResult, rollbackResult := hc.DeployWithHealthCheck(
		context.Background(),
		DeployConfig{Image: "nginx:latest", ContainerName: "tcp-app"},
		"localhost:8080",
		"tcp",
	)

	if status == nil {
		t.Error("status should not be nil")
	}
	if healthResult == nil {
		t.Fatal("healthResult should not be nil")
	}
	// TCP check to non-existent port should fail
	if healthResult.Healthy {
		t.Error("expected unhealthy for TCP check to non-existent port")
	}
	if rollbackResult == nil {
		t.Fatal("rollbackResult should not be nil when unhealthy")
	}
}

func TestDeployWithHealthCheck_DefaultType(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker inspect"] = ""
	mock.responses["docker pull"] = "ok"
	mock.responses["docker rm"] = ""
	mock.responses["docker run"] = "abc123"

	hc := NewHealthChecker(mock)
	status, healthResult, rollbackResult := hc.DeployWithHealthCheck(
		context.Background(),
		DeployConfig{Image: "nginx:latest", ContainerName: "default-app"},
		"http://localhost:8080",
		"unknown_type",
	)

	if status == nil {
		t.Error("status should not be nil")
	}
	if healthResult == nil {
		t.Fatal("healthResult should not be nil")
	}
	if !healthResult.Healthy {
		t.Error("expected healthy for default/unknown health type")
	}
	if rollbackResult != nil {
		t.Error("rollbackResult should be nil when healthy")
	}
}

func TestDeployWithHealthCheck_ExistingContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mock := newMockExecutor()
	mock.responses["docker inspect"] = "old-id|/my-app|old-img:latest|running|2026-04-06T12:00:00Z"
	mock.responses["docker pull"] = "ok"
	mock.responses["docker rm"] = ""
	mock.responses["docker run"] = "new-id"
	mock.responses["docker stop"] = ""

	hc := NewHealthChecker(mock)
	status, healthResult, rollbackResult := hc.DeployWithHealthCheck(
		context.Background(),
		DeployConfig{Image: "new-img:latest", ContainerName: "my-app"},
		server.URL,
		"http",
	)

	if status == nil {
		t.Error("status should not be nil")
	}
	if !healthResult.Healthy {
		t.Errorf("expected healthy, error: %s", healthResult.Error)
	}
	if rollbackResult != nil {
		t.Error("rollbackResult should be nil when healthy")
	}
}

func TestRollback_DeployFail(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker inspect"] = "old-img|/my-app|old-img:latest|running|2026-04-06T12:00:00Z"
	mock.responses["docker stop"] = ""
	mock.responses["docker rm"] = ""
	mock.errors["docker pull"] = fmt.Errorf("image not found")

	hc := NewHealthChecker(mock)
	result := hc.Rollback(context.Background(), "my-app", "old-img:latest")

	if result.RolledBack {
		t.Error("should not be rolled back when rollback deploy fails")
	}
	if result.Success {
		t.Error("should not be success when rollback deploy fails")
	}
}

func TestCheckHTTP_InvalidURL(t *testing.T) {
	hc := NewHealthChecker(&noopExecutor{})
	result := hc.CheckHTTP(context.Background(), "://invalid-url", 1, 50*time.Millisecond)

	if result.Healthy {
		t.Error("expected unhealthy for invalid URL")
	}
}

func TestDoHTTPCheck_InvalidURL(t *testing.T) {
	hc := NewHealthChecker(&noopExecutor{})
	result := hc.doHTTPCheck(context.Background(), "://invalid")
	if result {
		t.Error("expected false for invalid URL")
	}
}

func TestCheckTCP_InvalidPort(t *testing.T) {
	hc := NewHealthChecker(&noopExecutor{})
	result := hc.CheckTCP(context.Background(), "127.0.0.1", -1, 1, 50*time.Millisecond)

	if result.Healthy {
		t.Error("expected unhealthy for invalid port")
	}
}

func TestGetCurrentImage_Error(t *testing.T) {
	mock := newMockExecutor()
	mock.errors["docker inspect"] = fmt.Errorf("not found")

	hc := NewHealthChecker(mock)
	img, err := hc.getCurrentImage(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent container")
	}
	if img != "" {
		t.Errorf("expected empty image, got %s", img)
	}
}

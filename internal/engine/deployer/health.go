package deployer

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HealthResult represents the result of a health check.
type HealthResult struct {
	Healthy   bool          `json:"healthy"`
	Attempts  int           `json:"attempts"`
	Latency   time.Duration `json:"latency"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// RollbackResult represents the result of a rollback operation.
type RollbackResult struct {
	Success    bool   `json:"success"`
	OldImage   string `json:"old_image"`
	NewImage   string `json:"new_image"`
	Message    string `json:"message"`
	RolledBack bool   `json:"rolled_back"`
}

// HealthChecker performs HTTP/TCP health checks with retry and rollback.
type HealthChecker struct {
	executor     CommandExecutor
	httpClient   *http.Client
	rollbackLock sync.Mutex
}

// NewHealthChecker creates a new HealthChecker.
func NewHealthChecker(executor CommandExecutor) *HealthChecker {
	return &HealthChecker{
		executor: executor,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckHTTP performs an HTTP health check with retries.
func (h *HealthChecker) CheckHTTP(ctx context.Context, target string, retries int, interval time.Duration) *HealthResult {
	var lastErr string
	for i := 0; i < retries; i++ {
		select {
		case <-ctx.Done():
			return &HealthResult{Healthy: false, Attempts: i + 1, Error: ctx.Err().Error(), Timestamp: time.Now()}
		default:
		}

		start := time.Now()
		result := h.doHTTPCheck(ctx, target)
		latency := time.Since(start)

		if result {
			return &HealthResult{
				Healthy:   true,
				Attempts:  i + 1,
				Latency:   latency,
				Timestamp: time.Now(),
			}
		}

		lastErr = fmt.Sprintf("attempt %d failed", i+1)
		if i < retries-1 {
			time.Sleep(interval)
		}
	}

	return &HealthResult{
		Healthy:   false,
		Attempts:  retries,
		Error:     lastErr,
		Timestamp: time.Now(),
	}
}

func (h *HealthChecker) doHTTPCheck(ctx context.Context, target string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return false
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// CheckTCP performs a TCP health check with retries.
func (h *HealthChecker) CheckTCP(ctx context.Context, host string, port int, retries int, interval time.Duration) *HealthResult {
	addr := fmt.Sprintf("%s:%d", host, port)
	var lastErr string

	for i := 0; i < retries; i++ {
		select {
		case <-ctx.Done():
			return &HealthResult{Healthy: false, Attempts: i + 1, Error: ctx.Err().Error(), Timestamp: time.Now()}
		default:
		}

		start := time.Now()
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		latency := time.Since(start)

		if err == nil {
			conn.Close()
			return &HealthResult{
				Healthy:   true,
				Attempts:  i + 1,
				Latency:   latency,
				Timestamp: time.Now(),
			}
		}

		lastErr = fmt.Sprintf("attempt %d: %v", i+1, err)
		if i < retries-1 {
			time.Sleep(interval)
		}
	}

	return &HealthResult{
		Healthy:   false,
		Attempts:  retries,
		Error:     lastErr,
		Timestamp: time.Now(),
	}
}

// DeployWithHealthCheck deploys a container and performs health checks, rolling back on failure.
func (h *HealthChecker) DeployWithHealthCheck(ctx context.Context, cfg DeployConfig, healthTarget string, healthType string) (*ContainerStatus, *HealthResult, *RollbackResult) {
	// Step 1: Get current image (for rollback)
	var currentImage string
	existingStatus, err := h.getCurrentImage(ctx, cfg.ContainerName)
	if err == nil && existingStatus != "" {
		currentImage = existingStatus
	}

	// Step 2: Deploy new container
	deployer := New(h.executor)
	status, err := deployer.Deploy(ctx, cfg)
	if err != nil {
		return nil, &HealthResult{Healthy: false, Error: fmt.Sprintf("deploy failed: %v", err), Timestamp: time.Now()}, nil
	}

	// Step 3: Wait a moment before health check
	time.Sleep(2 * time.Second)

	// Step 4: Health check
	var healthResult *HealthResult
	switch strings.ToLower(healthType) {
	case "http":
		healthResult = h.CheckHTTP(ctx, healthTarget, 3, 3*time.Second)
	case "tcp":
		parts := strings.Split(healthTarget, ":")
		host := parts[0]
		port := 80
		if len(parts) > 1 {
			fmt.Sscanf(parts[1], "%d", &port)
		}
		healthResult = h.CheckTCP(ctx, host, port, 3, 3*time.Second)
	default:
		healthResult = &HealthResult{Healthy: true, Attempts: 0, Timestamp: time.Now()}
	}

	// Step 5: Rollback if unhealthy
	if !healthResult.Healthy {
		rollbackResult := h.Rollback(ctx, cfg.ContainerName, currentImage)
		return status, healthResult, rollbackResult
	}

	return status, healthResult, nil
}

// Rollback stops the current container and redeploys with the previous image.
func (h *HealthChecker) Rollback(ctx context.Context, containerName, previousImage string) *RollbackResult {
	h.rollbackLock.Lock()
	defer h.rollbackLock.Unlock()

	if previousImage == "" {
		// No previous image, just stop
		deployer := New(h.executor)
		_ = deployer.Stop(ctx, containerName)
		return &RollbackResult{
			Success:    false,
			Message:    "no previous image to rollback to, container stopped",
			RolledBack: false,
		}
	}

	// Stop current container
	deployer := New(h.executor)
	_ = deployer.Stop(ctx, containerName)
	_ = deployer.Remove(ctx, containerName)

	// Redeploy with previous image
	rollbackCfg := DeployConfig{
		Image:         previousImage,
		ContainerName: containerName,
	}

	_, err := deployer.Deploy(ctx, rollbackCfg)
	if err != nil {
		return &RollbackResult{
			Success:    false,
			OldImage:   previousImage,
			Message:    fmt.Sprintf("rollback deploy failed: %v", err),
			RolledBack: false,
		}
	}

	return &RollbackResult{
		Success:    true,
		OldImage:   previousImage,
		Message:    fmt.Sprintf("successfully rolled back %s to %s", containerName, previousImage),
		RolledBack: true,
	}
}

func (h *HealthChecker) getCurrentImage(ctx context.Context, containerName string) (string, error) {
	deployer := New(h.executor)
	status, err := deployer.GetContainerStatus(ctx, containerName)
	if err != nil {
		return "", err
	}
	return status.Image, nil
}

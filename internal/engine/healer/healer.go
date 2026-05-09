package healer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
	"github.com/Yogdunana/deploypilot/internal/util"
)

// HealingConfig configures the self-healing behavior.
type HealingConfig struct {
	AutoRestart         bool          `json:"auto_restart"`
	AutoRollback        bool          `json:"auto_rollback"`
	MaxRestarts         int           `json:"max_restarts"`
	RestartWindow       time.Duration `json:"restart_window"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
}

// DefaultHealingConfig returns sensible defaults for healing configuration.
func DefaultHealingConfig() HealingConfig {
	return HealingConfig{
		AutoRestart:         true,
		AutoRollback:        true,
		MaxRestarts:         3,
		RestartWindow:       5 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
	}
}

// HealingResult records what action was taken during a healing cycle.
type HealingResult struct {
	AppID         string    `json:"app_id"`
	AppName       string    `json:"app_name"`
	Action        string    `json:"action"` // none, restarted, rolled_back, notified
	Reason        string    `json:"reason"`
	PreviousState string    `json:"previous_state"`
	NewState      string    `json:"new_state"`
	Timestamp     time.Time `json:"timestamp"`
}

// restartTracker tracks restart attempts per container within a time window.
type restartTracker struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func newRestartTracker() *restartTracker {
	return &restartTracker{
		attempts: make(map[string][]time.Time),
	}
}

// RecordRestart records a restart attempt for a container.
func (rt *restartTracker) RecordRestart(containerName string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	now := time.Now()
	rt.attempts[containerName] = append(rt.attempts[containerName], now)
}

// CountRecentRestarts returns the number of restarts within the given window.
func (rt *restartTracker) CountRecentRestarts(containerName string, window time.Duration) int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	cutoff := time.Now().Add(-window)
	var count int
	var remaining []time.Time
	for _, t := range rt.attempts[containerName] {
		if t.After(cutoff) {
			count++
			remaining = append(remaining, t)
		}
	}
	rt.attempts[containerName] = remaining
	return count
}

// Reset clears restart history for a container.
func (rt *restartTracker) Reset(containerName string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.attempts, containerName)
}

// Healer performs self-healing actions on containers.
type Healer struct {
	executor deployer.CommandExecutor
	config   HealingConfig
	tracker  *restartTracker
}

// NewHealer creates a new Healer with the given executor and configuration.
func NewHealer(executor deployer.CommandExecutor, config HealingConfig) *Healer {
	if config.MaxRestarts <= 0 {
		config.MaxRestarts = 3
	}
	if config.RestartWindow <= 0 {
		config.RestartWindow = 5 * time.Minute
	}
	if config.HealthCheckInterval <= 0 {
		config.HealthCheckInterval = 30 * time.Second
	}
	return &Healer{
		executor: executor,
		config:   config,
		tracker:  newRestartTracker(),
	}
}

// CheckAndHeal inspects a container and takes healing action if needed.
func (h *Healer) CheckAndHeal(ctx context.Context, containerName string) (*HealingResult, error) {
	d := deployer.New(h.executor)
	detail, err := d.GetContainerDetail(ctx, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get container detail: %w", err)
	}

	result := &HealingResult{
		AppName:       containerName,
		PreviousState: detail.Status,
		Timestamp:     time.Now(),
	}

	// If running and healthy, no action needed
	if detail.Status == "running" && (detail.Health == "healthy" || detail.Health == "none") && !detail.OOMKilled {
		result.Action = "none"
		result.NewState = detail.Status
		result.Reason = "container is running and healthy"
		return result, nil
	}

	// If running but unhealthy
	if detail.Status == "running" && detail.Health == "unhealthy" {
		if h.config.AutoRestart {
			recentRestarts := h.tracker.CountRecentRestarts(containerName, h.config.RestartWindow)
			if h.needsRollback(containerName, recentRestarts) {
				result.Action = "rolled_back"
				result.Reason = fmt.Sprintf("container unhealthy, restart count %d exceeded max %d", recentRestarts, h.config.MaxRestarts)
				result.NewState = "rolled_back"
				return result, nil
			}
			if err := h.RestartContainer(ctx, containerName); err != nil {
				result.Action = "notified"
				result.Reason = fmt.Sprintf("container unhealthy, restart failed: %v", err)
				result.NewState = detail.Status
				return result, nil
			}
			h.tracker.RecordRestart(containerName)
			result.Action = "restarted"
			result.Reason = "container was unhealthy"
			result.NewState = "restarting"
			return result, nil
		}
		result.Action = "notified"
		result.Reason = "container is unhealthy, auto_restart disabled"
		result.NewState = detail.Status
		return result, nil
	}

	// If exited with OOMKilled
	if detail.OOMKilled {
		if h.config.AutoRestart {
			recentRestarts := h.tracker.CountRecentRestarts(containerName, h.config.RestartWindow)
			if h.needsRollback(containerName, recentRestarts) {
				result.Action = "rolled_back"
				result.Reason = fmt.Sprintf("container OOMKilled, restart count %d exceeded max %d", recentRestarts, h.config.MaxRestarts)
				result.NewState = "rolled_back"
				return result, nil
			}
			if err := h.RestartContainer(ctx, containerName); err != nil {
				result.Action = "notified"
				result.Reason = fmt.Sprintf("container OOMKilled, restart failed: %v", err)
				result.NewState = detail.Status
				return result, nil
			}
			h.tracker.RecordRestart(containerName)
			result.Action = "restarted"
			result.Reason = "container was OOMKilled"
			result.NewState = "restarting"
			return result, nil
		}
		result.Action = "notified"
		result.Reason = "container was OOMKilled, auto_restart disabled"
		result.NewState = detail.Status
		return result, nil
	}

	// If exited with non-zero exit code
	if detail.Status == "exited" && detail.ExitCode != 0 {
		if h.config.AutoRestart {
			recentRestarts := h.tracker.CountRecentRestarts(containerName, h.config.RestartWindow)
			if h.needsRollback(containerName, recentRestarts) {
				result.Action = "rolled_back"
				result.Reason = fmt.Sprintf("container exited with code %d, restart count %d exceeded max %d", detail.ExitCode, recentRestarts, h.config.MaxRestarts)
				result.NewState = "rolled_back"
				return result, nil
			}
			if err := h.RestartContainer(ctx, containerName); err != nil {
				result.Action = "notified"
				result.Reason = fmt.Sprintf("container exited with code %d, restart failed: %v", detail.ExitCode, err)
				result.NewState = detail.Status
				return result, nil
			}
			h.tracker.RecordRestart(containerName)
			result.Action = "restarted"
			result.Reason = fmt.Sprintf("container exited with code %d", detail.ExitCode)
			result.NewState = "restarting"
			return result, nil
		}
		result.Action = "notified"
		result.Reason = fmt.Sprintf("container exited with code %d, auto_restart disabled", detail.ExitCode)
		result.NewState = detail.Status
		return result, nil
	}

	// If restarting (crash loop)
	if detail.Status == "restarting" {
		// Use the container's actual restart count from docker inspect
		recentRestarts := h.tracker.CountRecentRestarts(containerName, h.config.RestartWindow)
		effectiveRestarts := recentRestarts + detail.RestartCount
		if h.needsRollback(containerName, effectiveRestarts) {
			result.Action = "rolled_back"
			result.Reason = fmt.Sprintf("container in crash loop, restart count %d exceeded max %d", effectiveRestarts, h.config.MaxRestarts)
			result.NewState = "rolled_back"
			return result, nil
		}
		h.tracker.RecordRestart(containerName)
		result.Action = "notified"
		result.Reason = fmt.Sprintf("container is restarting, restart count %d/%d", effectiveRestarts+1, h.config.MaxRestarts)
		result.NewState = detail.Status
		return result, nil
	}

	// Default: no action
	result.Action = "none"
	result.NewState = detail.Status
	result.Reason = fmt.Sprintf("container status %s, health %s", detail.Status, detail.Health)
	return result, nil
}

// RestartContainer restarts a container.
func (h *Healer) RestartContainer(ctx context.Context, name string) error {
	_, err := h.executor.RunCommand(ctx, fmt.Sprintf("docker restart %s", util.ShellQuote(name)))
	if err != nil {
		return fmt.Errorf("failed to restart container %s: %w", name, err)
	}
	return nil
}

// needsRollback determines if rollback should be triggered based on restart count.
func (h *Healer) needsRollback(containerName string, restartCount int) bool {
	return h.config.AutoRollback && restartCount >= h.config.MaxRestarts
}

// GetConfig returns the current healing configuration.
func (h *Healer) GetConfig() HealingConfig {
	return h.config
}

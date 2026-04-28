package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
)

// ---------- 11. DetectEnv ----------

func (b *Bridge) DetectEnv(ctx context.Context, level int, ports []int, services []string) (interface{}, error) {
	env := map[string]interface{}{}

	// Level 1: OS info
	if level >= 1 {
		if out, err := b.Executor.RunCommand(ctx, "uname -a"); err == nil {
			env["os"] = strings.TrimSpace(out)
		}
		if out, err := b.Executor.RunCommand(ctx, "cat /etc/os-release 2>/dev/null | head -5"); err == nil {
			env["os_release"] = strings.TrimSpace(out)
		}
	}

	// Level 2: Docker info
	if level >= 2 {
		if out, err := b.Executor.RunCommand(ctx, "docker version --format '{{.Server.Version}}' 2>/dev/null"); err == nil {
			env["docker_version"] = strings.TrimSpace(out)
		}
		if out, err := b.Executor.RunCommand(ctx, "docker info --format '{{.NCPU}}' 2>/dev/null"); err == nil {
			env["docker_cpus"] = strings.TrimSpace(out)
		}
		if out, err := b.Executor.RunCommand(ctx, "docker info --format '{{.MemTotal}}' 2>/dev/null"); err == nil {
			env["docker_memory"] = strings.TrimSpace(out)
		}
	}

	// Level 3: Port checks
	if level >= 3 {
		portResults := map[string]bool{}
		for _, p := range ports {
			cmd := fmt.Sprintf("ss -tlnp 2>/dev/null | grep ':%d ' || true", p)
			if out, err := b.Executor.RunCommand(ctx, cmd); err == nil && strings.TrimSpace(out) != "" {
				portResults[fmt.Sprintf("%d", p)] = true
			} else {
				portResults[fmt.Sprintf("%d", p)] = false
			}
		}
		env["ports"] = portResults
	}

	// Level 4: Service checks
	if level >= 4 {
		svcResults := map[string]bool{}
		for _, svc := range services {
			svc = strings.TrimSpace(svc)
			if svc == "" {
				continue
			}
			cmd := fmt.Sprintf("timeout 2 bash -c 'echo > /dev/tcp/%s' 2>/dev/null && echo ok || echo fail", strings.TrimPrefix(svc, "tcp://"))
			if out, err := b.Executor.RunCommand(ctx, cmd); err == nil && strings.Contains(out, "ok") {
				svcResults[svc] = true
			} else {
				svcResults[svc] = false
			}
		}
		env["services"] = svcResults
	}

	return env, nil
}
// ---------- 12. HealthCheck ----------

func (b *Bridge) HealthCheck(ctx context.Context, target, healthType string) (interface{}, error) {
	cfg := deployer.HealthCheckConfig{
		Type:     defaultVal(healthType, "http"),
		Target:   target,
		Timeout:  5 * time.Second,
		Interval: 3 * time.Second,
		Retries:  3,
	}

	start := time.Now()
	err := b.d().HealthCheck(ctx, cfg)
	elapsed := time.Since(start)

	if err != nil {
		return map[string]interface{}{
			"status":   "unhealthy",
			"target":   target,
			"type":     cfg.Type,
			"error":    err.Error(),
			"duration": elapsed.String(),
		}, nil
	}

	return map[string]interface{}{
		"status":   "healthy",
		"target":   target,
		"type":     cfg.Type,
		"duration": elapsed.String(),
	}, nil
}
// ---------- 33. CheckDeployReadiness ----------

func (b *Bridge) CheckDeployReadiness(ctx context.Context, appConfig map[string]interface{}) (interface{}, error) {
	checks := map[string]interface{}{}

	// Check Docker availability
	if out, err := b.Executor.RunCommand(ctx, "docker version --format '{{.Server.Version}}' 2>/dev/null"); err == nil {
		checks["docker"] = map[string]interface{}{"available": true, "version": strings.TrimSpace(out)}
	} else {
		checks["docker"] = map[string]interface{}{"available": false, "error": err.Error()}
	}

	// Check if the requested port is free
	if portsRaw, ok := appConfig["ports"]; ok {
		portsStr := fmt.Sprintf("%v", portsRaw)
		hostPort := portsStr
		if idx := strings.Index(portsStr, ":"); idx >= 0 {
			hostPort = portsStr[:idx]
		}
		cmd := fmt.Sprintf("ss -tlnp 2>/dev/null | grep ':%s ' || true", hostPort)
		if out, err := b.Executor.RunCommand(ctx, cmd); err == nil && strings.TrimSpace(out) == "" {
			checks["port"] = map[string]interface{}{"port": hostPort, "available": true}
		} else {
			checks["port"] = map[string]interface{}{"port": hostPort, "available": false, "in_use_by": strings.TrimSpace(out)}
		}
	}

	overallReady := true
	for _, v := range checks {
		if m, ok := v.(map[string]interface{}); ok {
			if avail, ok := m["available"].(bool); ok && !avail {
				overallReady = false
			}
		}
	}

	return map[string]interface{}{
		"ready":  overallReady,
		"checks": checks,
	}, nil
}
// ---------- 39. CheckSystemUpdate ----------
// Implemented in upgrade.go

// PerformSystemUpdate delegates to UpgradeService.
func (b *Bridge) PerformSystemUpdate(ctx context.Context) (interface{}, error) {
	if b.UpgradeSvc == nil {
		return nil, fmt.Errorf("upgrade service not initialized")
	}
	return b.UpgradeSvc.PerformUpgrade(ctx, "latest")
}
// ---------- 40. HealContainer ----------

func (b *Bridge) HealContainer(ctx context.Context, containerName string) (interface{}, error) {
	h := b.getHealer()
	result, err := h.CheckAndHeal(ctx, containerName)
	if err != nil {
		return nil, fmt.Errorf("heal failed for %s: %w", containerName, err)
	}

	// If healer determined a rollback is needed, execute it automatically
	if result.Action == "rolled_back" && h.GetConfig().AutoRollback {
		slog.Info("healer: initiating auto-rollback", "container", containerName, "reason", result.Reason)
		_, rollbackErr := b.RollbackWithType(ctx, containerName, "", "auto_rollback")
		if rollbackErr != nil {
			slog.Error("healer: auto-rollback failed", "container", containerName, "error", rollbackErr)
			result.Action = "rollback_failed"
			result.Reason = fmt.Sprintf("%s (rollback failed: %v)", result.Reason, rollbackErr)
		} else {
			result.NewState = "rolled_back"
			slog.Info("healer: auto-rollback succeeded", "container", containerName)
		}
	}

	return result, nil
}

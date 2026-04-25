package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Yogdunana/deploypilot/internal/mcp"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// batchDeployer abstracts the single-app deploy operation for testability.
type batchDeployer interface {
	deploySingle(ctx context.Context, index int, appCfg map[string]interface{}) mcp.BatchDeployItemResult
}

// bridgeDeployer adapts Bridge to the batchDeployer interface.
type bridgeDeployer struct {
	bridge *Bridge
}

func (d *bridgeDeployer) deploySingle(ctx context.Context, index int, appCfg map[string]interface{}) mcp.BatchDeployItemResult {
	cfg := mcp.DeployConfig{
		Image:         toStringOrDefault(appCfg["image"], ""),
		ContainerName: toStringOrDefault(appCfg["container_name"], fmt.Sprintf("batch-app-%d", index)),
		Ports:         toStringOrDefault(appCfg["ports"], ""),
		RestartPolicy: toStringOrDefault(appCfg["restart_policy"], "unless-stopped"),
	}
	if envRaw, ok := appCfg["env_vars"]; ok {
		if s, ok := envRaw.(string); ok && s != "" {
			var m map[string]string
			if json.Unmarshal([]byte(s), &m) == nil {
				cfg.EnvVars = m
			}
		}
	}

	appName := cfg.ContainerName
	cs, err := d.bridge.Deploy(ctx, cfg)
	if err != nil {
		return mcp.BatchDeployItemResult{
			Index:   index,
			AppName: appName,
			Success: false,
			Error:   err.Error(),
		}
	}
	slog.Info("batch deploy: app deployed successfully",
		"index", index, "app", appName, "container_id", cs.ID)
	return mcp.BatchDeployItemResult{
		Index:   index,
		AppName: appName,
		Success: true,
	}
}

// executeBatchDeploy runs the batch deployment using the given strategy.
func executeBatchDeploy(ctx context.Context, deployer batchDeployer, config mcp.BatchDeployConfig) *mcp.BatchDeployResult {
	start := time.Now()

	if len(config.Apps) == 0 {
		return &mcp.BatchDeployResult{
			Total:    0,
			Success:  0,
			Failed:   0,
			Results:  nil,
			Duration: time.Since(start).Seconds(),
		}
	}

	// Normalize strategy: fall back to sequential for unknown values
	strategy := config.Strategy
	switch strategy {
	case mcp.StrategySequential, mcp.StrategyParallel, mcp.StrategyRolling:
		// valid
	default:
		slog.Warn("batch deploy: unknown strategy, falling back to sequential", "strategy", strategy)
		strategy = mcp.StrategySequential
	}

	var results []mcp.BatchDeployItemResult
	switch strategy {
	case mcp.StrategyParallel:
		results = deployParallel(ctx, deployer, config)
	case mcp.StrategyRolling:
		results = deployRolling(ctx, deployer, config)
	default:
		results = deploySequential(ctx, deployer, config)
	}

	success, failed := 0, 0
	for _, r := range results {
		if r.Success {
			success++
		} else {
			failed++
		}
	}

	return &mcp.BatchDeployResult{
		Total:    len(config.Apps),
		Success:  success,
		Failed:   failed,
		Results:  results,
		Duration: time.Since(start).Seconds(),
	}
}

// deploySequential deploys apps one by one.
func deploySequential(ctx context.Context, d batchDeployer, config mcp.BatchDeployConfig) []mcp.BatchDeployItemResult {
	results := make([]mcp.BatchDeployItemResult, 0, len(config.Apps))
	for i, appCfg := range config.Apps {
		result := d.deploySingle(ctx, i, appCfg)
		results = append(results, result)
	}
	return results
}

// deployParallel deploys all apps concurrently with a concurrency limit.
func deployParallel(ctx context.Context, d batchDeployer, config mcp.BatchDeployConfig) []mcp.BatchDeployItemResult {
	maxConcurrent := config.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	sem := semaphore.NewWeighted(int64(maxConcurrent))
	g, gCtx := errgroup.WithContext(ctx)

	results := make([]mcp.BatchDeployItemResult, len(config.Apps))
	var mu sync.Mutex

	for i, appCfg := range config.Apps {
		i, appCfg := i, appCfg
		if err := sem.Acquire(gCtx, 1); err != nil {
			// Context cancelled, record remaining as failed
			mu.Lock()
			results[i] = mcp.BatchDeployItemResult{
				Index:   i,
				AppName: toStringOrDefault(appCfg["container_name"], fmt.Sprintf("batch-app-%d", i)),
				Success: false,
				Error:   "context cancelled",
			}
			mu.Unlock()
			continue
		}
		g.Go(func() error {
			defer sem.Release(1)
			result := d.deploySingle(gCtx, i, appCfg)
			mu.Lock()
			results[i] = result
			mu.Unlock()
			// Return nil so other goroutines are not cancelled
			return nil
		})
	}

	_ = g.Wait()
	return results
}

// deployRolling deploys apps in batches, waiting for each batch to complete before starting the next.
func deployRolling(ctx context.Context, d batchDeployer, config mcp.BatchDeployConfig) []mcp.BatchDeployItemResult {
	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 3
	}

	results := make([]mcp.BatchDeployItemResult, 0, len(config.Apps))

	for batchStart := 0; batchStart < len(config.Apps); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(config.Apps) {
			batchEnd = len(config.Apps)
		}

		slog.Info("batch deploy: rolling batch starting",
			"batch", batchStart/batchSize+1,
			"range", fmt.Sprintf("[%d,%d)", batchStart, batchEnd))

		for i := batchStart; i < batchEnd; i++ {
			result := d.deploySingle(ctx, i, config.Apps[i])
			results = append(results, result)
		}
	}

	return results
}

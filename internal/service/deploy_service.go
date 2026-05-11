package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Yogdunana/deploypilot/internal/engine/builder"
	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/metrics"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/tracing"
	"github.com/Yogdunana/deploypilot/internal/util/timeutil"
	"gorm.io/gorm"
)

// ---------- 1. Deploy ----------

// DeployAsync starts a deploy in a goroutine and returns a task ID immediately.
// It uses a detached context so the deploy is not cancelled when the HTTP request ends.
func (b *Bridge) DeployAsync(ctx context.Context, cfg mcp.DeployConfig, appID string) (taskID string, err error) {
	taskID = b.createTask("deploy")
	traceID := tracing.TraceIDFromContext(ctx)
	b.updateTask(taskID, "running", 0, "deploy started")

	// Use a detached context with timeout so the deploy continues even if
	// the original HTTP request context is cancelled (e.g. client disconnects),
	// but still has a reasonable upper bound to prevent runaway goroutines.
	detachedCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)

	go func() {
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("deploy panicked", "task_id", taskID, "app_id", appID, "panic", r)
				b.updateTask(taskID, "failed", 100, "deploy panicked: see server logs for details")
				b.EventBus.Publish(DeployEvent{
					TaskID:    taskID,
					AppID:     appID,
					Step:      "done",
					Status:    "failed",
					Progress:  100,
					Message:   "deploy panicked: see server logs for details",
					Timestamp: timeutil.FormatRFC3339(),
					TraceID:   traceID,
				})
			}
		}()

		cs, deployErr := b.Deploy(detachedCtx, cfg)
		if deployErr != nil {
			slog.Error("deploy failed", "task_id", taskID, "app_id", appID, "error", deployErr)
			b.updateTask(taskID, "failed", 100, "deploy failed: see server logs for details")
			b.EventBus.Publish(DeployEvent{
				TaskID:    taskID,
				AppID:     appID,
				Step:      "done",
				Status:    "failed",
				Progress:  100,
				Message:   "deploy failed: see server logs for details",
				Timestamp: timeutil.FormatRFC3339(),
				TraceID:   traceID,
			})
		} else {
			b.updateTask(taskID, "success", 100, "deploy completed")
			b.EventBus.Publish(DeployEvent{
				TaskID:    taskID,
				AppID:     appID,
				Step:      "done",
				Status:    "success",
				Progress:  100,
				Message:   "deploy completed",
				Timestamp: timeutil.FormatRFC3339(),
				TraceID:   traceID,
			})
			b.taskMu.Lock()
			if t, ok := b.tasks[taskID]; ok {
				t.Result = cs
			}
			b.taskMu.Unlock()
		}
	}()
	return taskID, nil
}

func (b *Bridge) Deploy(ctx context.Context, cfg mcp.DeployConfig) (*mcp.ContainerStatus, error) {
	// Record deploy start time for metrics
	deployStart := timeutil.Now()

	// Determine executor: remote SSH if server_id provided, otherwise local
	executor := b.Executor
	var host string
	var port int

	if cfg.ServerID != "" {
		// Look up server info for preflight
		row := make(map[string]interface{})
		if err := b.DB.Table("servers").Where("id = ?", cfg.ServerID).Take(&row).Error; err != nil {
			return nil, fmt.Errorf("server not found: %w", err)
		}
		host = toString(row["host"])
		port = toInt(row["port"])

		remoteExec, err := b.getRemoteExecutor(ctx, cfg.ServerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get remote executor for server %s: %w", cfg.ServerID, err)
		}
		defer func() {
			if cerr := remoteExec.Close(); cerr != nil {
				slog.Warn("failed to close remote executor", "error", cerr)
			}
		}()
		executor = remoteExec
	}

	// Run preflight checks
	pfCfg := PreflightConfig{
		Host:         host,
		Port:         port,
		Executor:     executor,
		PortMappings: cfg.Ports,
	}
	pfResult := RunPreflight(ctx, pfCfg)
	if !pfResult.Passed {
		// Save preflight failure to deployment records
		b.saveDeploymentRecord(ctx, mcp.DeployConfig{
			Image:         cfg.Image,
			ContainerName: cfg.ContainerName,
			ServerID:      cfg.ServerID,
		}, "preflight_failed", pfResult)

		// Log structured preflight failure
		logPreflightResult(cfg.ContainerName, pfResult)

		b.EventBus.Publish(DeployEvent{
		TaskID:    "", AppID: cfg.ContainerName,
		Step: "preflight", Status: "failed", Progress: 20,
		Message:   pfResult.Message,
		Timestamp: timeutil.FormatRFC3339(),
	})

	// Record preflight failure metric
	metrics.DeployTotal.WithLabelValues(cfg.ContainerName, cfg.ServerID, "failed").Inc()
	metrics.DeployDuration.Observe(timeutil.Since(deployStart).Seconds())

		return nil, &PreflightError{
			Code:    pfResult.Code,
			Message: pfResult.Message,
			Checks:  pfResult.Checks,
		}
	}

	b.EventBus.Publish(DeployEvent{
		TaskID:    "", AppID: cfg.ContainerName,
		Step: "preflight", Status: "success", Progress: 20,
		Message:   "preflight checks passed",
		Timestamp: timeutil.FormatRFC3339(),
	})

	dCfg := deployer.DeployConfig{
		Image:         cfg.Image,
		ContainerName: cfg.ContainerName,
		Ports:         cfg.Ports,
		EnvVars:       cfg.EnvVars,
		RestartPolicy: cfg.RestartPolicy,
		Network:       cfg.Network,
		Volumes:       cfg.Volumes,
		Labels:        cfg.Labels,
		ResourceLimits: deployer.ResourceLimits{
			CPU:    cfg.CPU,
			Memory: cfg.Memory,
		},
	}

	d := deployer.New(executor)

	b.EventBus.Publish(DeployEvent{
		TaskID:    "", AppID: cfg.ContainerName,
		Step: "pull", Status: "running", Progress: 30,
		Message:   "pulling image: " + cfg.Image,
		Timestamp: timeutil.FormatRFC3339(),
	})

	cs, err := d.Deploy(ctx, dCfg)
	if err != nil {
		// Save deployment failure
		b.saveDeploymentRecord(ctx, cfg, "failed", nil)
		slog.Error("container deployment failed", "container", cfg.ContainerName, "error", err)

		// Record deploy failure metric
		metrics.DeployTotal.WithLabelValues(cfg.ContainerName, cfg.ServerID, "failed").Inc()
		metrics.DeployDuration.Observe(timeutil.Since(deployStart).Seconds())

		slog.Error("deploy run failed", "app_id", cfg.ContainerName, "server_id", cfg.ServerID, "error", err)
		b.EventBus.Publish(DeployEvent{
			TaskID:    "", AppID: cfg.ContainerName,
			Step: "run", Status: "failed", Progress: 60,
			Message:   "deploy failed: see server logs for details",
			Timestamp: timeutil.FormatRFC3339(),
		})

		return nil, err
	}
	// Save deployment success
	b.saveDeploymentRecord(ctx, cfg, "success", nil)
	slog.Info("container deployed successfully", "container", cfg.ContainerName, "container_id", cs.ID)

	// Record deploy success metric
	metrics.DeployTotal.WithLabelValues(cfg.ContainerName, cfg.ServerID, "success").Inc()
	metrics.DeployDuration.Observe(timeutil.Since(deployStart).Seconds())

	b.EventBus.Publish(DeployEvent{
		TaskID:    "", AppID: cfg.ContainerName,
		Step: "run", Status: "success", Progress: 90,
		Message:   "container deployed successfully",
		Timestamp: timeutil.FormatRFC3339(),
	})

	return &mcp.ContainerStatus{
		ID:        cs.ID,
		Name:      cs.Name,
		Image:     cs.Image,
		Status:    cs.Status,
		Ports:     cs.Ports,
		CreatedAt: cs.CreatedAt.UTC().Format(time.RFC3339),
		Labels:    cs.Labels,
	}, nil
}

// ---------- 1b. BuildAndDeploy ----------

// BuildAndDeploy orchestrates the full build-and-deploy pipeline:
// git clone -> detect tech stack -> generate Dockerfile -> docker build -> deploy.
func (b *Bridge) BuildAndDeploy(ctx context.Context, cfg mcp.BuildAndDeployConfig) (*mcp.BuildAndDeployResult, error) {
	exec := b.Executor
	if exec == nil {
		return nil, fmt.Errorf("no executor available")
	}

	// Convert MCP config to builder config
	bldCfg := builder.BuildConfig{
		RepoURL:             cfg.RepoURL,
		Branch:              cfg.Branch,
		TechStack:           cfg.TechStack,
		AppName:             cfg.AppName,
		ProjectDir:          cfg.ProjectDir,
		BuildArgs:           cfg.BuildArgs,
		EnvVars:             cfg.EnvVars,
		Ports:               cfg.Ports,
		ServerID:            cfg.ServerID,
		DockerfileOverrides: cfg.DockerfileOverrides,
		RegistryID:          cfg.RegistryID,
		PushImage:           cfg.PushImage,
		ImageTag:            cfg.ImageTag,
	}

	bld := builder.NewBuilder(exec)
	result, err := bld.BuildAndDeploy(ctx, bldCfg)
	if err != nil {
		return nil, err
	}

	// After successful build, deploy the built image
	deployCfg := mcp.DeployConfig{
		Image:         result.Image,
		ContainerName: cfg.AppName,
		Ports:         cfg.Ports,
		EnvVars:       cfg.EnvVars,
	}
	if cfg.ServerID != "" {
		deployCfg.ServerID = cfg.ServerID
	}

	_, err = b.Deploy(ctx, deployCfg)
	if err != nil {
		return nil, fmt.Errorf("build succeeded but deploy failed: %w", err)
	}

	// Convert builder result to MCP result
	return &mcp.BuildAndDeployResult{
		Image:      result.Image,
		Digest:     result.Digest,
		Size:       result.Size,
		BuildLog:   result.BuildLog,
		Duration:   result.Duration,
		TechStack:  result.TechStack,
		CommitHash: result.CommitHash,
	}, nil
}
// ---------- 2. GetContainerStatus ----------

func (b *Bridge) GetContainerStatus(ctx context.Context, name string) (*mcp.ContainerStatus, error) {
	cs, err := b.d().GetContainerStatus(ctx, name)
	if err != nil {
		return nil, err
	}

	return &mcp.ContainerStatus{
		ID:        cs.ID,
		Name:      cs.Name,
		Image:     cs.Image,
		Status:    cs.Status,
		Ports:     cs.Ports,
		CreatedAt: cs.CreatedAt.UTC().Format(time.RFC3339),
		Labels:    cs.Labels,
	}, nil
}
// ---------- 7. Stop ----------

func (b *Bridge) Stop(ctx context.Context, name string) error {
	return b.d().Stop(ctx, name)
}
// ---------- 8. Remove ----------

func (b *Bridge) Remove(ctx context.Context, name string) error {
	return b.d().Remove(ctx, name)
}
// ---------- 9. GetContainerLogs ----------

func (b *Bridge) GetContainerLogs(ctx context.Context, name string, tail int) (string, error) {
	return b.d().GetContainerLogs(ctx, name, tail)
}
// ---------- 10. Rollback ----------

// RollbackResult contains detailed information about a rollback operation.
type RollbackResult struct {
	ContainerStatus *mcp.ContainerStatus `json:"container_status"`
	PreviousImage   string               `json:"previous_image"`
	TargetImage     string               `json:"target_image"`
	DeployType      string               `json:"deploy_type"` // rollback or auto_rollback
}

// Rollback reverts a container to a previous image version with full config preservation.
// If previousImage is empty, it automatically finds the last successful deployment image.
// The rollback preserves ports, env vars, volumes, network, labels, and resource limits.
func (b *Bridge) Rollback(ctx context.Context, containerName, previousImage string) (*mcp.ContainerStatus, error) {
	return b.RollbackWithType(ctx, containerName, previousImage, "rollback")
}
// RollbackWithType performs a rollback with a specified deploy type (rollback or auto_rollback).
func (b *Bridge) RollbackWithType(ctx context.Context, containerName, previousImage, deployType string) (*mcp.ContainerStatus, error) {
	if deployType == "" {
		deployType = "rollback"
	}

	// Step 1: Resolve the target image — auto-find if not specified
	if previousImage == "" {
		found, err := b.findPreviousSuccessfulImage(ctx, containerName)
		if err != nil {
			return nil, fmt.Errorf("no previous image found for rollback: %w", err)
		}
		previousImage = found
		slog.Info("rollback: auto-resolved previous image", "container", containerName, "image", previousImage)
	}

	// Step 2: Get current running image for recording
	currentImage := ""
	if cs, err := b.d().GetContainerStatus(ctx, containerName); err == nil {
		currentImage = cs.Image
	}

	// Step 3: Build full deploy config from app record (preserves all settings)
	cfg, err := b.buildRollbackConfig(ctx, containerName, previousImage)
	if err != nil {
		return nil, fmt.Errorf("failed to build rollback config: %w", err)
	}

	// Step 4: Atomic rollback — pull new image FIRST, then stop/remove old container
	// This ensures we don't destroy the running container if the image pull fails
	slog.Info("rollback: pulling target image", "container", containerName, "image", previousImage)
	pullCmd := fmt.Sprintf("docker pull %s", previousImage)
	if _, err := b.Executor.RunCommand(ctx, pullCmd); err != nil {
		return nil, fmt.Errorf("rollback failed: could not pull image %s: %w", previousImage, err)
	}

	// Step 5: Stop and remove the current container
	if err := b.d().Stop(ctx, containerName); err != nil {
		slog.Warn("rollback: stop warning", "error", err)
	}
	if err := b.d().Remove(ctx, containerName); err != nil {
		slog.Warn("rollback: remove warning", "error", err)
	}

	// Step 6: Deploy with full config
	cs, err := b.Deploy(ctx, cfg)
	if err != nil {
		// Record failed rollback
		b.saveRollbackRecord(ctx, cfg, currentImage, deployType, "failed", err.Error())
		return nil, fmt.Errorf("rollback deploy failed: %w", err)
	}

	// Step 7: Record successful rollback
	b.saveRollbackRecord(ctx, cfg, currentImage, deployType, "success", "")

	// Step 8: Update app's current version
	if cfg.AppName != "" {
		b.DB.Table("apps").Where("id = ? OR name = ?", cfg.AppName, cfg.AppName).
			Update("current_version", previousImage)
	}

	slog.Info("rollback completed", "container", containerName,
		"from", currentImage, "to", previousImage, "type", deployType)

	return cs, nil
}
// findPreviousSuccessfulImage looks up the most recent successful deployment record
// for a container and returns its image.
func (b *Bridge) findPreviousSuccessfulImage(ctx context.Context, containerName string) (string, error) {
	if b.DB == nil {
		return "", fmt.Errorf("database not available")
	}

	var record model.DeploymentRecord
	err := b.DB.Where("container_name = ? AND status = ?", containerName, "success").
		Order("created_at DESC").
		Limit(1).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("no successful deployment found for container %s", containerName)
		}
		return "", fmt.Errorf("failed to query deployment record: %w", err)
	}

	return record.Image, nil
}
// buildRollbackConfig constructs a full DeployConfig for rollback by reading the app record.
// This ensures ports, env vars, resource limits, etc. are preserved during rollback.
func (b *Bridge) buildRollbackConfig(ctx context.Context, containerName, targetImage string) (mcp.DeployConfig, error) {
	cfg := mcp.DeployConfig{
		Image:         targetImage,
		ContainerName: containerName,
		RestartPolicy: "unless-stopped",
	}

	if b.DB == nil {
		return cfg, nil
	}

	// Look up the app by container_name or name
	var app model.App
	err := b.DB.Where("container_name = ? OR name = ?", containerName, containerName).First(&app).Error
	if err != nil {
		slog.Debug("rollback: no app record found, using minimal config", "container", containerName)
		return cfg, nil
	}

	cfg.AppName = app.Name

	// Restore ports from app record
	if app.Domain != "" {
		cfg.Ports = "80:80, 443:443"
	}

	// Restore env vars
	if app.EnvVars != "" {
		var envMap map[string]string
		if json.Unmarshal([]byte(app.EnvVars), &envMap) == nil {
			cfg.EnvVars = envMap
		}
	}

	// Restore resource limits
	if app.ResourceLimits != "" {
		var limits map[string]string
		if json.Unmarshal([]byte(app.ResourceLimits), &limits) == nil {
			cfg.CPU = limits["cpu"]
			cfg.Memory = limits["memory"]
		}
	}

	// Try to get the config snapshot from the last successful deployment
	var lastRecord model.DeploymentRecord
	if err := b.DB.Where("container_name = ? AND status = ?", containerName, "success").
		Order("created_at DESC").Limit(1).First(&lastRecord).Error; err == nil {
		if lastRecord.ConfigSnapshot != "" {
			var snapCfg mcp.DeployConfig
			if json.Unmarshal([]byte(lastRecord.ConfigSnapshot), &snapCfg) == nil {
				// Merge snapshot config (takes precedence over app record)
				if snapCfg.Ports != "" {
					cfg.Ports = snapCfg.Ports
				}
				if len(snapCfg.EnvVars) > 0 {
					cfg.EnvVars = snapCfg.EnvVars
				}
				if snapCfg.Network != "" {
					cfg.Network = snapCfg.Network
				}
				if snapCfg.Volumes != "" {
					cfg.Volumes = snapCfg.Volumes
				}
				if len(snapCfg.Labels) > 0 {
					cfg.Labels = snapCfg.Labels
				}
				if snapCfg.CPU != "" {
					cfg.CPU = snapCfg.CPU
				}
				if snapCfg.Memory != "" {
					cfg.Memory = snapCfg.Memory
				}
				if snapCfg.RestartPolicy != "" {
					cfg.RestartPolicy = snapCfg.RestartPolicy
				}
			}
		}
	}

	return cfg, nil
}
// saveRollbackRecord persists a rollback operation to the deployment records.
func (b *Bridge) saveRollbackRecord(ctx context.Context, cfg mcp.DeployConfig, currentImage, deployType, status, errMsg string) {
	if b.DB == nil {
		return
	}

	// Serialize config snapshot
	snapshotJSON, _ := json.Marshal(cfg)

	record := &model.DeploymentRecord{
		ID:             generateID(),
		TenantID:       model.DefaultTenantID,
		ServerID:       cfg.ServerID,
		AppName:        cfg.AppName,
		ContainerName:  cfg.ContainerName,
		Image:          cfg.Image,
		PreviousImage:  currentImage,
		DeployType:     deployType,
		ConfigSnapshot: string(snapshotJSON),
		Status:         status,
		ErrorMessage:   errMsg,
		CreatedAt:      timeutil.Now(),
		UpdatedAt:      timeutil.Now(),
	}
	if err := b.DB.Create(record).Error; err != nil {
		slog.Error("failed to save rollback record", "error", err)
	}
}
// GetDeploymentHistory returns the deployment history for a container, ordered by most recent first.
func (b *Bridge) GetDeploymentHistory(ctx context.Context, containerName string, limit int) ([]model.DeploymentRecord, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var records []model.DeploymentRecord
	err := b.DB.Where("container_name = ?", containerName).
		Order("created_at DESC").
		Limit(limit).
		Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query deployment history: %w", err)
	}
	return records, nil
}
// ---------- helpers ----------

// saveDeploymentRecord persists a deployment attempt to the database.
func (b *Bridge) saveDeploymentRecord(ctx context.Context, cfg mcp.DeployConfig, status string, pfResult *PreflightResult) {
	if b.DB == nil {
		return
	}

	// Serialize config snapshot for rollback support
	snapshotJSON, _ := json.Marshal(cfg)

	// Resolve app_id from app_name if possible
	var appID string
	if cfg.AppName != "" {
		var app model.App
		if err := b.DB.Select("id").Where("name = ?", cfg.AppName).First(&app).Error; err == nil {
			appID = app.ID
		}
	}

	record := &model.DeploymentRecord{
		ID:             generateID(),
		TenantID:       model.DefaultTenantID,
		ServerID:       cfg.ServerID,
		AppName:        cfg.AppName,
		AppID:          appID,
		ContainerName:  cfg.ContainerName,
		Image:          cfg.Image,
		DeployType:     "deploy",
		ConfigSnapshot: string(snapshotJSON),
		Status:         status,
		CreatedAt:      timeutil.Now(),
		UpdatedAt:      timeutil.Now(),
	}
	if pfResult != nil {
		record.PreflightCode = string(pfResult.Code)
		record.PreflightMessage = pfResult.Message
		checksJSON, _ := json.Marshal(pfResult.Checks)
		record.PreflightChecks = string(checksJSON)
	}
	if err := b.DB.Create(record).Error; err != nil {
		slog.Error("failed to save deployment record", "error", err)
	}
}
// GetLatestDeploymentRecord returns the most recent deployment record for a container.
func (b *Bridge) GetLatestDeploymentRecord(ctx context.Context, containerName string) (*model.DeploymentRecord, error) {
	var record model.DeploymentRecord
	err := b.DB.Where("container_name = ?", containerName).
		Order("created_at DESC").
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}
// ---------- 34. BatchDeploy ----------

// BatchDeploy deploys multiple applications using the configured strategy.
// It accepts the legacy []map[string]interface{} parameter (backward compatible)
// and defaults to sequential strategy.
func (b *Bridge) BatchDeploy(ctx context.Context, apps []map[string]interface{}) (interface{}, error) {
	config := mcp.BatchDeployConfig{
		Apps:     apps,
		Strategy: mcp.StrategySequential, // default
	}
	d := &bridgeDeployer{bridge: b}
	result := executeBatchDeploy(ctx, d, config)
	return result, nil
}
// BatchDeployWithConfig deploys multiple applications using the given configuration.
func (b *Bridge) BatchDeployWithConfig(ctx context.Context, config mcp.BatchDeployConfig) (*mcp.BatchDeployResult, error) {
	d := &bridgeDeployer{bridge: b}
	result := executeBatchDeploy(ctx, d, config)
	return result, nil
}

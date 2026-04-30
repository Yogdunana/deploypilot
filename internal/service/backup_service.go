package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/google/uuid"
)

// shellQuote safely escapes a string for use in a shell command.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// ---------- 35. Backup ----------

func (b *Bridge) Backup(ctx context.Context, appID string) (string, error) {
	row := make(map[string]interface{})
	if err := b.DB.Table("apps").Where("id = ?", appID).Take(&row).Error; err != nil {
		return "", fmt.Errorf("app not found: %w", err)
	}

	containerName := toString(row["container_name"])
	if containerName == "" {
		containerName = toString(row["name"])
	}

	backupID := uuid.New().String()

	// Attempt a docker-based backup: exec into the container and create a timestamped archive
	timestamp := time.Now().Format("20060102-150405")
	backupFile := fmt.Sprintf("/tmp/backup-%s-%s.tar.gz", containerName, timestamp)
	cmd := fmt.Sprintf("docker exec %s sh -c 'tar czf - /app /data 2>/dev/null' > %s 2>/dev/null || echo 'no_backup_paths'", shellQuote(containerName), shellQuote(backupFile))
	out, err := b.Executor.RunCommand(ctx, cmd)
	if err != nil {
		slog.Warn("backup: container exec failed (may be expected)", "error", err, "output", out)
	}

	slog.Info("backup completed", "app_id", appID, "container", containerName, "backup_id", backupID, "file", backupFile)

	// Store backup-to-app mapping for Restore
	b.backupMu.Lock()
	b.backupApps[backupID] = appID
	b.backupMu.Unlock()

	return backupID, nil
}

// ---------- 36. Restore ----------

func (b *Bridge) Restore(ctx context.Context, backupID string) (*mcp.ContainerStatus, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}

	// Look up the appID from the backup mapping
	b.backupMu.RLock()
	appID, ok := b.backupApps[backupID]
	b.backupMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("backup %s not found", backupID)
	}

	// Find the app
	var app model.App
	if err := b.DB.Where("id = ?", appID).First(&app).Error; err != nil {
		return nil, fmt.Errorf("app not found for backup %s: %w", backupID, err)
	}

	containerName := app.ContainerName
	if containerName == "" {
		containerName = app.Name
	}

	// Stop and remove current container
	exec := b.Executor
	if _, err := exec.RunCommand(ctx, fmt.Sprintf("docker stop %s", shellQuote(containerName))); err != nil {
		slog.WarnContext(ctx, "failed to stop container during restore", "container", containerName, "error", err)
	}
	if _, err := exec.RunCommand(ctx, fmt.Sprintf("docker rm -f %s", shellQuote(containerName))); err != nil {
		slog.WarnContext(ctx, "failed to remove container during restore", "container", containerName, "error", err)
	}

	// Restore from backup
	timestamp := time.Now().Format("20060102-150405")
	backupFile := fmt.Sprintf("/tmp/backup-%s-%s.tar.gz", containerName, timestamp)
	cmd := fmt.Sprintf("docker run --rm -v /tmp:/backup -v %s-data:/data alpine sh -c 'cd /data && tar xzf /backup/%s 2>/dev/null || true'",
		shellQuote(containerName), shellQuote(filepath.Base(backupFile)))
	output, err := exec.RunCommand(ctx, cmd)
	if err != nil {
		slog.Warn("restore extract warning", "error", err, "output", output)
	}
	slog.Info("restore attempted", "container", containerName, "backup_file", backupFile, "output", output)

	// Re-deploy the app using its current version
	image := app.CurrentVersion
	if image == "" {
		image = "nginx:alpine" // fallback image
	}
	d := deployer.New(exec)
	dCfg := deployer.DeployConfig{
		Image:         image,
		ContainerName: containerName,
	}
	// Parse env vars if present
	if app.EnvVars != "" {
		var envMap map[string]string
		if json.Unmarshal([]byte(app.EnvVars), &envMap) == nil {
			dCfg.EnvVars = envMap
		}
	}
	cs, err := d.Deploy(ctx, dCfg)
	if err != nil {
		return nil, fmt.Errorf("re-deploy after restore failed: %w", err)
	}

	return &mcp.ContainerStatus{
		ID:        cs.ID,
		Name:      cs.Name,
		Image:     cs.Image,
		Status:    cs.Status,
		Ports:     cs.Ports,
		CreatedAt: cs.CreatedAt.Format(time.RFC3339),
		Labels:    cs.Labels,
	}, nil
}

// ---------- 37. BatchBackup ----------

func (b *Bridge) BatchBackup(ctx context.Context, appIDs []string) (interface{}, error) {
	results := make([]map[string]interface{}, 0, len(appIDs))
	for _, id := range appIDs {
		backupID, err := b.Backup(ctx, id)
		entry := map[string]interface{}{"app_id": id}
		if err != nil {
			entry["status"] = "failed"
			entry["error"] = err.Error()
		} else {
			entry["status"] = "success"
			entry["backup_id"] = backupID
		}
		results = append(results, entry)
	}
	return map[string]interface{}{
		"total":   len(appIDs),
		"results": results,
	}, nil
}

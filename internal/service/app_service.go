package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/google/uuid"
)

// ---------- 3. ListApps ----------

func (b *Bridge) ListApps(ctx context.Context) ([]mcp.ContainerStatus, error) {
	var rows []map[string]interface{}
	if err := b.DB.Table("apps").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to query apps: %w", err)
	}

	result := make([]mcp.ContainerStatus, 0, len(rows))
	for _, r := range rows {
		cs := mcp.ContainerStatus{
			ID:     toString(r["id"]),
			Name:   toString(r["name"]),
			Image:  toString(r["container_name"]),
			Status: toString(r["status"]),
		}
		if v, ok := r["env_vars"]; ok {
			if s, ok := v.(string); ok && s != "" {
				var m map[string]string
				if json.Unmarshal([]byte(s), &m) == nil {
					cs.Labels = m
				}
			}
		}
		result = append(result, cs)
	}
	return result, nil
}

// ---------- 5. CreateApp ----------

func (b *Bridge) CreateApp(ctx context.Context, cfg mcp.CreateAppConfig) (string, error) {
	id := uuid.New().String()
	if err := b.DB.Table("apps").Create(map[string]interface{}{
		"id":          id,
		"name":        cfg.Name,
		"repo_url":    cfg.RepoURL,
		"branch":      defaultVal(cfg.Branch, "main"),
		"domain":      cfg.Domain,
		"tech_stack":  defaultVal(cfg.TechStack, "docker"),
		"deploy_mode": defaultVal(cfg.DeployMode, "api"),
		"server_id":   cfg.ServerID,
		"status":      "created",
	}).Error; err != nil {
		return "", fmt.Errorf("failed to create app: %w", err)
	}
	return id, nil
}

// ---------- 6. DeleteApp ----------

func (b *Bridge) DeleteApp(ctx context.Context, appID string) error {
	row := make(map[string]interface{})
	if err := b.DB.Table("apps").Where("id = ?", appID).Take(&row).Error; err != nil {
		return fmt.Errorf("app not found: %w", err)
	}

	containerName := toString(row["container_name"])
	if containerName != "" {
		if err := b.d().Stop(ctx, containerName); err != nil {
			slog.WarnContext(ctx, "failed to stop container during app deletion", "container", containerName, "error", err)
		}
		if err := b.d().Remove(ctx, containerName); err != nil {
			slog.WarnContext(ctx, "failed to remove container during app deletion", "container", containerName, "error", err)
		}
	}

	if err := b.DB.Table("apps").Where("id = ?", appID).Delete(nil).Error; err != nil {
		return fmt.Errorf("failed to delete app from DB: %w", err)
	}
	return nil
}

// ---------- 10. GetAppDeploymentHistory ----------

func (b *Bridge) GetAppDeploymentHistory(ctx context.Context, appID string, limit int) ([]model.DeploymentRecord, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var records []model.DeploymentRecord
	err := b.DB.Where("app_id = ?", appID).
		Order("created_at DESC").
		Limit(limit).
		Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query app deployment history: %w", err)
	}
	return records, nil
}

// ---------- 25. GetAppDetail ----------

func (b *Bridge) GetAppDetail(ctx context.Context, appID string) (interface{}, error) {
	row := make(map[string]interface{})
	if err := b.DB.Table("apps").Where("id = ?", appID).Take(&row).Error; err != nil {
		return nil, fmt.Errorf("app not found: %w", err)
	}
	return row, nil
}

// ---------- 26. UpdateApp ----------

func (b *Bridge) UpdateApp(ctx context.Context, appID string, config map[string]interface{}) (interface{}, error) {
	if err := b.DB.Table("apps").Where("id = ?", appID).Updates(config).Error; err != nil {
		return nil, fmt.Errorf("failed to update app: %w", err)
	}

	row := make(map[string]interface{})
	if err := b.DB.Table("apps").Where("id = ?", appID).Take(&row).Error; err != nil {
		return map[string]interface{}{"status": "updated", "id": appID}, nil
	}
	return row, nil
}

// ---------- 29. SearchAppLogs ----------

func (b *Bridge) SearchAppLogs(ctx context.Context, appID, keyword string, limit int) (interface{}, error) {
	// Look up container name from app record
	row := make(map[string]interface{})
	if err := b.DB.Table("apps").Where("id = ?", appID).Take(&row).Error; err != nil {
		return nil, fmt.Errorf("app not found: %w", err)
	}

	containerName := toString(row["container_name"])
	if containerName == "" {
		containerName = toString(row["name"])
	}

	logs, err := b.d().GetContainerLogs(ctx, containerName, 2000)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	// Filter lines containing the keyword
	var matches []string
	lines := strings.Split(logs, "\n")
	for _, line := range lines {
		if strings.Contains(line, keyword) {
			matches = append(matches, line)
			if len(matches) >= limit {
				break
			}
		}
	}

	return map[string]interface{}{
		"app_id":         appID,
		"container":      containerName,
		"keyword":        keyword,
		"total_lines":    len(lines),
		"match_count":    len(matches),
		"limit":          limit,
		"matching_lines": matches,
	}, nil
}

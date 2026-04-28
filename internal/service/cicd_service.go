package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/provider/cicd"
)

// ---------- 45. TriggerCIBuild ----------

func (b *Bridge) TriggerCIBuild(ctx context.Context, providerType, repo, branch string) (interface{}, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}

	var provider model.Provider
	err := b.DB.Where("type = ? AND enabled = ?", "cicd-"+providerType, true).First(&provider).Error
	if err != nil {
		return map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("no enabled CI/CD provider found for type: %s", providerType),
		}, nil
	}

	var cfg struct {
		Token   string `json:"token"`
		Owner   string `json:"owner"`
		BaseURL string `json:"base_url"`
	}
	if err := json.Unmarshal([]byte(provider.Config), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse CI/CD provider config: %w", err)
	}

	switch providerType {
	case "github-actions":
		gh := cicd.NewGitHubActionsProvider(cfg.Token, cfg.Owner)
		runID, err := gh.TriggerBuild(ctx, repo, branch)
		if err != nil {
			return map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			}, nil
		}
		return map[string]interface{}{
			"status":   "triggered",
			"run_id":   runID,
			"repo":     repo,
			"branch":   branch,
			"provider": providerType,
		}, nil
	case "gitea":
		gt := cicd.NewGiteaActionsProvider(cfg.Token, cfg.Owner, cfg.BaseURL)
		runID, err := gt.TriggerBuild(ctx, repo, branch)
		if err != nil {
			return map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			}, nil
		}
		return map[string]interface{}{
			"status":   "triggered",
			"run_id":   runID,
			"repo":     repo,
			"branch":   branch,
			"provider": providerType,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported CI/CD provider type: %s", providerType)
	}
}
// ---------- 46. GetCIBuildStatus ----------

func (b *Bridge) GetCIBuildStatus(ctx context.Context, providerType, runID string) (interface{}, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}

	var provider model.Provider
	err := b.DB.Where("type = ? AND enabled = ?", "cicd-"+providerType, true).First(&provider).Error
	if err != nil {
		return map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("no enabled CI/CD provider found for type: %s", providerType),
		}, nil
	}

	var cfg struct {
		Token   string `json:"token"`
		Owner   string `json:"owner"`
		BaseURL string `json:"base_url"`
	}
	if err := json.Unmarshal([]byte(provider.Config), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse CI/CD provider config: %w", err)
	}

	switch providerType {
	case "github-actions":
		gh := cicd.NewGitHubActionsProvider(cfg.Token, cfg.Owner)
		status, err := gh.GetBuildStatus(ctx, runID)
		if err != nil {
			return map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			}, nil
		}
		return map[string]interface{}{
			"status":   "success",
			"build":    status,
			"provider": providerType,
		}, nil
	case "gitea":
		gt := cicd.NewGiteaActionsProvider(cfg.Token, cfg.Owner, cfg.BaseURL)
		status, err := gt.GetBuildStatus(ctx, runID)
		if err != nil {
			return map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			}, nil
		}
		return map[string]interface{}{
			"status":   "success",
			"build":    status,
			"provider": providerType,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported CI/CD provider type: %s", providerType)
	}
}

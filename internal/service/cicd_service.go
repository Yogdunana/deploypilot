package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/plugin"
	"github.com/Yogdunana/deploypilot/internal/provider/cicd"
)

// errCICDProviderNotFound indicates the CI/CD provider type is not supported or not configured.
// This is a client error (should return 200 with error in body), not a server error (500).
var errCICDProviderNotFound = errors.New("CI/CD provider not found")

// getCICDProvider resolves a CI/CD provider via the plugin registry.
func (b *Bridge) getCICDProvider(ctx context.Context, providerType string) (cicd.CICDProvider, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	var provider model.Provider
	err := b.DB.Where("type = ? AND enabled = ?", "cicd-"+providerType, true).First(&provider).Error
	if err != nil {
		return nil, fmt.Errorf("%w: no enabled CI/CD provider found for type: %s", errCICDProviderNotFound, providerType)
	}

	typeMap := map[string]string{
		"github-actions": "github_actions",
		"gitea":          "gitea_actions",
	}
	pluginType, ok := typeMap[providerType]
	if !ok {
		return nil, fmt.Errorf("%w: unsupported CI/CD provider type: %s", errCICDProviderNotFound, providerType)
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Config), &config); err != nil {
		return nil, fmt.Errorf("failed to parse CI/CD provider config: %w", err)
	}

	desc, ok := plugin.Global().GetDescriptor("cicd", pluginType)
	if !ok {
		return nil, fmt.Errorf("%w: no plugin registered for cicd:%s", errCICDProviderNotFound, pluginType)
	}
	instance, err := plugin.Global().CreateInstance(fmt.Sprintf("cicd-%s", provider.ID), desc, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create CI/CD provider: %w", err)
	}
	cicdProvider, ok := instance.(cicd.CICDProvider)
	if !ok {
		return nil, fmt.Errorf("plugin cicd:%s does not implement CICDProvider", pluginType)
	}
	return cicdProvider, nil
}

// ---------- 45. TriggerCIBuild ----------

func (b *Bridge) TriggerCIBuild(ctx context.Context, providerType, repo, branch string) (interface{}, error) {
	provider, err := b.getCICDProvider(ctx, providerType)
	if err != nil {
		if errors.Is(err, errCICDProviderNotFound) {
			return map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			}, nil
		}
		return nil, err
	}
	runID, err := provider.TriggerBuild(ctx, repo, branch)
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
}

// ---------- 46. GetCIBuildStatus ----------

func (b *Bridge) GetCIBuildStatus(ctx context.Context, providerType, runID string) (interface{}, error) {
	provider, err := b.getCICDProvider(ctx, providerType)
	if err != nil {
		if errors.Is(err, errCICDProviderNotFound) {
			return map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			}, nil
		}
		return nil, err
	}
	status, err := provider.GetBuildStatus(ctx, runID)
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
}

package service

import (
	"context"
	"fmt"

	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/plugin"
	registry "github.com/Yogdunana/deploypilot/internal/provider/registry"
)

// ---------- RegistryOps ----------

// RegistryOps handles registry operations (login, push, list_tags, ping).
func (b *Bridge) RegistryOps(registryID string, operation string, args map[string]interface{}) (interface{}, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}

	// Load registry from DB
	var reg model.Registry
	if err := b.DB.Where("id = ?", registryID).First(&reg).Error; err != nil {
		return nil, fmt.Errorf("registry not found: %w", err)
	}

	// Decrypt password
	var row struct {
		Password string `gorm:"column:password"`
	}
	if err := b.DB.Table("registries").Where("id = ?", registryID).Select("password").Take(&row).Error; err != nil {
		return nil, fmt.Errorf("failed to load registry credentials: %w", err)
	}
	plainPassword := ""
	if b.EncryptionKey != nil && row.Password != "" {
		if decrypted, err := crypto.Decrypt(b.EncryptionKey, row.Password); err == nil {
			plainPassword = decrypted
		}
	}

	// Allow args to override registry fields (for inline auth)
	regURL := reg.URL
	regUser := reg.Username
	regPass := plainPassword
	if args != nil {
		if v, ok := args["registry_url"].(string); ok && v != "" {
			regURL = v
		}
		if v, ok := args["username"].(string); ok && v != "" {
			regUser = v
		}
		if v, ok := args["password"].(string); ok && v != "" {
			regPass = v
		}
	}

	// Create registry provider via plugin registry
	config := map[string]interface{}{
		"url":      regURL,
		"username": regUser,
		"password": regPass,
	}
	desc, ok := plugin.Global().GetDescriptor("registry", reg.Provider)
	if !ok {
		return nil, fmt.Errorf("no plugin registered for registry:%s", reg.Provider)
	}
	instance, err := plugin.Global().CreateInstance(fmt.Sprintf("registry-%s", registryID), desc, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry provider: %w", err)
	}
	provider, ok := instance.(registry.RegistryProvider)
	if !ok {
		return nil, fmt.Errorf("plugin registry:%s does not implement RegistryProvider", reg.Provider)
	}

	ctx := context.Background()

	switch operation {
	case "login":
		if err := provider.Login(ctx); err != nil {
			return nil, fmt.Errorf("registry login failed: %w", err)
		}
		return map[string]interface{}{
			"status":  "success",
			"message": fmt.Sprintf("successfully authenticated with registry %s", reg.Name),
			"registry_id": reg.ID,
		}, nil

	case "push":
		localImage, _ := args["local_image"].(string)
		remoteTag, _ := args["remote_tag"].(string)
		if localImage == "" {
			return nil, fmt.Errorf("local_image is required")
		}
		if err := provider.Push(ctx, localImage, remoteTag); err != nil {
			return nil, fmt.Errorf("push failed: %w", err)
		}
		result := map[string]interface{}{
			"status":      "success",
			"message":     "image pushed successfully",
			"local_image": localImage,
			"registry_id": reg.ID,
		}
		if remoteTag != "" {
			result["remote_tag"] = remoteTag
		}
		return result, nil

	case "list_tags":
		repo, _ := args["repository"].(string)
		if repo == "" {
			return nil, fmt.Errorf("repository is required")
		}
		tags, err := provider.ListTags(ctx, repo)
		if err != nil {
			return nil, fmt.Errorf("list tags failed: %w", err)
		}
		return map[string]interface{}{
			"status":     "success",
			"repository": repo,
			"tags":       tags,
			"total":      len(tags),
		}, nil

	case "ping":
		if err := provider.Ping(ctx); err != nil {
			return map[string]interface{}{
				"status":  "unreachable",
				"message": err.Error(),
				"registry_id": reg.ID,
			}, nil
		}
		return map[string]interface{}{
			"status":      "reachable",
			"message":     "registry is accessible",
			"registry_id": reg.ID,
		}, nil

	default:
		return nil, fmt.Errorf("unknown registry operation: %s", operation)
	}
}

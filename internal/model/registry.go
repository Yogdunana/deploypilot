package model

import (
	"fmt"
	"time"

	"github.com/Yogdunana/deploypilot/internal/crypto"
)

// Registry represents a container image registry configuration.
type Registry struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	TenantID  string    `gorm:"index" json:"tenant_id"`
	Name      string    `gorm:"not null" json:"name"`
	Provider  string    `gorm:"not null" json:"provider"` // docker_hub, ghcr, harbor, acr
	URL       string    `gorm:"not null" json:"url"`      // registry URL (e.g., https://registry.hub.docker.com/v2/)
	Username  string    `json:"username"`
	Password  string    `gorm:"-" json:"-"`               // never expose in JSON; stored encrypted in DB
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Registry) TableName() string { return "registries" }

// registryRow is the DB representation with encrypted password.
type registryRow struct {
	ID        string    `gorm:"primaryKey"`
	TenantID  string    `gorm:"index"`
	Name      string    `gorm:"not null"`
	Provider  string    `gorm:"not null"`
	URL       string    `gorm:"not null"`
	Username  string
	Password  string // encrypted
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (registryRow) TableName() string { return "registries" }

// CreateRegistry creates a new registry with encrypted password.
func CreateRegistry(tenantID, name, provider, url, username, password string) (*Registry, error) {
	encrypted, err := crypto.Encrypt(getEncKey(), password)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt registry password: %w", err)
	}

	row := &registryRow{
		ID:       "reg-" + generateUUID(),
		TenantID: tenantID,
		Name:     name,
		Provider: provider,
		URL:      url,
		Username: username,
		Password: encrypted,
	}

	result := getDB().Create(row)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create registry: %w", result.Error)
	}

	return &Registry{
		ID:        row.ID,
		TenantID:  row.TenantID,
		Name:      row.Name,
		Provider:  row.Provider,
		URL:       row.URL,
		Username:  row.Username,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

// GetRegistry retrieves a registry by ID and decrypts its password.
func GetRegistry(id string) (*Registry, error) {
	var row registryRow
	result := getDB().First(&row, "id = ?", id)
	if result.Error != nil {
		return nil, fmt.Errorf("registry not found: %w", result.Error)
	}

	plainPassword, err := crypto.Decrypt(getEncKey(), row.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt registry password: %w", err)
	}

	return &Registry{
		ID:        row.ID,
		TenantID:  row.TenantID,
		Name:      row.Name,
		Provider:  row.Provider,
		URL:       row.URL,
		Username:  row.Username,
		Password:  plainPassword,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

// ListRegistries returns all registries for a tenant (without decrypted passwords).
func ListRegistries(tenantID string) ([]Registry, error) {
	var rows []registryRow
	result := getDB().Where("tenant_id = ?", tenantID).Find(&rows)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list registries: %w", result.Error)
	}

	registries := make([]Registry, 0, len(rows))
	for _, row := range rows {
		registries = append(registries, Registry{
			ID:        row.ID,
			TenantID:  row.TenantID,
			Name:      row.Name,
			Provider:  row.Provider,
			URL:       row.URL,
			Username:  row.Username,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return registries, nil
}

// UpdateRegistry updates a registry's fields. If password is non-empty, it is re-encrypted.
func UpdateRegistry(id, name, provider, url, username, password string) (*Registry, error) {
	updates := map[string]interface{}{}

	if name != "" {
		updates["name"] = name
	}
	if provider != "" {
		updates["provider"] = provider
	}
	if url != "" {
		updates["url"] = url
	}
	if username != "" {
		updates["username"] = username
	}
	if password != "" {
		encrypted, err := crypto.Encrypt(getEncKey(), password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt registry password: %w", err)
		}
		updates["password"] = encrypted
	}

	if len(updates) == 0 {
		return GetRegistry(id)
	}

	result := getDB().Model(&registryRow{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update registry: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("registry not found")
	}

	return GetRegistry(id)
}

// DeleteRegistry removes a registry by ID.
func DeleteRegistry(id string) error {
	result := getDB().Delete(&registryRow{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete registry: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("registry not found")
	}
	return nil
}

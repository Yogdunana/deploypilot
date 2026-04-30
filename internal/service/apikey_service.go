package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// APIKeyService handles API key CRUD and validation.
type APIKeyService struct {
	DB *gorm.DB
}

// NewAPIKeyService creates a new APIKeyService.
func NewAPIKeyService(db *gorm.DB) *APIKeyService {
	return &APIKeyService{DB: db}
}

// Create generates a new API key. Returns the APIKey model and the raw key (shown only once).
func (s *APIKeyService) Create(ctx context.Context, userID, tenantID, name string, scopes []string, expiresInDays int) (*model.APIKey, string, error) {
	// Generate random key: dp_ + 32 hex chars = 35 chars total
	rawBytes := make([]byte, 16)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate key: %w", err)
	}
	rawKey := "dp_" + hex.EncodeToString(rawBytes)

	// Hash the key for storage
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	// Prefix for identification (first 10 chars)
	prefix := rawKey[:10]
	if len(prefix) > 10 {
		prefix = prefix[:10]
	}

	// Serialize scopes
	scopesJSON, err := json.Marshal(scopes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal scopes: %w", err)
	}

	apiKey := &model.APIKey{
		ID:        generateAPIKeyID(),
		TenantID:  tenantID,
		UserID:    userID,
		Name:      name,
		KeyHash:   keyHash,
		KeyPrefix: prefix,
		Scopes:    string(scopesJSON),
	}

	if expiresInDays > 0 {
		expires := time.Now().AddDate(0, 0, expiresInDays)
		apiKey.ExpiresAt = &expires
	}

	if err := s.DB.WithContext(ctx).Create(apiKey).Error; err != nil {
		return nil, "", fmt.Errorf("failed to create API key: %w", err)
	}

	slog.Info("API key created", "id", apiKey.ID, "prefix", prefix, "name", name, "user_id", userID)
	return apiKey, rawKey, nil
}

// Validate checks if a raw API key is valid and not expired.
// On success, it updates LastUsedAt and returns the APIKey.
func (s *APIKeyService) Validate(ctx context.Context, rawKey string) (*model.APIKey, error) {
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	var apiKey model.APIKey
	if err := s.DB.WithContext(ctx).Where("key_hash = ?", keyHash).First(&apiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid API key")
		}
		return nil, fmt.Errorf("failed to query API key: %w", err)
	}

	// Check expiration
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("API key expired")
	}

	// Update last used time (fire-and-forget)
	now := time.Now()
	s.DB.WithContext(ctx).Model(&apiKey).Update("last_used_at", now)

	return &apiKey, nil
}

// List returns all API keys for a given user.
func (s *APIKeyService) List(ctx context.Context, userID string) ([]model.APIKey, error) {
	var keys []model.APIKey
	if err := s.DB.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&keys).Error; err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	return keys, nil
}

// Delete removes an API key by ID. Only the owner can delete their own keys.
func (s *APIKeyService) Delete(ctx context.Context, keyID, userID string) error {
	result := s.DB.WithContext(ctx).
		Where("id = ? AND user_id = ?", keyID, userID).
		Delete(&model.APIKey{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete API key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return sql.ErrNoRows
	}
	slog.Info("API key deleted", "id", keyID, "user_id", userID)
	return nil
}

// generateAPIKeyID generates a random hex ID for API keys.
func generateAPIKeyID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ParseScopes parses the JSON scopes string into a slice.
func ParseScopes(scopesJSON string) []string {
	var scopes []string
	if err := json.Unmarshal([]byte(scopesJSON), &scopes); err != nil {
		return nil
	}
	return scopes
}

// HasScope checks if the given scopes contain the required scope.
func HasScope(scopes []string, required string) bool {
	for _, s := range scopes {
		if strings.EqualFold(s, required) || s == "admin" {
			return true
		}
	}
	return false
}


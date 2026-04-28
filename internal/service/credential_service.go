package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/google/uuid"
)

// ---------- 16. CreateCredential ----------

func (b *Bridge) CreateCredential(ctx context.Context, tenantID, name, credType, plainValue string) (interface{}, error) {
	id := uuid.New().String()

	// Encrypt the value before storing
	encrypted, err := crypto.Encrypt(b.EncryptionKey, plainValue)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credential: %w", err)
	}

	if err := b.DB.Table("credentials").Create(map[string]interface{}{
		"id":              id,
		"tenant_id":       tenantID,
		"name":            name,
		"type":            credType,
		"encrypted_value": encrypted,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	return map[string]interface{}{
		"id":        id,
		"tenant_id": tenantID,
		"name":      name,
		"type":      credType,
	}, nil
}

// ---------- 17. ListCredentials ----------

func (b *Bridge) ListCredentials(ctx context.Context, tenantID string) (interface{}, error) {
	var rows []map[string]interface{}
	if err := b.DB.Table("credentials").Where("tenant_id = ?", tenantID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}

	// Mask values before returning
	sanitized := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		entry := map[string]interface{}{
			"id":        toString(r["id"]),
			"tenant_id": toString(r["tenant_id"]),
			"name":      toString(r["name"]),
			"type":      toString(r["type"]),
		}
		sanitized = append(sanitized, entry)
	}
	return sanitized, nil
}

// ---------- 18. DeleteCredential ----------

func (b *Bridge) DeleteCredential(ctx context.Context, credID string) error {
	result := b.DB.Table("credentials").Where("id = ?", credID).Delete(nil)
	if result.Error != nil {
		return fmt.Errorf("failed to delete credential: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("credential %s not found", credID)
	}
	return nil
}

// ---------- 31. UpdateCredential ----------

func (b *Bridge) UpdateCredential(ctx context.Context, credID string, value string) (interface{}, error) {
	// Encrypt the new value before storing
	encrypted, err := crypto.Encrypt(b.EncryptionKey, value)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credential: %w", err)
	}

	if err := b.DB.Table("credentials").Where("id = ?", credID).Update("encrypted_value", encrypted).Error; err != nil {
		return nil, fmt.Errorf("failed to update credential: %w", err)
	}
	return map[string]interface{}{
		"id":      credID,
		"status":  "updated",
		"message": "credential value updated",
	}, nil
}

// ---------- 31b. CreateCredentialWithExpiry ----------

func (b *Bridge) CreateCredentialWithExpiry(ctx context.Context, tenantID, name, credType, plainValue string, expiresInDays int) (interface{}, error) {
	id := uuid.New().String()

	// Encrypt the value before storing
	encrypted, err := crypto.Encrypt(b.EncryptionKey, plainValue)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credential: %w", err)
	}

	createData := map[string]interface{}{
		"id":              id,
		"tenant_id":       tenantID,
		"name":            name,
		"type":            credType,
		"encrypted_value": encrypted,
		"rotation_days":   expiresInDays,
	}

	// Set expires_at if expiresInDays > 0
	if expiresInDays > 0 {
		expiresAt := time.Now().Add(time.Duration(expiresInDays) * 24 * time.Hour)
		createData["expires_at"] = expiresAt
	}

	if err := b.DB.Table("credentials").Create(createData).Error; err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	result := map[string]interface{}{
		"id":            id,
		"tenant_id":     tenantID,
		"name":          name,
		"type":          credType,
		"rotation_days": expiresInDays,
	}
	if expiresInDays > 0 {
		result["expires_at"] = time.Now().Add(time.Duration(expiresInDays) * 24 * time.Hour).Format(time.RFC3339)
	}
	return result, nil
}

// ---------- 31c. RotateCredential ----------

func (b *Bridge) RotateCredential(ctx context.Context, credID string, newPlainValue string) (interface{}, error) {
	// Check credential exists
	var row map[string]interface{}
	if err := b.DB.Table("credentials").Where("id = ?", credID).Take(&row).Error; err != nil {
		return nil, fmt.Errorf("credential not found: %w", err)
	}

	// Encrypt the new value
	encrypted, err := crypto.Encrypt(b.EncryptionKey, newPlainValue)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credential: %w", err)
	}

	now := time.Now()
	updates := map[string]interface{}{
		"encrypted_value": encrypted,
		"last_rotated":    now,
	}

	if err := b.DB.Table("credentials").Where("id = ?", credID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to rotate credential: %w", err)
	}

	return toString(row["name"]), nil
}

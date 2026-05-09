package model

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/Yogdunana/deploypilot/internal/crypto"
	"gorm.io/gorm"
)

// CredentialWithPlain is a Credential with the decrypted plain value.
type CredentialWithPlain struct {
	Credential
	PlainValue string `json:"plain_value" gorm:"-"`
}

// dbHolder stores the global DB instance for CRUD operations.
var dbHolder struct {
	db      *gorm.DB
	encKey  []byte
}

// InitDB initializes the database connection for model CRUD operations.
func InitDB(db *gorm.DB, encKey []byte) {
	dbHolder.db = db
	dbHolder.encKey = encKey
}

// getDB returns the current DB or panics if not initialized.
func getDB() *gorm.DB {
	if dbHolder.db == nil {
		panic("model: database not initialized, call InitDB() first")
	}
	return dbHolder.db
}

// getEncKey returns the encryption key or panics if not initialized.
func getEncKey() []byte {
	if len(dbHolder.encKey) == 0 {
		panic("model: encryption key not initialized, call InitDB() first")
	}
	return dbHolder.encKey
}

// CreateCredential creates a new credential with encrypted value.
func CreateCredential(encKey []byte, tenantID, name, credType, plainValue string) (*CredentialWithPlain, error) {
	encrypted, err := crypto.Encrypt(encKey, plainValue)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credential value: %w", err)
	}

	cred := &Credential{
		ID:             "cred-" + uuid.New().String()[:8],
		TenantID:       tenantID,
		Name:           name,
		Type:           credType,
		EncryptedValue: encrypted,
	}

	result := getDB().Create(cred)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create credential: %w", result.Error)
	}

	return &CredentialWithPlain{
		Credential: *cred,
		PlainValue: plainValue,
	}, nil
}

// GetCredential retrieves a credential by ID and decrypts its value.
func GetCredential(encKey []byte, id string) (*CredentialWithPlain, error) {
	var cred Credential
	result := getDB().First(&cred, "id = ?", id)
	if result.Error != nil {
		return nil, fmt.Errorf("credential not found: %w", result.Error)
	}

	plainValue, err := crypto.Decrypt(encKey, cred.EncryptedValue)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credential: %w", err)
	}

	return &CredentialWithPlain{
		Credential: cred,
		PlainValue: plainValue,
	}, nil
}

// ListCredentials returns all credentials for a tenant (without decrypted values).
func ListCredentials(tenantID string) ([]Credential, error) {
	var creds []Credential
	result := getDB().Where("tenant_id = ?", tenantID).Find(&creds)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", result.Error)
	}
	return creds, nil
}

// DeleteCredential removes a credential by ID.
func DeleteCredential(id string) error {
	result := getDB().Delete(&Credential{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete credential: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("credential not found")
	}
	return nil
}

// UpdateCredential updates a credential's encrypted value.
func UpdateCredential(encKey []byte, id, newPlainValue string) (*CredentialWithPlain, error) {
	encrypted, err := crypto.Encrypt(encKey, newPlainValue)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credential value: %w", err)
	}

	result := getDB().Model(&Credential{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"encrypted_value": encrypted,
		})
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update credential: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("credential not found")
	}

	return GetCredential(encKey, id)
}

// generateUUID creates a short random UUID string.
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// ---------- Credential Lifecycle Methods ----------

// IsExpired returns true if the credential's ExpiresAt is set and before time.Now().
func IsExpired(cred *Credential) bool {
	if cred.ExpiresAt == nil {
		return false
	}
	return cred.ExpiresAt.Before(time.Now())
}

// DaysUntilExpiry returns the number of days until the credential expires.
// Returns -1 if the credential never expires (ExpiresAt is nil).
func DaysUntilExpiry(cred *Credential) int {
	if cred.ExpiresAt == nil {
		return -1
	}
	duration := time.Until(*cred.ExpiresAt)
	days := int(duration.Hours() / 24)
	return days
}

// ListExpiringCredentials finds credentials that will expire within the given duration.
func ListExpiringCredentials(db *gorm.DB, within time.Duration) ([]Credential, error) {
	var creds []Credential
	threshold := time.Now().Add(within)
	result := db.Where("expires_at IS NOT NULL AND expires_at <= ?", threshold).Find(&creds)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list expiring credentials: %w", result.Error)
	}
	return creds, nil
}

// RotateCredential encrypts a new value, updates the credential's encrypted_value
// and LastRotated to now, while preserving the existing ExpiresAt.
func RotateCredential(db *gorm.DB, encKey []byte, id string, newPlainValue string) (*Credential, error) {
	encrypted, err := crypto.Encrypt(encKey, newPlainValue)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credential value: %w", err)
	}

	now := time.Now()
	result := db.Model(&Credential{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"encrypted_value": encrypted,
			"last_rotated":    now,
		})
	if result.Error != nil {
		return nil, fmt.Errorf("failed to rotate credential: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("credential not found")
	}

	var cred Credential
	if err := db.Where("id = ?", id).First(&cred).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve credential after rotation: %w", err)
	}
	return &cred, nil
}

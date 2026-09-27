package api

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/signing"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// globalSigningAPI is the package-level SigningAPI instance.
var globalSigningAPI *SigningAPI

// SigningAPI handles code signing HTTP endpoints.
type SigningAPI struct {
	db *gorm.DB
}

// NewSigningAPI creates a new SigningAPI.
func NewSigningAPI(db *gorm.DB) *SigningAPI {
	return &SigningAPI{db: db}
}

// SetSigningAPI sets the global SigningAPI instance.
func SetSigningAPI(s *SigningAPI) {
	globalSigningAPI = s
}

// GetGlobalSigningAPI returns the global SigningAPI instance.
func GetGlobalSigningAPI() *SigningAPI {
	return globalSigningAPI
}

// GetSigningStatus returns the current signing key status.
// GET /api/v1/security/signing/status
func GetSigningStatus(c *gin.Context) {
	if globalSigningAPI == nil {
		respondError(c, http.StatusInternalServerError, "signing service not initialized")
		return
	}

	var activeKey model.SigningKey
	result := globalSigningAPI.db.Where("is_active = ?", true).Order("key_version DESC").First(&activeKey)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			respondSuccess(c, gin.H{
				"initialized": false,
				"fingerprint": "",
				"key_version": 0,
				"verified":    false,
			})
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to query signing key status")
		return
	}

	respondSuccess(c, gin.H{
		"initialized": true,
		"fingerprint": activeKey.Fingerprint,
		"key_version": activeKey.KeyVersion,
		"verified":    false,
	})
}

// VerifySignature verifies the current running binary's signature.
// POST /api/v1/security/signing/verify
func VerifySignature(c *gin.Context) {
	if globalSigningAPI == nil {
		respondError(c, http.StatusInternalServerError, "signing service not initialized")
		return
	}

	var activeKey model.SigningKey
	result := globalSigningAPI.db.Where("is_active = ?", true).Order("key_version DESC").First(&activeKey)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			respondError(c, http.StatusNotFound, "no active signing key found")
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to query signing key")
		return
	}

	signer, err := globalSigningAPI.loadSignerFromModel(&activeKey)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to load signing key")
		return
	}

	verified, err := signing.VerifySelf("", ed25519.PublicKey(signer.PublicKeyBytes()))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to verify binary")
		return
	}

	respondSuccess(c, gin.H{
		"verified":    verified,
		"fingerprint": activeKey.Fingerprint,
		"key_version": activeKey.KeyVersion,
	})
}

// GenerateKeys generates a new Ed25519 key pair and stores it.
// POST /api/v1/security/signing/keys/generate (admin only)
func GenerateKeys(c *gin.Context) {
	if globalSigningAPI == nil {
		respondError(c, http.StatusInternalServerError, "signing service not initialized")
		return
	}

	userID, exists := c.Get(string(auth.UserIDKey))
	if !exists {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		respondErrori18n(c, http.StatusInternalServerError, "error.auth.invalid_user_id")
		return
	}

	publicKey, privateKey, err := signing.GenerateKeyPair()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to generate key pair")
		return
	}

	signer := signing.NewSigner(publicKey, privateKey, 0)
	fingerprint := signer.Fingerprint()

	// Determine next key version
	var maxVersion struct {
		MaxVersion int
	}
	globalSigningAPI.db.Raw("SELECT COALESCE(MAX(key_version), 0) as max_version FROM signing_keys").Scan(&maxVersion)
	nextVersion := maxVersion.MaxVersion + 1

	sealedPrivateKey, err := encryptPrivateKey(base64.StdEncoding.EncodeToString(privateKey.Seed()))
	if err != nil {
		slog.Error("failed to encrypt signing private key", "error", err)
		respondError(c, http.StatusInternalServerError, "failed to seal signing key")
		return
	}

	keyRecord := model.SigningKey{
		ID:          uuid.New().String(),
		KeyVersion:  nextVersion,
		PublicKey:   base64.StdEncoding.EncodeToString(publicKey),
		PrivateKey:  sealedPrivateKey,
		Fingerprint: fingerprint,
		IsActive:    true,
		CreatedBy:   userIDStr,
	}

	// Deactivate all existing keys
	if err := globalSigningAPI.db.Model(&model.SigningKey{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to deactivate old keys")
		return
	}

	if err := globalSigningAPI.db.Create(&keyRecord).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to save signing key")
		return
	}

	respondSuccess(c, gin.H{
		"id":          keyRecord.ID,
		"key_version": keyRecord.KeyVersion,
		"fingerprint": keyRecord.Fingerprint,
		"is_active":   keyRecord.IsActive,
		"created_by":  keyRecord.CreatedBy,
	})
}

// RotateKeys generates a new key pair and keeps old keys for verification.
// POST /api/v1/security/signing/keys/rotate (admin only)
func RotateKeys(c *gin.Context) {
	if globalSigningAPI == nil {
		respondError(c, http.StatusInternalServerError, "signing service not initialized")
		return
	}

	userID, exists := c.Get(string(auth.UserIDKey))
	if !exists {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		respondErrori18n(c, http.StatusInternalServerError, "error.auth.invalid_user_id")
		return
	}

	publicKey, privateKey, err := signing.GenerateKeyPair()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to generate key pair")
		return
	}

	signer := signing.NewSigner(publicKey, privateKey, 0)
	fingerprint := signer.Fingerprint()

	// Determine next key version
	var maxVersion struct {
		MaxVersion int
	}
	globalSigningAPI.db.Raw("SELECT COALESCE(MAX(key_version), 0) as max_version FROM signing_keys").Scan(&maxVersion)
	nextVersion := maxVersion.MaxVersion + 1

	sealedPrivateKey, err := encryptPrivateKey(base64.StdEncoding.EncodeToString(privateKey.Seed()))
	if err != nil {
		slog.Error("failed to encrypt signing private key", "error", err)
		respondError(c, http.StatusInternalServerError, "failed to seal signing key")
		return
	}

	keyRecord := model.SigningKey{
		ID:          uuid.New().String(),
		KeyVersion:  nextVersion,
		PublicKey:   base64.StdEncoding.EncodeToString(publicKey),
		PrivateKey:  sealedPrivateKey,
		Fingerprint: fingerprint,
		IsActive:    true,
		CreatedBy:   userIDStr,
	}

	// Deactivate all existing keys (keep them in DB for verification)
	if err := globalSigningAPI.db.Model(&model.SigningKey{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to deactivate old keys")
		return
	}

	if err := globalSigningAPI.db.Create(&keyRecord).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to save new signing key")
		return
	}

	// Count total keys (including inactive ones kept for verification)
	var totalKeys int64
	globalSigningAPI.db.Model(&model.SigningKey{}).Count(&totalKeys)

	respondSuccess(c, gin.H{
		"id":          keyRecord.ID,
		"key_version": keyRecord.KeyVersion,
		"fingerprint": keyRecord.Fingerprint,
		"is_active":   keyRecord.IsActive,
		"created_by":  keyRecord.CreatedBy,
		"total_keys":  totalKeys,
	})
}

// encPrefix marks a value that is stored encrypted. Anything without the prefix is
// a legacy plaintext row written before signing keys were encrypted at rest.
const encPrefix = "enc:"

// encryptPrivateKey encrypts an Ed25519 seed (base64) before it hits the database.
// The private key signs every release artifact, so a leaked database dump must not
// hand out a working signing key along with it.
func encryptPrivateKey(plain string) (string, error) {
	key := encryptionKey()
	if len(key) == 0 {
		return "", fmt.Errorf("encryption key not initialized")
	}
	sealed, err := crypto.Encrypt(key, plain)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt private key: %w", err)
	}
	return encPrefix + sealed, nil
}

// decryptPrivateKey reverses encryptPrivateKey. Values written before encryption
// was introduced have no prefix and are returned as-is so existing keys keep
// working after upgrade.
func decryptPrivateKey(stored string) (string, error) {
	if !strings.HasPrefix(stored, encPrefix) {
		slog.Warn("signing key stored in plaintext; rotate it to encrypt at rest",
			"hint", "POST /api/v1/security/signing/keys/rotate")
		return stored, nil
	}
	key := encryptionKey()
	if len(key) == 0 {
		return "", fmt.Errorf("encryption key not initialized")
	}
	plain, err := crypto.Decrypt(key, strings.TrimPrefix(stored, encPrefix))
	if err != nil {
		return "", fmt.Errorf("failed to decrypt private key: %w", err)
	}
	return plain, nil
}

// loadSignerFromModel reconstructs a Signer from a SigningKey database record.
func (a *SigningAPI) loadSignerFromModel(key *model.SigningKey) (*signing.Signer, error) {
	pubBytes, err := base64.StdEncoding.DecodeString(key.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	privSeedB64, err := decryptPrivateKey(key.PrivateKey)
	if err != nil {
		return nil, err
	}

	privSeedBytes, err := base64.StdEncoding.DecodeString(privSeedB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(pubBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: got %d, want %d", len(pubBytes), ed25519.PublicKeySize)
	}
	if len(privSeedBytes) != ed25519.SeedSize {
		return nil, fmt.Errorf("invalid private key seed size: got %d, want %d", len(privSeedBytes), ed25519.SeedSize)
	}

	privateKey := ed25519.NewKeyFromSeed(privSeedBytes)
	return signing.NewSigner(pubBytes, privateKey, key.KeyVersion), nil
}

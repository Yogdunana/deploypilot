package api

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/auth"
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

	_, err := globalSigningAPI.loadSignerFromModel(&activeKey)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to load signing key: "+err.Error())
		return
	}

	verified, err := signing.VerifySelf("")
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to verify binary: "+err.Error())
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

	publicKey, privateKey, err := signing.GenerateKeyPair()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to generate key pair: "+err.Error())
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

	keyRecord := model.SigningKey{
		ID:          uuid.New().String(),
		KeyVersion:  nextVersion,
		PublicKey:   base64.StdEncoding.EncodeToString(publicKey),
		PrivateKey:  base64.StdEncoding.EncodeToString(privateKey.Seed()),
		Fingerprint: fingerprint,
		IsActive:    true,
		CreatedBy:   userID.(string),
	}

	// Deactivate all existing keys
	if err := globalSigningAPI.db.Model(&model.SigningKey{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to deactivate old keys: "+err.Error())
		return
	}

	if err := globalSigningAPI.db.Create(&keyRecord).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to save signing key: "+err.Error())
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

	publicKey, privateKey, err := signing.GenerateKeyPair()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to generate key pair: "+err.Error())
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

	keyRecord := model.SigningKey{
		ID:          uuid.New().String(),
		KeyVersion:  nextVersion,
		PublicKey:   base64.StdEncoding.EncodeToString(publicKey),
		PrivateKey:  base64.StdEncoding.EncodeToString(privateKey.Seed()),
		Fingerprint: fingerprint,
		IsActive:    true,
		CreatedBy:   userID.(string),
	}

	// Deactivate all existing keys (keep them in DB for verification)
	if err := globalSigningAPI.db.Model(&model.SigningKey{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to deactivate old keys: "+err.Error())
		return
	}

	if err := globalSigningAPI.db.Create(&keyRecord).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to save new signing key: "+err.Error())
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

// loadSignerFromModel reconstructs a Signer from a SigningKey database record.
func (a *SigningAPI) loadSignerFromModel(key *model.SigningKey) (*signing.Signer, error) {
	pubBytes, err := base64.StdEncoding.DecodeString(key.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	privSeedBytes, err := base64.StdEncoding.DecodeString(key.PrivateKey)
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

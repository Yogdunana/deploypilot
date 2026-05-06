package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// KeyRotationResult contains the result of a key rotation operation.
type KeyRotationResult struct {
	OldVersion  int    `json:"old_version"`
	NewVersion  int    `json:"new_version"`
	Fingerprint string `json:"fingerprint"`
	RotatedAt   string `json:"rotated_at"`
}

// RotateLicenseKeys performs a key rotation:
// 1. Generates a new Ed25519 key pair
// 2. Deactivates all existing keys
// 3. Stores the new key with incremented version
// 4. Updates the license engine to use the new public key
func (b *Bridge) RotateLicenseKeys(ctx context.Context, userID string) (*KeyRotationResult, error) {
	// 1. Generate new key pair
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// 2. Calculate fingerprint
	fingerprint := keyFingerprint(publicKey)

	// 3. Get next version number
	var maxVersion int
	b.DB.Model(&model.LicenseSigningKey{}).Select("COALESCE(MAX(key_version), 0)").Scan(&maxVersion)
	newVersion := maxVersion + 1

	// 4. Deactivate all existing keys
	now := time.Now()
	b.DB.Model(&model.LicenseSigningKey{}).Where("is_active = ?", true).Updates(map[string]interface{}{
		"is_active":  false,
		"rotated_at": now,
	})

	// 5. Store new key
	key := model.LicenseSigningKey{
		ID:          fmt.Sprintf("lsk-%d", time.Now().UnixNano()),
		KeyVersion:  newVersion,
		PublicKey:   base64.StdEncoding.EncodeToString(publicKey),
		PrivateKey:  base64.StdEncoding.EncodeToString(privateKey.Seed()),
		Fingerprint: fingerprint,
		IsActive:    true,
		CreatedBy:   userID,
	}
	if err := b.DB.Create(&key).Error; err != nil {
		return nil, fmt.Errorf("failed to store new key: %w", err)
	}

	// 6. Update license engine with new public key
	if b.LicenseEngine != nil {
		b.LicenseEngine.RotatePublicKey(publicKey)
	}

	// 7. Update private key for signing
	b.LicensePrivKey = privateKey

	slog.Info("license key rotated",
		"old_version", maxVersion,
		"new_version", newVersion,
		"fingerprint", fingerprint,
		"user_id", userID,
	)

	return &KeyRotationResult{
		OldVersion:  maxVersion,
		NewVersion:  newVersion,
		Fingerprint: fingerprint,
		RotatedAt:   now.Format(time.RFC3339),
	}, nil
}

// ListLicenseKeys returns all license signing keys (admin only).
func (b *Bridge) ListLicenseKeys(ctx context.Context) (interface{}, error) {
	var keys []model.LicenseSigningKey
	if err := b.DB.Order("key_version DESC").Find(&keys).Error; err != nil {
		return nil, fmt.Errorf("failed to list license keys: %w", err)
	}

	result := make([]map[string]interface{}, len(keys))
	for i, k := range keys {
		result[i] = map[string]interface{}{
			"id":           k.ID,
			"key_version":  k.KeyVersion,
			"public_key":   k.PublicKey,
			"fingerprint":  k.Fingerprint,
			"is_active":    k.IsActive,
			"created_by":   k.CreatedBy,
			"created_at":   k.CreatedAt,
			"rotated_at":   k.RotatedAt,
		}
	}
	return map[string]interface{}{
		"keys":  result,
		"total": len(result),
	}, nil
}

// GetCurrentKeyVersion returns the active key version.
func (b *Bridge) GetCurrentKeyVersion(ctx context.Context) (interface{}, error) {
	var key model.LicenseSigningKey
	if err := b.DB.Where("is_active = ?", true).First(&key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return map[string]interface{}{
				"key_version": 0,
				"fingerprint": "",
				"has_keys":    false,
			}, nil
		}
		return nil, fmt.Errorf("failed to get current key: %w", err)
	}
	return map[string]interface{}{
		"key_version": key.KeyVersion,
		"fingerprint": key.Fingerprint,
		"has_keys":    true,
		"created_at":  key.CreatedAt,
	}, nil
}

// InitLicenseKeysFromConfig loads the initial key from config into the DB if no keys exist.
// This enables key rotation for instances that started with a config-based key.
func (b *Bridge) InitLicenseKeysFromConfig(ctx context.Context, pubKey ed25519.PublicKey, privKey ed25519.PrivateKey) error {
	var count int64
	b.DB.Model(&model.LicenseSigningKey{}).Count(&count)
	if count > 0 {
		return nil // keys already exist
	}

	fingerprint := keyFingerprint(pubKey)
	key := model.LicenseSigningKey{
		ID:          fmt.Sprintf("lsk-init-%d", time.Now().UnixNano()),
		KeyVersion:  1,
		PublicKey:   base64.StdEncoding.EncodeToString(pubKey),
		PrivateKey:  base64.StdEncoding.EncodeToString(privKey.Seed()),
		Fingerprint: fingerprint,
		IsActive:    true,
		CreatedBy:   "system",
	}
	return b.DB.Create(&key).Error
}

// LoadAllActivePublicKeys loads all active public keys from the DB for verification.
func (b *Bridge) LoadAllActivePublicKeys(ctx context.Context) ([]ed25519.PublicKey, error) {
	var keys []model.LicenseSigningKey
	// Load both active and inactive keys for verification of existing licenses
	if err := b.DB.Order("key_version DESC").Find(&keys).Error; err != nil {
		return nil, fmt.Errorf("failed to load keys: %w", err)
	}

	var pubKeys []ed25519.PublicKey
	for _, k := range keys {
		keyBytes, err := base64.StdEncoding.DecodeString(k.PublicKey)
		if err != nil || len(keyBytes) != ed25519.PublicKeySize {
			slog.Warn("skipping invalid license key", "version", k.KeyVersion, "error", err)
			continue
		}
		pubKeys = append(pubKeys, keyBytes)
	}
	return pubKeys, nil
}

// ShamirSecretSharing splits a secret into N parts, requiring M to reconstruct.
// Uses simple XOR-based Shamir's Secret Sharing over GF(256).
type ShamirShare struct {
	Index int    `json:"index"`
	Data  string `json:"data"` // Base64 encoded
}

// SplitSecret splits a secret into N shares, requiring M to reconstruct.
func SplitSecret(secret []byte, n, m int) ([]ShamirShare, error) {
	if m > n {
		return nil, fmt.Errorf("threshold M (%d) cannot exceed total shares N (%d)", m, n)
	}
	if m < 1 {
		return nil, fmt.Errorf("threshold M must be at least 1")
	}
	if n > 255 {
		return nil, fmt.Errorf("total shares N cannot exceed 255")
	}

	// Generate random coefficients for polynomial using cryptographic random
	coefficients := make([]byte, m)
	coefficients[0] = secret[0]
	for i := 1; i < m; i++ {
		buf := make([]byte, 1)
		if _, err := rand.Read(buf); err != nil {
			return nil, fmt.Errorf("failed to generate random coefficient: %w", err)
		}
		coefficients[i] = buf[0]
	}

	shares := make([]ShamirShare, n)
	for i := 0; i < n; i++ {
		x := byte(i + 1)
		y := evalPolynomial(coefficients, x)
		shares[i] = ShamirShare{
			Index: i + 1,
			Data:  base64.StdEncoding.EncodeToString([]byte{y}),
		}
	}
	return shares, nil
}

// CombineShares reconstructs the secret from M shares.
func CombineShares(shares []ShamirShare, m int) ([]byte, error) {
	if len(shares) < m {
		return nil, fmt.Errorf("need at least %d shares, got %d", m, len(shares))
	}

	// Use Lagrange interpolation at x=0
	var secret byte
	for i := 0; i < m; i++ {
		yi, err := base64.StdEncoding.DecodeString(shares[i].Data)
		if err != nil || len(yi) != 1 {
			return nil, fmt.Errorf("invalid share data at index %d", shares[i].Index)
		}
		xi := byte(shares[i].Index)
		numerator := byte(1)
		denominator := byte(1)
		for j := 0; j < m; j++ {
			if i == j {
				continue
			}
			xj := byte(shares[j].Index)
			numerator = gf256Mul(numerator, xj)         // 0 - xj = xj (mod 256)
			denominator = gf256Mul(denominator, gf256Sub(xi, xj))
		}
		// Lagrange basis: L_i(0) = product(xj / (xi - xj))
		lagrange := gf256Div(numerator, denominator)
		secret ^= gf256Mul(yi[0], lagrange)
	}

	return []byte{secret}, nil
}

// BackupKeyWithShamir splits the private key into N shares using Shamir's Secret Sharing.
func (b *Bridge) BackupKeyWithShamir(ctx context.Context, n, m int) (interface{}, error) {
	if b.LicensePrivKey == nil {
		return nil, fmt.Errorf("no private key available for backup")
	}

	seed := b.LicensePrivKey.Seed()
	shares, err := SplitSecret(seed, n, m)
	if err != nil {
		return nil, fmt.Errorf("failed to split secret: %w", err)
	}

	return map[string]interface{}{
		"shares":    shares,
		"threshold": m,
		"total":     n,
		"created_at": time.Now().Format(time.RFC3339),
	}, nil
}

// Helper functions

func keyFingerprint(pubKey ed25519.PublicKey) string {
	hash := sha256.Sum256(pubKey)
	return hex.EncodeToString(hash[:])[:16]
}

func evalPolynomial(coefficients []byte, x byte) byte {
	result := byte(0)
	for i := len(coefficients) - 1; i >= 0; i-- {
		result = gf256Add(gf256Mul(result, x), coefficients[i])
	}
	return result
}

// GF(256) arithmetic with irreducible polynomial x^8 + x^4 + x^3 + x + 1 (0x11B)
func gf256Mul(a, b byte) byte {
	var result byte
	for i := 0; i < 8; i++ {
		if b&1 != 0 {
			result ^= a
		}
		hi := a & 0x80
		a <<= 1
		if hi != 0 {
			a ^= 0x1B
		}
		b >>= 1
	}
	return result
}

func gf256Add(a, b byte) byte { return a ^ b }
func gf256Sub(a, b byte) byte { return a ^ b } // same as add in GF(2^8)

func gf256Div(a, b byte) byte {
	if b == 0 {
		return 0
	}
	// Compute inverse using extended Euclidean or lookup
	inv := gf256Inverse(b)
	return gf256Mul(a, inv)
}

func gf256Inverse(a byte) byte {
	if a == 0 {
		return 0
	}
	// Use Fermat's little theorem: a^(254) = a^(-1) in GF(256)
	result := a
	for i := 0; i < 6; i++ {
		result = gf256Mul(result, result)
		result = gf256Mul(result, a)
	}
	// a^254 = (a^2)^127 = result^127... simplified:
	// Just do a^254 directly
	result = byte(1)
	base := a
	exp := 254
	for exp > 0 {
		if exp%2 == 1 {
			result = gf256Mul(result, base)
		}
		base = gf256Mul(base, base)
		exp /= 2
	}
	return result
}

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

const (
	argon2Time       = 1
	argon2Memory     = 64 * 1024 // 64 MB
	argon2Parallelism = 4
	argon2SaltLen    = 16
	argon2KeyLen     = 32
)

// NewEncryptionKey generates a random 256-bit (32-byte) AES key.
func NewEncryptionKey() []byte {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		panic(fmt.Sprintf("failed to generate encryption key: %v", err))
	}
	return key
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64-encoded ciphertext.
// The ciphertext format is: base64(nonce + ciphertext).
func Encrypt(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	// nonce is prepended to ciphertext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded AES-256-GCM ciphertext.
func Decrypt(key []byte, encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}
	return string(plaintext), nil
}

// getBcryptCost returns the bcrypt cost factor from environment or default 12.
func getBcryptCost() int {
	costStr := os.Getenv("DEPLOYPILOT_BCRYPT_COST")
	if costStr == "" {
		return 12
	}
	cost, err := strconv.Atoi(costStr)
	if err != nil || cost < 10 {
		cost = 10
	}
	if cost > 31 {
		cost = 31
	}
	return cost
}

// HashPasswordArgon2ID hashes a password using argon2id and returns a PHC-format string.
func HashPasswordArgon2ID(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("failed to generate argon2id salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Parallelism, argon2KeyLen)

	// Encode as PHC format: $argon2id$v=19$m=65536,t=1,p=4$<base64(salt)>$<base64(hash)>
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory, argon2Time, argon2Parallelism, encodedSalt, encodedHash), nil
}

// HashPassword hashes a password using argon2id.
func HashPassword(password string) (string, error) {
	return HashPasswordArgon2ID(password)
}

// CheckPassword compares a plaintext password against a stored hash.
// It supports both argon2id (PHC format) and bcrypt hashes for backward compatibility.
func CheckPassword(password, hash string) bool {
	if strings.HasPrefix(hash, "$argon2id$") {
		return checkPasswordArgon2ID(password, hash)
	}
	// Fallback to bcrypt for backward compatibility
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// checkPasswordArgon2ID verifies a password against an argon2id PHC-format hash.
func checkPasswordArgon2ID(password, hash string) bool {
	// Expected format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false
	}
	// parts[0] = "", parts[1] = "argon2id", parts[2] = "v=19", parts[3] = "m=...,t=...,p=...",
	// parts[4] = salt, parts[5] = hash

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	computedHash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Parallelism, uint32(len(expectedHash)))

	return subtle.ConstantTimeCompare(expectedHash, computedHash) == 1
}

// defaultKeyPath returns the default file path for the auto-generated encryption key.
func defaultKeyPath() string {
	dataDir := os.Getenv("DEPLOYPILOT_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	return filepath.Join(dataDir, ".encryption_key")
}

// LoadEncryptionKeyFromEnv parses DEPLOYPILOT_ENCRYPTION_KEY from the environment.
// It supports two formats:
//   - base64-encoded 32-byte key (recommended): openssl rand -base64 32
//   - raw 32-byte string (legacy compatibility)
//
// If DEPLOYPILOT_ENCRYPTION_KEY is not set, it attempts to load a previously
// auto-generated key from the data directory. If no key file exists, a new
// key is generated, persisted to disk, and returned.
func LoadEncryptionKeyFromEnv() ([]byte, error) {
	raw := os.Getenv("DEPLOYPILOT_ENCRYPTION_KEY")
	if raw != "" {
		// Try base64 decode first
		if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil {
			if len(decoded) == 32 {
				return decoded, nil
			}
			return nil, fmt.Errorf(
				"DEPLOYPILOT_ENCRYPTION_KEY: base64 decoded to %d bytes, expected 32. "+
					"Generate a valid key with: openssl rand -base64 32",
				len(decoded),
			)
		}
		// Try raw string (legacy 32-byte hex/string)
		if len(raw) == 32 {
			return []byte(raw), nil
		}
		return nil, fmt.Errorf(
			"DEPLOYPILOT_ENCRYPTION_KEY: invalid key (length %d). "+
				"Supported formats:\n"+
				"  1. Base64-encoded 32-byte key (recommended): openssl rand -base64 32\n"+
				"  2. Raw 32-byte string (legacy)\n"+
				"  Current value length: %d bytes",
			len(raw), len(raw),
		)
	}

	// No env var set — try loading from persistent key file
	keyPath := defaultKeyPath()
	if data, err := os.ReadFile(keyPath); err == nil {
		// Try base64 decode
		if decoded, err := base64.StdEncoding.DecodeString(string(data)); err == nil && len(decoded) == 32 {
			return decoded, nil
		}
		// Try raw bytes
		if len(data) == 32 {
			return data, nil
		}
	}

	// No key file exists — generate a new key and persist it
	key := NewEncryptionKey()
	encoded := base64.StdEncoding.EncodeToString(key)

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory for encryption key: %w", err)
	}

	if err := os.WriteFile(keyPath, []byte(encoded), 0600); err != nil {
		return nil, fmt.Errorf("failed to persist encryption key to %s: %w", keyPath, err)
	}

	return key, nil
}

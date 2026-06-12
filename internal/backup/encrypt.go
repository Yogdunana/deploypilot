package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const (
	// Encryption header: "DPENC01" (7 bytes) + nonce (12 bytes) + ciphertext
	encryptionHeader = "DPENC01"
	headerSize       = 7
	nonceSize        = 12
)

// EncryptBackup encrypts data using AES-256-GCM.
// The output format is: header (7 bytes) + nonce (12 bytes) + ciphertext + tag.
// The key must be exactly 32 bytes (AES-256).
func EncryptBackup(key, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Prepend header + nonce
	result := make([]byte, 0, headerSize+nonceSize+len(ciphertext))
	result = append(result, []byte(encryptionHeader)...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// DecryptBackup decrypts data that was encrypted with EncryptBackup.
func DecryptBackup(key, data []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
	}

	if len(data) < headerSize+nonceSize {
		return nil, fmt.Errorf("encrypted data too short: %d bytes", len(data))
	}

	// Verify header
	header := string(data[:headerSize])
	if header != encryptionHeader {
		return nil, fmt.Errorf("invalid encryption header: expected %q, got %q", encryptionHeader, header)
	}

	nonce := data[headerSize : headerSize+nonceSize]
	ciphertext := data[headerSize+nonceSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// IsEncryptedBackup checks if data was encrypted by EncryptBackup.
func IsEncryptedBackup(data []byte) bool {
	if len(data) < headerSize {
		return false
	}
	return string(data[:headerSize]) == encryptionHeader
}

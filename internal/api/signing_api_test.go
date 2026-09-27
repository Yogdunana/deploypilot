package api

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestSigningPrivateKeyAtRest(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes
	original := encryptionKey()
	SetEncryptionKey(key)
	defer SetEncryptionKey(original)

	seed := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))

	sealed, err := encryptPrivateKey(seed)
	if err != nil {
		t.Fatalf("encryptPrivateKey() error = %v", err)
	}
	if !strings.HasPrefix(sealed, "enc:") {
		t.Errorf("encryptPrivateKey() = %q, want enc: prefix", sealed)
	}
	if strings.Contains(sealed, seed) {
		t.Errorf("encryptPrivateKey() leaked the plaintext seed: %q", sealed)
	}

	plain, err := decryptPrivateKey(sealed)
	if err != nil {
		t.Fatalf("decryptPrivateKey() error = %v", err)
	}
	if plain != seed {
		t.Errorf("round trip = %q, want %q", plain, seed)
	}
}

// Rows written before encryption existed have no prefix and must still load,
// otherwise every existing key becomes unusable on upgrade.
func TestDecryptPrivateKey_LegacyPlaintext(t *testing.T) {
	original := encryptionKey()
	SetEncryptionKey([]byte("0123456789abcdef0123456789abcdef"))
	defer SetEncryptionKey(original)

	legacy := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	plain, err := decryptPrivateKey(legacy)
	if err != nil {
		t.Fatalf("decryptPrivateKey(legacy) error = %v", err)
	}
	if plain != legacy {
		t.Errorf("decryptPrivateKey(legacy) = %q, want %q", plain, legacy)
	}
}

// Without a key configured the seed must never be written in the clear.
func TestEncryptPrivateKey_NoKeyConfigured(t *testing.T) {
	original := encryptionKey()
	SetEncryptionKey(nil)
	defer SetEncryptionKey(original)

	if _, err := encryptPrivateKey(base64.StdEncoding.EncodeToString([]byte("seed"))); err == nil {
		t.Error("encryptPrivateKey() = nil error, want failure when no key is configured")
	}
}

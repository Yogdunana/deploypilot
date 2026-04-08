package crypto

import (
	"encoding/base64"
	"strings"
	"testing"
)

// ========== AES-256-GCM ==========

func TestEncryptDecrypt(t *testing.T) {
	key := NewEncryptionKey()
	plaintext := "my-secret-password"

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if ciphertext == plaintext {
		t.Error("ciphertext should differ from plaintext")
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypt() = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDifferentEachTime(t *testing.T) {
	key := NewEncryptionKey()
	plaintext := "same-input"

	c1, _ := Encrypt(key, plaintext)
	c2, _ := Encrypt(key, plaintext)

	if c1 == c2 {
		t.Error("two encryptions of same plaintext should produce different ciphertexts (random nonce)")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := NewEncryptionKey()
	key2 := NewEncryptionKey()
	plaintext := "secret"

	ciphertext, _ := Encrypt(key1, plaintext)

	_, err := Decrypt(key2, ciphertext)
	if err == nil {
		t.Error("Decrypt() with wrong key should fail")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := NewEncryptionKey()
	ciphertext, _ := Encrypt(key, "secret")

	// Tamper with ciphertext
	tampered := "AAAA" + ciphertext[4:]

	_, err := Decrypt(key, tampered)
	if err == nil {
		t.Error("Decrypt() with tampered ciphertext should fail")
	}
}

func TestDecryptEmptyCiphertext(t *testing.T) {
	key := NewEncryptionKey()
	_, err := Decrypt(key, "")
	if err == nil {
		t.Error("Decrypt() empty ciphertext should fail")
	}
}

func TestEncryptEmptyPlaintext(t *testing.T) {
	key := NewEncryptionKey()
	ciphertext, err := Encrypt(key, "")
	if err != nil {
		t.Fatalf("Encrypt() empty plaintext error = %v", err)
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if decrypted != "" {
		t.Errorf("Decrypt() = %q, want empty", decrypted)
	}
}

func TestNewEncryptionKeyLength(t *testing.T) {
	key := NewEncryptionKey()
	if len(key) != 32 {
		t.Errorf("key length = %d, want 32", len(key))
	}
}

func TestNewEncryptionKeyUnique(t *testing.T) {
	k1 := NewEncryptionKey()
	k2 := NewEncryptionKey()
	if string(k1) == string(k2) {
		t.Error("two generated keys should differ")
	}
}

func TestEncryptInvalidKeyLength(t *testing.T) {
	shortKey := []byte("too-short")

	_, err := Encrypt(shortKey, "secret")
	if err == nil {
		t.Error("Encrypt() with short key should fail")
	}
}

// ========== bcrypt ==========

func TestHashPassword(t *testing.T) {
	password := "my-password"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == password {
		t.Error("hash should differ from password")
	}

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword() should return true for correct password")
	}
}

func TestCheckPasswordWrong(t *testing.T) {
	hash, _ := HashPassword("correct-password")

	if CheckPassword("wrong-password", hash) {
		t.Error("CheckPassword() should return false for wrong password")
	}
}

func TestHashPasswordEmpty(t *testing.T) {
	_, err := HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword() empty password error = %v", err)
	}
}

func TestCheckPasswordInvalidHash(t *testing.T) {
	if CheckPassword("password", "not-a-bcrypt-hash") {
		t.Error("CheckPassword() should return false for invalid hash")
	}
}

func TestHashPasswordDifferentEachTime(t *testing.T) {
	h1, _ := HashPassword("password")
	h2, _ := HashPassword("password")

	if h1 == h2 {
		t.Error("two hashes of same password should differ (bcrypt salt)")
	}
}

// ========== LoadEncryptionKeyFromEnv ==========

func TestLoadEncryptionKeyFromEnv_Base64(t *testing.T) {
	// Generate a valid 32-byte key and base64 encode it
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	b64 := base64.StdEncoding.EncodeToString(key)

	t.Setenv("DEPLOYPILOT_ENCRYPTION_KEY", b64)
	result, err := LoadEncryptionKeyFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(result))
	}
	if string(result) != string(key) {
		t.Fatal("round-trip base64 key mismatch")
	}
}

func TestLoadEncryptionKeyFromEnv_Raw32(t *testing.T) {
	raw := "aaaa!aaaa@aaaa#aaaa$aaaa%aaaa^aa" // exactly 32 bytes, not valid base64
	if len(raw) != 32 {
		t.Fatalf("test bug: raw key length = %d, want 32", len(raw))
	}
	t.Setenv("DEPLOYPILOT_ENCRYPTION_KEY", raw)
	result, err := LoadEncryptionKeyFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != raw {
		t.Fatal("raw 32-byte key mismatch")
	}
}

func TestLoadEncryptionKeyFromEnv_InvalidLength(t *testing.T) {
	t.Setenv("DEPLOYPILOT_ENCRYPTION_KEY", "too-short")
	_, err := LoadEncryptionKeyFromEnv()
	if err == nil {
		t.Fatal("expected error for invalid key length")
	}
	if !strings.Contains(err.Error(), "invalid key") {
		t.Errorf("error should mention 'invalid key', got: %v", err)
	}
}

func TestLoadEncryptionKeyFromEnv_Empty(t *testing.T) {
	t.Setenv("DEPLOYPILOT_ENCRYPTION_KEY", "")
	result, err := LoadEncryptionKeyFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatal("expected nil for empty env")
	}
}

func TestLoadEncryptionKeyFromEnv_Base64WrongSize(t *testing.T) {
	// base64 of 16 bytes = valid base64 but wrong key size
	shortKey := make([]byte, 16)
	b64 := base64.StdEncoding.EncodeToString(shortKey)
	t.Setenv("DEPLOYPILOT_ENCRYPTION_KEY", b64)
	_, err := LoadEncryptionKeyFromEnv()
	if err == nil {
		t.Fatal("expected error for wrong base64 decoded size")
	}
	if !strings.Contains(err.Error(), "base64 decoded to 16") {
		t.Errorf("error should mention decoded size, got: %v", err)
	}
}

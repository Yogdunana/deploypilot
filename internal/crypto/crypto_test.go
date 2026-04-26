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
	if len(result) != 32 {
		t.Fatalf("expected auto-generated 32-byte key, got %d bytes", len(result))
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

func TestHashPassword_Verify(t *testing.T) {
	hash, err := HashPassword("test-password")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if !CheckPassword("test-password", hash) {
		t.Error("CheckPassword should return true for correct password")
	}
	if CheckPassword("wrong-password", hash) {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := NewEncryptionKey()
	plaintext := "hello world 1234"
	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("round-trip mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := NewEncryptionKey()
	key2 := NewEncryptionKey()
	encrypted, _ := Encrypt(key1, "secret")
	_, err := Decrypt(key2, encrypted)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	key := NewEncryptionKey()
	_, err := Decrypt(key, "not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	key := NewEncryptionKey()
	// Valid base64 but too short for nonce + ciphertext
	short := "AQID" // 4 bytes decoded = 3 bytes
	_, err := Decrypt(key, short)
	if err == nil {
		t.Error("expected error for too-short ciphertext")
	}
}

func TestEncrypt_ExactNonceSize(t *testing.T) {
	key := NewEncryptionKey()
	// Encrypt and decrypt to verify nonce handling
	plaintext := "test-plaintext-data"
	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("got %q, want %q", decrypted, plaintext)
	}
}

func TestDecrypt_InvalidBase64Chars(t *testing.T) {
	key := NewEncryptionKey()
	_, err := Decrypt(key, "!!!invalid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64 characters")
	}
}

func TestEncrypt_LargePlaintext(t *testing.T) {
	key := NewEncryptionKey()
	// 1MB plaintext
	largeText := strings.Repeat("a", 1024*1024)
	ciphertext, err := Encrypt(key, largeText)
	if err != nil {
		t.Fatalf("Encrypt large plaintext failed: %v", err)
	}
	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt large plaintext failed: %v", err)
	}
	if decrypted != largeText {
		t.Errorf("large plaintext round-trip failed: got %d bytes, want %d", len(decrypted), len(largeText))
	}
}

func TestDecrypt_CorruptedNonce(t *testing.T) {
	key := NewEncryptionKey()
	ciphertext, _ := Encrypt(key, "secret")
	// Corrupt the first few bytes (nonce)
	data, _ := base64.StdEncoding.DecodeString(ciphertext)
	data[0] ^= 0xFF
	corrupted := base64.StdEncoding.EncodeToString(data)
	_, err := Decrypt(key, corrupted)
	if err == nil {
		t.Error("expected error for corrupted nonce")
	}
}

func TestDecrypt_CorruptedCiphertext(t *testing.T) {
	key := NewEncryptionKey()
	ciphertext, _ := Encrypt(key, "secret")
	data, _ := base64.StdEncoding.DecodeString(ciphertext)
	// Corrupt the last byte (ciphertext, not nonce)
	data[len(data)-1] ^= 0xFF
	corrupted := base64.StdEncoding.EncodeToString(data)
	_, err := Decrypt(key, corrupted)
	if err == nil {
		t.Error("expected error for corrupted ciphertext")
	}
}

func TestEncrypt_InvalidKeySize(t *testing.T) {
	// 15 bytes instead of 16/24/32
	shortKey := make([]byte, 15)
	for i := range shortKey {
		shortKey[i] = byte(i)
	}
	_, err := Encrypt(shortKey, "test")
	if err == nil {
		t.Error("Encrypt() should fail with 15-byte key")
	}
}

func TestDecrypt_InvalidKeySize(t *testing.T) {
	shortKey := make([]byte, 15)
	for i := range shortKey {
		shortKey[i] = byte(i)
	}
	_, err := Decrypt(shortKey, "AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyAhIiMk")
	if err == nil {
		t.Error("Decrypt() should fail with 15-byte key")
	}
}

func TestHashPassword_LongPassword(t *testing.T) {
	// bcrypt has a 72-byte limit - passwords at exactly 72 bytes should work
	longPassword := strings.Repeat("a", 72)
	hash, err := HashPassword(longPassword)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !CheckPassword(longPassword, hash) {
		t.Error("CheckPassword() should return true for 72-byte password")
	}
}

func TestCheckPassword_EmptyHash(t *testing.T) {
	if CheckPassword("password", "") {
		t.Error("CheckPassword() should return false for empty hash")
	}
}

func TestCheckPassword_EmptyBoth(t *testing.T) {
	if CheckPassword("", "") {
		t.Error("CheckPassword() should return false for empty password and hash")
	}
}

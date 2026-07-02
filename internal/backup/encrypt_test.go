package backup

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptBackup_ValidKey(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	plaintext := []byte("hello world, this is a backup!")

	ciphertext, err := EncryptBackup(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptBackup failed: %v", err)
	}
	if len(ciphertext) == 0 {
		t.Error("expected non-empty ciphertext")
	}
	// Should have header (8) + nonce (12) + sealed data
	if len(ciphertext) <= headerSize+nonceSize {
		t.Errorf("ciphertext too short: %d bytes", len(ciphertext))
	}
}

func TestEncryptBackup_InvalidKeySize(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{"too short", make([]byte, 16)},
		{"too long", make([]byte, 64)},
		{"empty", []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncryptBackup(tt.key, []byte("test"))
			if err == nil {
				t.Error("expected error for invalid key size")
			}
		})
	}
}

func TestDecryptBackup_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	plaintext := []byte("sensitive backup data that needs protection")

	ciphertext, err := EncryptBackup(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptBackup failed: %v", err)
	}

	decrypted, err := DecryptBackup(key, ciphertext)
	if err != nil {
		t.Fatalf("DecryptBackup failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted data doesn't match original\ngot:  %x\nwant: %x", decrypted, plaintext)
	}
}

func TestDecryptBackup_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	rand.Read(key1)
	rand.Read(key2)

	ciphertext, _ := EncryptBackup(key1, []byte("secret data"))

	_, err := DecryptBackup(key2, ciphertext)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestDecryptBackup_InvalidKeySize(t *testing.T) {
	_, err := DecryptBackup(make([]byte, 16), make([]byte, 100))
	if err == nil {
		t.Error("expected error for invalid key size")
	}
}

func TestDecryptBackup_DataTooShort(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	_, err := DecryptBackup(key, []byte("short"))
	if err == nil {
		t.Error("expected error for data too short")
	}
}

func TestDecryptBackup_InvalidHeader(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	// Create data with wrong header
	data := make([]byte, headerSize+nonceSize+16)
	copy(data, []byte("BADHEAD0"))

	_, err := DecryptBackup(key, data)
	if err == nil {
		t.Error("expected error for invalid header")
	}
}

func TestDecryptBackup_TamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	ciphertext, _ := EncryptBackup(key, []byte("important data"))

	// Tamper with the ciphertext
	ciphertext[len(ciphertext)-1] ^= 0xFF

	_, err := DecryptBackup(key, ciphertext)
	if err == nil {
		t.Error("expected error for tampered ciphertext (GCM authentication)")
	}
}

func TestDecryptBackup_TamperedNonce(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	ciphertext, _ := EncryptBackup(key, []byte("important data"))

	// Tamper with the nonce
	ciphertext[headerSize] ^= 0xFF

	_, err := DecryptBackup(key, ciphertext)
	if err == nil {
		t.Error("expected error for tampered nonce (GCM authentication)")
	}
}

func TestIsEncryptedBackup_ValidHeader(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	ciphertext, _ := EncryptBackup(key, []byte("test"))

	if !IsEncryptedBackup(ciphertext) {
		t.Error("expected IsEncryptedBackup=true for encrypted data")
	}
}

func TestIsEncryptedBackup_PlainData(t *testing.T) {
	if IsEncryptedBackup([]byte("plain text data")) {
		t.Error("expected IsEncryptedBackup=false for plain data")
	}
}

func TestIsEncryptedBackup_EmptyData(t *testing.T) {
	if IsEncryptedBackup([]byte{}) {
		t.Error("expected IsEncryptedBackup=false for empty data")
	}
}

func TestIsEncryptedBackup_TooShort(t *testing.T) {
	if IsEncryptedBackup([]byte("DPEN")) {
		t.Error("expected IsEncryptedBackup=false for data shorter than header")
	}
}

func TestEncryptBackup_EmptyPlaintext(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	ciphertext, err := EncryptBackup(key, []byte{})
	if err != nil {
		t.Fatalf("EncryptBackup with empty plaintext failed: %v", err)
	}

	decrypted, err := DecryptBackup(key, ciphertext)
	if err != nil {
		t.Fatalf("DecryptBackup failed: %v", err)
	}
	if len(decrypted) != 0 {
		t.Errorf("expected empty decrypted data, got %d bytes", len(decrypted))
	}
}

func TestEncryptBackup_LargeData(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	plaintext := make([]byte, 1024*1024) // 1MB
	rand.Read(plaintext)

	ciphertext, err := EncryptBackup(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptBackup with large data failed: %v", err)
	}

	decrypted, err := DecryptBackup(key, ciphertext)
	if err != nil {
		t.Fatalf("DecryptBackup with large data failed: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Error("decrypted large data doesn't match original")
	}
}

func TestEncryptBackup_DifferentNonces(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	plaintext := []byte("same data")

	ct1, _ := EncryptBackup(key, plaintext)
	ct2, _ := EncryptBackup(key, plaintext)

	// Two encryptions of the same data should produce different ciphertexts
	if bytes.Equal(ct1, ct2) {
		t.Error("expected different ciphertexts due to different nonces")
	}
}

package backup

import (
	"bytes"
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	testCases := []struct {
		name      string
		plaintext []byte
	}{
		{"empty", []byte{}},
		{"short", []byte("hello")},
		{"with spaces", []byte("hello world")},
		{"binary", []byte{0x00, 0xff, 0x80, 0x7f}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := EncryptBackup(key, tc.plaintext)
			if err != nil {
				t.Fatalf("EncryptBackup failed: %v", err)
			}

			decrypted, err := DecryptBackup(key, encrypted)
			if err != nil {
				t.Fatalf("DecryptBackup failed: %v", err)
			}

			if !bytes.Equal(decrypted, tc.plaintext) {
				t.Errorf("round trip failed: got %v, want %v", decrypted, tc.plaintext)
			}
		})
	}
}

func TestEncryptBackup_InvalidKey(t *testing.T) {
	testCases := []struct {
		name string
		key  []byte
	}{
		{"short", []byte("short")},
		{"long", make([]byte, 40)},
		{"empty", []byte{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := EncryptBackup(tc.key, []byte("test"))
			if err == nil {
				t.Error("expected error for invalid key")
			}
		})
	}
}

func TestIsEncryptedBackup(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encrypted, _ := EncryptBackup(key, []byte("test"))

	testCases := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{"encrypted", encrypted, true},
		{"empty", []byte{}, false},
		{"short", []byte("DPE"), false},
		{"wrong header", []byte("INVALID01"+strings.Repeat("x", 20)), false},
		{"plain text", []byte("hello"), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsEncryptedBackup(tc.data)
			if result != tc.expected {
				t.Errorf("IsEncryptedBackup = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestEncryptBackup_DifferentKeys(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key1 {
		key1[i] = byte(i)
		key2[i] = byte(i + 1)
	}

	enc1, _ := EncryptBackup(key1, []byte("test"))
	enc2, _ := EncryptBackup(key2, []byte("test"))

	if bytes.Equal(enc1, enc2) {
		t.Error("different keys should produce different output")
	}
}
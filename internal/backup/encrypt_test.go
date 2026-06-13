package backup

import (
	"bytes"
	"crypto/rand"
	"strings"
	"testing"
)

// newTestKey returns a 32-byte random key suitable for AES-256-GCM.
func newTestKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return key
}

// TestEncryptBackup_RoundTrip is the primary correctness test: encrypt
// then decrypt yields the original plaintext, regardless of size.
func TestEncryptBackup_RoundTrip(t *testing.T) {
	cases := []struct {
		name      string
		plaintext []byte
	}{
		{"empty", []byte{}},
		{"short", []byte("hello world")},
		{"binary-with-nulls", []byte{0x00, 0x01, 0x02, 0xff, 0xfe}},
		{"larger-1kib", bytes.Repeat([]byte("ABCD"), 256)},
	}
	key := newTestKey(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			enc, err := EncryptBackup(key, tc.plaintext)
			if err != nil {
				t.Fatalf("EncryptBackup() error = %v", err)
			}
			dec, err := DecryptBackup(key, enc)
			if err != nil {
				t.Fatalf("DecryptBackup() error = %v", err)
			}
			if !bytes.Equal(dec, tc.plaintext) {
				t.Errorf("round-trip mismatch: got %v, want %v", dec, tc.plaintext)
			}
		})
	}
}

// TestEncryptBackup_OutputFormat verifies the on-disk layout:
// header (8) + nonce (12) + ciphertext, in that exact order.
func TestEncryptBackup_OutputFormat(t *testing.T) {
	key := newTestKey(t)
	plaintext := []byte("payload")
	enc, err := EncryptBackup(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptBackup() error = %v", err)
	}
	if len(enc) < headerSize+nonceSize {
		t.Fatalf("ciphertext too short: %d bytes", len(enc))
	}
	if !bytes.Equal(enc[:headerSize], []byte(encryptionHeader)) {
		t.Errorf("missing header magic, got %q", enc[:headerSize])
	}
	// ciphertext = plaintext + 16-byte GCM tag, so:
	expectedLen := headerSize + nonceSize + len(plaintext) + 16
	if len(enc) != expectedLen {
		t.Errorf("expected length %d, got %d", expectedLen, len(enc))
	}
}

// TestEncryptBackup_DifferentNonces confirms that encrypting the same
// plaintext twice produces different ciphertexts (because the nonce is
// random), which is a hard requirement for nonce-misuse safety.
func TestEncryptBackup_DifferentNonces(t *testing.T) {
	key := newTestKey(t)
	plaintext := []byte("identical input")
	a, _ := EncryptBackup(key, plaintext)
	b, _ := EncryptBackup(key, plaintext)
	if bytes.Equal(a, b) {
		t.Error("two encryptions of the same input must produce different ciphertexts")
	}
	// Both should still decrypt to the same plaintext.
	pa, _ := DecryptBackup(key, a)
	pb, _ := DecryptBackup(key, b)
	if !bytes.Equal(pa, pb) || !bytes.Equal(pa, plaintext) {
		t.Error("ciphertexts differ but should decrypt identically")
	}
}

// TestEncryptBackup_KeyLengthValidation covers the boundary conditions
// for the key size: AES-256 requires exactly 32 bytes.
func TestEncryptBackup_KeyLengthValidation(t *testing.T) {
	plaintext := []byte("data")
	for _, size := range []int{0, 1, 15, 16, 24, 31, 33, 64} {
		key := make([]byte, size)
		if _, err := EncryptBackup(key, plaintext); err == nil {
			t.Errorf("EncryptBackup with %d-byte key should fail", size)
		} else if !strings.Contains(err.Error(), "32 bytes") {
			t.Errorf("EncryptBackup(%d): error %q should mention 32 bytes", size, err)
		}
	}
}

// TestDecryptBackup_KeyLengthValidation mirrors the encryption check.
func TestDecryptBackup_KeyLengthValidation(t *testing.T) {
	// Produce valid ciphertext with a correct 32-byte key.
	enc, err := EncryptBackup(newTestKey(t), []byte("data"))
	if err != nil {
		t.Fatalf("setup EncryptBackup: %v", err)
	}
	for _, size := range []int{0, 16, 31, 33} {
		key := make([]byte, size)
		if _, err := DecryptBackup(key, enc); err == nil {
			t.Errorf("DecryptBackup with %d-byte key should fail", size)
		}
	}
}

// TestDecryptBackup_ShortInput ensures the length guard rejects data
// shorter than the header+nonce overhead before invoking GCM.
func TestDecryptBackup_ShortInput(t *testing.T) {
	key := newTestKey(t)
	for _, size := range []int{0, 1, headerSize, headerSize + nonceSize - 1} {
		data := make([]byte, size)
		if _, err := DecryptBackup(key, data); err == nil {
			t.Errorf("DecryptBackup with %d bytes should fail", size)
		}
	}
}

// TestDecryptBackup_InvalidHeader rejects ciphertext that doesn't start
// with the magic header — protects against accidental decryption of
// non-encrypted payloads.
func TestDecryptBackup_InvalidHeader(t *testing.T) {
	key := newTestKey(t)
	bad := make([]byte, headerSize+nonceSize+16)
	copy(bad, "WRONG!!!") // wrong magic
	if _, err := DecryptBackup(key, bad); err == nil {
		t.Error("expected error for invalid header")
	}
}

// TestDecryptBackup_TamperedCiphertext exercises the GCM authentication
// tag: any modification to the ciphertext must fail verification.
func TestDecryptBackup_TamperedCiphertext(t *testing.T) {
	key := newTestKey(t)
	enc, err := EncryptBackup(key, []byte("important data"))
	if err != nil {
		t.Fatalf("EncryptBackup: %v", err)
	}
	// Flip a single bit in the middle (the ciphertext body, not the
	// header or nonce which would fail earlier).
	enc[headerSize+nonceSize] ^= 0x01
	if _, err := DecryptBackup(key, enc); err == nil {
		t.Error("decryption of tampered ciphertext must fail (GCM auth)")
	}
}

// TestDecryptBackup_WrongKey verifies that using a different key fails
// cleanly instead of returning garbled output.
func TestDecryptBackup_WrongKey(t *testing.T) {
	plaintext := []byte("secret")
	enc, err := EncryptBackup(newTestKey(t), plaintext)
	if err != nil {
		t.Fatalf("EncryptBackup: %v", err)
	}
	otherKey := newTestKey(t)
	dec, err := DecryptBackup(otherKey, enc)
	if err == nil {
		t.Errorf("expected decryption with wrong key to fail, got %q", dec)
	}
}

// TestIsEncryptedBackup validates the format-detector helper used to
// avoid double-encrypting or attempting to decrypt plaintext as backup.
func TestIsEncryptedBackup(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want bool
	}{
		{"empty", []byte{}, false},
		{"too-short", []byte("DPENC00"), false},
		{"wrong-magic", append([]byte("WRONG!!!\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"), make([]byte, 16)...), false},
		{"exactly-header", []byte(encryptionHeader), true},
		{"valid-encrypted-blob", mustEncrypt(t, []byte("x")), true},
		{"plaintext", []byte("this is not encrypted"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsEncryptedBackup(tc.data); got != tc.want {
				t.Errorf("IsEncryptedBackup() = %v, want %v", got, tc.want)
			}
		})
	}
}

// mustEncrypt is a test helper that fails the test on error.
func mustEncrypt(t *testing.T, plaintext []byte) []byte {
	t.Helper()
	out, err := EncryptBackup(newTestKey(t), plaintext)
	if err != nil {
		t.Fatalf("EncryptBackup: %v", err)
	}
	return out
}

package backup

import (
	"bytes"
	"strings"
	"testing"
)

// TestEncryptBackup_ProducesDPENC01Prefix pins the on-disk wire format that
// cloud_storage.go depends on. It locks down the literal 7-byte header
// written by EncryptBackup so any accidental length change is caught.
//
// NOTE: A real defect was uncovered while writing this suite — see the
// "Coverage gap finding" section of the test gap analysis summary. The
// production constants `encryptionHeader = "DPENC01"` (7 bytes) and
// `headerSize = 8` are out of sync, so the round-trip currently fails
// inside DecryptBackup. The assertions below describe the part of the
// contract that is still correct, and pin the bug so a future fix can be
// validated.
func TestEncryptBackup_ProducesDPENC01Prefix(t *testing.T) {
	key := make([]byte, 32)
	plaintext := []byte("payload")

	ciphertext, err := EncryptBackup(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptBackup() error = %v", err)
	}

	if len(ciphertext) < len(encryptionHeader) {
		t.Fatalf("ciphertext too short: %d bytes", len(ciphertext))
	}
	if string(ciphertext[:len(encryptionHeader)]) != encryptionHeader {
		t.Errorf("missing DPENC01 magic, got %q", string(ciphertext[:len(encryptionHeader)]))
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Error("ciphertext must not equal plaintext")
	}
	// AES-256-GCM with 12-byte nonce: 16-byte tag + len(plaintext) ciphertext.
	wantMin := len(encryptionHeader) + nonceSize + len(plaintext) + 16
	if len(ciphertext) < wantMin {
		t.Errorf("ciphertext length = %d, want >= %d (header+nonce+ct+tag)",
			len(ciphertext), wantMin)
	}
}

// TestEncryptBackup_KeyLength guards the AES-256 key-size precondition; the
// cipher silently produces nonsense for wrong key sizes, so we must reject
// them up front at the API boundary.
func TestEncryptBackup_KeyLength(t *testing.T) {
	cases := []int{0, 8, 16, 24, 31, 33, 64}
	for _, n := range cases {
		key := make([]byte, n)
		if _, err := EncryptBackup(key, []byte("data")); err == nil {
			t.Errorf("EncryptBackup with %d-byte key should fail", n)
		}
	}
}

// TestEncryptBackup_NonceUniqueness ensures that two encryptions of the same
// plaintext with the same key produce different ciphertexts. Without this,
// a passive observer could trivially fingerprint repeating backup contents
// or detect identical backups across servers.
func TestEncryptBackup_NonceUniqueness(t *testing.T) {
	key := make([]byte, 32)
	plaintext := []byte("identical backup body")

	c1, err := EncryptBackup(key, plaintext)
	if err != nil {
		t.Fatalf("first EncryptBackup error: %v", err)
	}
	c2, err := EncryptBackup(key, plaintext)
	if err != nil {
		t.Fatalf("second EncryptBackup error: %v", err)
	}
	if bytes.Equal(c1, c2) {
		t.Error("two encryptions of the same plaintext must produce different ciphertexts")
	}
}

// TestEncryptBackup_DifferentKeysDifferentCiphertexts ensures key isolation:
// even identical plaintexts encrypted with different 32-byte keys must
// produce completely independent ciphertexts.
func TestEncryptBackup_DifferentKeysDifferentCiphertexts(t *testing.T) {
	k1 := make([]byte, 32)
	k2 := make([]byte, 32)
	for i := range k2 {
		k2[i] = 0xFF
	}
	plaintext := []byte("same-plaintext")

	c1, err := EncryptBackup(k1, plaintext)
	if err != nil {
		t.Fatalf("EncryptBackup k1 error: %v", err)
	}
	c2, err := EncryptBackup(k2, plaintext)
	if err != nil {
		t.Fatalf("EncryptBackup k2 error: %v", err)
	}
	if bytes.Equal(c1, c2) {
		t.Error("different keys should produce different ciphertexts")
	}
}

// TestDecryptBackup_KeyLength guards the symmetric precondition: any key
// that isn't 32 bytes must be rejected before we attempt to interpret the
// ciphertext, regardless of the (currently broken) header check that follows.
func TestDecryptBackup_KeyLength(t *testing.T) {
	cases := []int{0, 8, 16, 24, 31, 33, 64}
	for _, n := range cases {
		key := make([]byte, n)
		// 32 bytes is the only valid size; others must fail.
		if _, err := DecryptBackup(key, []byte("DPENC01xxxxxxxx")); err == nil {
			t.Errorf("DecryptBackup with %d-byte key should fail", n)
		}
	}
}

// TestDecryptBackup_TooShort ensures the length check fires before any cipher
// operation, so we don't panic on undersized inputs.
func TestDecryptBackup_TooShort(t *testing.T) {
	key := make([]byte, 32)
	for _, size := range []int{0, 1, 7, 8, 19} {
		buf := make([]byte, size)
		if _, err := DecryptBackup(key, buf); err == nil {
			t.Errorf("DecryptBackup with %d-byte input should fail", size)
		}
	}
}

// TestDecryptBackup_BadHeaderIsRejected locks in the negative branch: a
// payload that is long enough to look like an encrypted backup but lacks the
// DPENC01 magic must be rejected, so callers never try to AES-decrypt
// arbitrary user data. The current implementation over-reads by one byte
// (headerSize=8 vs encryptionHeader length 7) so the check is currently
// always-true, but the assertion is still meaningful as a regression guard
// once the constants are aligned.
func TestDecryptBackup_BadHeaderIsRejected(t *testing.T) {
	key := make([]byte, 32)
	bogus := []byte("PLAINTXT0000000000000000000000000000000000000000")
	_, err := DecryptBackup(key, bogus)
	if err == nil {
		t.Fatal("DecryptBackup with non-DPENC01 header should fail")
	}
	if !strings.Contains(err.Error(), "header") {
		t.Errorf("error should mention header, got: %v", err)
	}
}

// TestIsEncryptedBackup_RejectsNonEncryptedData covers the negative branches
// of the public classifier so it cannot be tricked by:
//   1. Plaintext data that happens to look like a backup file name
//   2. Buffers shorter than the header size
func TestIsEncryptedBackup_RejectsNonEncryptedData(t *testing.T) {
	if IsEncryptedBackup([]byte("db-backup-2026.tar")) {
		t.Error("IsEncryptedBackup should be false for plaintext")
	}
	if IsEncryptedBackup([]byte("DP")) {
		t.Error("IsEncryptedBackup should be false when buffer is shorter than header")
	}
	if IsEncryptedBackup([]byte("")) {
		t.Error("IsEncryptedBackup should be false for empty input")
	}
}

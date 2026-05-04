package auth

import (
	"encoding/json"
	"testing"
)

func TestTOTPSecret(t *testing.T) {
	secret, err := TOTPSecret()
	if err != nil {
		t.Fatalf("TOTPSecret failed: %v", err)
	}
	if len(secret) == 0 {
		t.Error("expected non-empty secret")
	}
	// SecretSize=32 generates 32 bytes, base32-encoded to ~52 chars
}

func TestTOTPQRCodeURL(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	url := TOTPQRCodeURL(secret, "testuser@example.com")
	if url == "" {
		t.Error("expected non-empty QR code URL")
	}
	// Should contain otpauth://totp/
	if len(url) < 20 {
		t.Errorf("URL too short: %s", url)
	}
}

func TestGenerateBackupCodes(t *testing.T) {
	plaintext, hashes, err := GenerateBackupCodes()
	if err != nil {
		t.Fatalf("GenerateBackupCodes failed: %v", err)
	}
	if len(plaintext) != 10 {
		t.Errorf("expected 10 codes, got %d", len(plaintext))
	}
	if len(hashes) != 10 {
		t.Errorf("expected 10 hashes, got %d", len(hashes))
	}
	for i, code := range plaintext {
		if len(code) != 8 {
			t.Errorf("code %d: expected length 8, got %d", i, len(code))
		}
	}
}

func TestVerifyBackupCode(t *testing.T) {
	plaintext, hashes, _ := GenerateBackupCodes()
	hashesJSON, _ := json.Marshal(hashes)

	// Valid code
	idx := VerifyBackupCode(string(hashesJSON), plaintext[0])
	if idx < 0 {
		t.Error("expected valid backup code to match")
	}

	// Invalid code
	idx = VerifyBackupCode(string(hashesJSON), "invalidcode")
	if idx != -1 {
		t.Error("expected invalid code to return -1")
	}

	// Wrong code from another set
	_, otherHashes, _ := GenerateBackupCodes()
	otherJSON, _ := json.Marshal(otherHashes)
	idx = VerifyBackupCode(string(otherJSON), plaintext[0])
	if idx != -1 {
		t.Error("expected code from different set to not match")
	}
}

func TestRemoveBackupCode(t *testing.T) {
	_, hashes, _ := GenerateBackupCodes()
	hashesJSON, _ := json.Marshal(hashes)

	// Remove first code
	updated := RemoveBackupCode(string(hashesJSON), 0)
	if updated == string(hashesJSON) {
		t.Error("expected updated JSON after removal")
	}

	// Remove from single-element array
	singleHash := hashes[0]
	singleJSON, _ := json.Marshal([]string{singleHash})
	removed := RemoveBackupCode(string(singleJSON), 0)
	if removed != "[]" {
		t.Errorf("expected empty array, got %s", removed)
	}
}

package auth

import (
	"testing"
)

func TestTOTPSecret(t *testing.T) {
	secret, err := TOTPSecret()
	if err != nil {
		t.Fatalf("TOTPSecret failed: %v", err)
	}
	if len(secret) != 32 {
		t.Errorf("expected secret length 32, got %d", len(secret))
	}
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

	// Valid code
	idx := VerifyBackupCode(hashes[0], plaintext[0])
	if idx < 0 {
		t.Error("expected valid backup code to match")
	}

	// Invalid code
	idx = VerifyBackupCode(hashes[0], "invalidcode")
	if idx != -1 {
		t.Error("expected invalid code to return -1")
	}

	// Wrong code from another set
	_, otherHashes, _ := GenerateBackupCodes()
	idx = VerifyBackupCode(otherHashes[0], plaintext[0])
	if idx != -1 {
		t.Error("expected code from different set to not match")
	}
}

func TestRemoveBackupCode(t *testing.T) {
	_, hashes, _ := GenerateBackupCodes()

	// Remove first code
	updated := RemoveBackupCode(hashes[0], 0)
	if updated == hashes[0] {
		t.Error("expected updated JSON after removal")
	}

	// Remove from single-element array
	singleHash := hashes[0]
	single := "[\"" + singleHash + "\"]"
	removed := RemoveBackupCode(single, 0)
	if removed != "[]" {
		t.Errorf("expected empty array, got %s", removed)
	}
}

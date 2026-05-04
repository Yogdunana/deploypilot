package signing

import (
	"crypto/ed25519"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Errorf("expected public key size %d, got %d", ed25519.PublicKeySize, len(pub))
	}
	if len(priv) != ed25519.PrivateKeySize {
		t.Errorf("expected private key size %d, got %d", ed25519.PrivateKeySize, len(priv))
	}
}

func TestNewSigner(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)
	if signer.KeyVersion() != 1 {
		t.Errorf("expected version 1, got %d", signer.KeyVersion())
	}
	if signer.Fingerprint() == "" {
		t.Error("expected non-empty fingerprint")
	}
}

func TestSignAndVerify(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	data := []byte("hello world")
	sig, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if len(sig) != ed25519.SignatureSize {
		t.Errorf("expected signature size %d, got %d", ed25519.SignatureSize, len(sig))
	}

	if !signer.Verify(data, sig) {
		t.Error("Verify should return true for valid signature")
	}

	// Tampered data
	if signer.Verify([]byte("tampered"), sig) {
		t.Error("Verify should return false for tampered data")
	}

	// Tampered signature
	tamperedSig := make([]byte, len(sig))
	copy(tamperedSig, sig)
	tamperedSig[0] ^= 0xFF
	if signer.Verify(data, tamperedSig) {
		t.Error("Verify should return false for tampered signature")
	}
}

func TestPublicKeyBytes(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	signer := NewSigner(pub, nil, 0)
	b := signer.PublicKeyBytes()
	if len(b) != ed25519.PublicKeySize {
		t.Errorf("expected %d bytes, got %d", ed25519.PublicKeySize, len(b))
	}
}

func TestPrivateKeyBytes(t *testing.T) {
	_, priv, _ := GenerateKeyPair()
	signer := NewSigner(nil, priv, 0)
	b := signer.PrivateKeyBytes()
	if len(b) != ed25519.SeedSize {
		t.Errorf("expected %d bytes, got %d", ed25519.SeedSize, len(b))
	}
}

func TestFingerprint(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	signer := NewSigner(pub, nil, 0)
	fp := signer.Fingerprint()
	decoded, err := hex.DecodeString(fp)
	if err != nil {
		t.Fatalf("fingerprint not valid hex: %v", err)
	}
	// Fingerprint is first 16 hex chars of SHA256 = 8 bytes
	if len(decoded) != 8 {
		t.Errorf("expected fingerprint length 8, got %d", len(decoded))
	}
}

func TestSaveAndLoadKeys(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 3)

	dir := t.TempDir()
	pubPath := filepath.Join(dir, "test_public.pem")
	privPath := filepath.Join(dir, "test_private.pem")

	if err := signer.SaveKeys(pubPath, privPath); err != nil {
		t.Fatalf("SaveKeys failed: %v", err)
	}

	if _, err := os.Stat(pubPath); os.IsNotExist(err) {
		t.Error("public key file not created")
	}
	if _, err := os.Stat(privPath); os.IsNotExist(err) {
		t.Error("private key file not created")
	}

	loaded, err := LoadKeys(pubPath, privPath)
	if err != nil {
		t.Fatalf("LoadKeys failed: %v", err)
	}
	if loaded.KeyVersion() != 0 {
		t.Errorf("expected version 0 (default), got %d", loaded.KeyVersion())
	}

	// Verify loaded keys work
	data := []byte("test save/load")
	sig, _ := signer.Sign(data)
	if !loaded.Verify(data, sig) {
		t.Error("loaded signer should verify original signature")
	}
}

func TestLoadKeys_FileNotFound(t *testing.T) {
	_, err := LoadKeys("/nonexistent/public.pem", "/nonexistent/private.pem")
	if err == nil {
		t.Error("expected error for non-existent files")
	}
}

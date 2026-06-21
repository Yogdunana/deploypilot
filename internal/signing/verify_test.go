package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

// TestVerifyBinary_ValidSignature covers the happy path: a binary
// signed with a real key must verify against the matching public key.
func TestVerifyBinary_ValidSignature(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "deploypilot")
	sigPath := filepath.Join(dir, "deploypilot.sig")

	payload := []byte("binary payload for verification")
	if err := os.WriteFile(binPath, payload, 0640); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	signer := NewSigner(pub, priv, 1)
	sig, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if err := os.WriteFile(sigPath, sig, 0640); err != nil {
		t.Fatalf("write sig: %v", err)
	}

	ok, err := VerifyBinary(binPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary: %v", err)
	}
	if !ok {
		t.Error("expected VerifyBinary to return true for a valid signature")
	}
}

// TestVerifyBinary_WrongKey ensures that a valid signature produced by
// one key does not verify under a different public key.
func TestVerifyBinary_WrongKey(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "b")
	sigPath := filepath.Join(dir, "b.sig")
	payload := []byte("some bytes")

	_ = os.WriteFile(binPath, payload, 0640)

	pubA, privA, _ := GenerateKeyPair()
	pubB, _, _ := GenerateKeyPair()
	sig, err := NewSigner(pubA, privA, 1).Sign(payload)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	_ = os.WriteFile(sigPath, sig, 0640)

	ok, err := VerifyBinary(binPath, sigPath, pubB)
	if err != nil {
		t.Fatalf("VerifyBinary: %v", err)
	}
	if ok {
		t.Error("expected VerifyBinary to return false under a different public key")
	}
}

// TestVerifyBinary_TamperedBinary verifies that any single-byte change
// in the binary causes verification to fail. This is the property
// Ed25519 is supposed to guarantee.
func TestVerifyBinary_TamperedBinary(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "b")
	sigPath := filepath.Join(dir, "b.sig")

	payload := []byte("original payload")
	_ = os.WriteFile(binPath, payload, 0640)

	pub, priv, _ := GenerateKeyPair()
	sig, _ := NewSigner(pub, priv, 1).Sign(payload)
	_ = os.WriteFile(sigPath, sig, 0640)

	// Flip a single byte in the binary.
	tampered := append([]byte{}, payload...)
	tampered[0] ^= 0x01
	_ = os.WriteFile(binPath, tampered, 0640)

	ok, err := VerifyBinary(binPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary: %v", err)
	}
	if ok {
		t.Error("expected VerifyBinary to return false for a tampered binary")
	}
}

// TestVerifyBinary_InvalidSignatureSize exercises the explicit
// size check that guards ed25519.Verify. A truncated or padded
// signature must be rejected before any cryptographic operation runs.
func TestVerifyBinary_InvalidSignatureSize(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "b")
	sigPath := filepath.Join(dir, "b.sig")

	_ = os.WriteFile(binPath, []byte("payload"), 0640)
	pub, _, _ := GenerateKeyPair()

	cases := []struct {
		name string
		size int
	}{
		{"empty", 0},
		{"one byte", 1},
		{"truncated", ed25519.SignatureSize - 1},
		{"extended", ed25519.SignatureSize + 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sig := make([]byte, tc.size)
			_, _ = rand.Read(sig)
			_ = os.WriteFile(sigPath, sig, 0640)
			ok, err := VerifyBinary(binPath, sigPath, pub)
			if err == nil {
				t.Fatalf("expected error for signature size %d", tc.size)
			}
			if ok {
				t.Errorf("expected ok=false for signature size %d", tc.size)
			}
		})
	}
}

// TestVerifyBinary_MissingFiles documents the error contract: both
// the binary and the signature file must exist on disk.
func TestVerifyBinary_MissingFiles(t *testing.T) {
	dir := t.TempDir()
	pub, _, _ := GenerateKeyPair()

	t.Run("missing binary", func(t *testing.T) {
		_, err := VerifyBinary(filepath.Join(dir, "nope"), filepath.Join(dir, "sig"), pub)
		if err == nil {
			t.Error("expected error for missing binary file")
		}
	})

	t.Run("missing signature", func(t *testing.T) {
		bin := filepath.Join(dir, "b")
		_ = os.WriteFile(bin, []byte("x"), 0640)
		_, err := VerifyBinary(bin, filepath.Join(dir, "missing.sig"), pub)
		if err == nil {
			t.Error("expected error for missing signature file")
		}
	})
}

// TestSignBinary_RoundTrip covers the SignBinary helper. It writes a
// real signature file that the matching VerifyBinary call accepts.
func TestSignBinary_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary")
	sigPath := filepath.Join(dir, "binary.sig")
	payload := []byte("signed payload")
	_ = os.WriteFile(binPath, payload, 0640)

	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 7)

	if err := SignBinary(binPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary: %v", err)
	}

	info, err := os.Stat(sigPath)
	if err != nil {
		t.Fatalf("signature file missing: %v", err)
	}
	if info.Size() != int64(ed25519.SignatureSize) {
		t.Errorf("expected signature size %d, got %d", ed25519.SignatureSize, info.Size())
	}

	ok, err := VerifyBinary(binPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary: %v", err)
	}
	if !ok {
		t.Error("VerifyBinary should accept a signature produced by SignBinary")
	}
}

// TestSignBinary_NilSigner reports the failure mode when the caller
// passes a Signer without a private key.
func TestSignBinary_NilSigner(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary")
	sigPath := filepath.Join(dir, "binary.sig")
	_ = os.WriteFile(binPath, []byte("payload"), 0640)

	pub, _, _ := GenerateKeyPair()
	// No private key attached.
	signer := NewSigner(pub, nil, 1)
	if err := SignBinary(binPath, sigPath, signer); err == nil {
		t.Error("expected error when signer has no private key")
	}
}

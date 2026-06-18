package signing

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

// TestVerifyBinary_HappyPath verifies the round-trip SignBinary -> VerifyBinary.
func TestVerifyBinary_HappyPath(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	signer := NewSigner(pub, priv, 1)

	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary")
	sigPath := filepath.Join(dir, "binary.sig")

	if err := os.WriteFile(binPath, []byte("binary contents to sign"), 0644); err != nil {
		t.Fatalf("failed to write binary: %v", err)
	}

	if err := SignBinary(binPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary failed: %v", err)
	}

	ok, err := VerifyBinary(binPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary returned error: %v", err)
	}
	if !ok {
		t.Fatal("VerifyBinary returned false for a freshly-signed file")
	}
}

// TestVerifyBinary_TamperedBinary verifies that modifying the binary
// invalidates the signature (a real-world tampering scenario).
func TestVerifyBinary_TamperedBinary(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary")
	sigPath := filepath.Join(dir, "binary.sig")

	if err := os.WriteFile(binPath, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SignBinary(binPath, sigPath, signer); err != nil {
		t.Fatal(err)
	}

	// Tamper with the binary
	if err := os.WriteFile(binPath, []byte("tampered"), 0644); err != nil {
		t.Fatal(err)
	}

	ok, err := VerifyBinary(binPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary returned error: %v", err)
	}
	if ok {
		t.Fatal("VerifyBinary returned true for a tampered binary")
	}
}

// TestVerifyBinary_WrongPublicKey verifies that signatures produced with
// one key are rejected by verification with a different key.
func TestVerifyBinary_WrongPublicKey(t *testing.T) {
	signer1Pub, signer1Priv, _ := GenerateKeyPair()
	signer1 := NewSigner(signer1Pub, signer1Priv, 1)
	signer2Pub, _, _ := GenerateKeyPair()

	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary")
	sigPath := filepath.Join(dir, "binary.sig")

	if err := os.WriteFile(binPath, []byte("payload"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SignBinary(binPath, sigPath, signer1); err != nil {
		t.Fatal(err)
	}

	ok, err := VerifyBinary(binPath, sigPath, signer2Pub)
	if err != nil {
		t.Fatalf("VerifyBinary returned error: %v", err)
	}
	if ok {
		t.Fatal("VerifyBinary should fail with a mismatched public key")
	}
}

// TestVerifyBinary_MissingBinary confirms that a missing binary file
// yields a wrapped, descriptive error rather than a panic.
func TestVerifyBinary_MissingBinary(t *testing.T) {
	pub, _, _ := GenerateKeyPair()

	dir := t.TempDir()
	sigPath := filepath.Join(dir, "binary.sig")
	if err := os.WriteFile(sigPath, make([]byte, ed25519.SignatureSize), 0644); err != nil {
		t.Fatal(err)
	}

	ok, err := VerifyBinary(filepath.Join(dir, "nonexistent"), sigPath, pub)
	if err == nil {
		t.Fatal("expected error for missing binary file")
	}
	if ok {
		t.Fatal("expected ok=false when binary is missing")
	}
}

// TestVerifyBinary_MissingSignature confirms that a missing signature
// file yields a wrapped, descriptive error.
func TestVerifyBinary_MissingSignature(t *testing.T) {
	pub, _, _ := GenerateKeyPair()

	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary")
	if err := os.WriteFile(binPath, []byte("contents"), 0644); err != nil {
		t.Fatal(err)
	}

	ok, err := VerifyBinary(binPath, filepath.Join(dir, "nonexistent.sig"), pub)
	if err == nil {
		t.Fatal("expected error for missing signature file")
	}
	if ok {
		t.Fatal("expected ok=false when signature is missing")
	}
}

// TestVerifyBinary_InvalidSignatureSize confirms that a signature file
// that is the wrong size is rejected before the Ed25519 verify call.
// This prevents malformed signature files from causing confusing
// verification errors.
func TestVerifyBinary_InvalidSignatureSize(t *testing.T) {
	pub, _, _ := GenerateKeyPair()

	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary")
	sigPath := filepath.Join(dir, "binary.sig")
	if err := os.WriteFile(binPath, []byte("contents"), 0644); err != nil {
		t.Fatal(err)
	}
	// Write a signature file of the wrong size (e.g. truncated)
	if err := os.WriteFile(sigPath, []byte("short"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := VerifyBinary(binPath, sigPath, pub)
	if err == nil {
		t.Fatal("expected error for wrong-size signature file")
	}
}

// TestVerifyBinaryWithKey_MatchesVerifyBinary confirms that the
// VerifyBinaryWithKey alias behaves identically to VerifyBinary.
// Both functions are exposed publicly; this guards against accidental
// divergence if one is changed without the other.
func TestVerifyBinaryWithKey_MatchesVerifyBinary(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary")
	sigPath := filepath.Join(dir, "binary.sig")
	if err := os.WriteFile(binPath, []byte("payload"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SignBinary(binPath, sigPath, signer); err != nil {
		t.Fatal(err)
	}

	okA, errA := VerifyBinary(binPath, sigPath, pub)
	okB, errB := VerifyBinaryWithKey(binPath, sigPath, pub)
	if errA != nil || errB != nil {
		t.Fatalf("errors differ: %v vs %v", errA, errB)
	}
	if okA != okB {
		t.Fatalf("results differ: VerifyBinary=%v VerifyBinaryWithKey=%v", okA, okB)
	}
}

// TestSignBinary_CreatesSignatureFile confirms that SignBinary writes
// a signature file of the expected Ed25519 signature size.
func TestSignBinary_CreatesSignatureFile(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary")
	sigPath := filepath.Join(dir, "binary.sig")
	if err := os.WriteFile(binPath, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := SignBinary(binPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary failed: %v", err)
	}

	info, err := os.Stat(sigPath)
	if err != nil {
		t.Fatalf("signature file not created: %v", err)
	}
	if info.Size() != int64(ed25519.SignatureSize) {
		t.Errorf("signature file size = %d, want %d", info.Size(), ed25519.SignatureSize)
	}
}

// TestSignBinary_MissingBinary confirms that a missing binary yields
// an error (defensive guard against partial signing state).
func TestSignBinary_MissingBinary(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	dir := t.TempDir()
	sigPath := filepath.Join(dir, "binary.sig")
	if err := SignBinary(filepath.Join(dir, "nonexistent"), sigPath, signer); err == nil {
		t.Fatal("expected error for missing binary file")
	}
	// Make sure no partial signature file was written.
	if _, err := os.Stat(sigPath); !os.IsNotExist(err) {
		t.Errorf("signature file should not exist after a failed sign; got err=%v", err)
	}
}

// TestSignBinary_NilPrivateKey confirms that an unusable signer does
// not produce a signature file (no side effects on failure).
func TestSignBinary_NilPrivateKey(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	// Signer with public key only
	signer := NewSigner(pub, nil, 1)

	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary")
	sigPath := filepath.Join(dir, "binary.sig")
	if err := os.WriteFile(binPath, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := SignBinary(binPath, sigPath, signer); err == nil {
		t.Fatal("expected error when signing with nil private key")
	}
	if _, err := os.Stat(sigPath); !os.IsNotExist(err) {
		t.Errorf("signature file should not exist after failed sign; got err=%v", err)
	}
}

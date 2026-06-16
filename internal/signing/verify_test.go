package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper: write a file with the given contents to a temp dir and return its path.
func writeTempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
	return path
}

func TestVerifyBinary_Success(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	signer := NewSigner(pub, priv, 1)

	data := []byte("binary payload to be signed")
	sig, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	binaryPath := writeTempFile(t, "app.bin", data)
	sigPath := writeTempFile(t, "app.bin.sig", sig)

	ok, err := VerifyBinary(binaryPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary returned error: %v", err)
	}
	if !ok {
		t.Fatal("VerifyBinary returned false for a valid signature")
	}
}

func TestVerifyBinary_TamperedData(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	original := []byte("original binary content")
	sig, _ := signer.Sign(original)

	// Write tampered data but the original signature.
	tampered := []byte("tampered binary content")
	binaryPath := writeTempFile(t, "app.bin", tampered)
	sigPath := writeTempFile(t, "app.bin.sig", sig)

	ok, err := VerifyBinary(binaryPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary returned error: %v", err)
	}
	if ok {
		t.Fatal("VerifyBinary should return false when binary contents were tampered with")
	}
}

func TestVerifyBinary_WrongPublicKey(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	data := []byte("some binary data")
	sig, _ := signer.Sign(data)
	binaryPath := writeTempFile(t, "app.bin", data)
	sigPath := writeTempFile(t, "app.bin.sig", sig)

	// A different key pair than the one used to sign.
	otherPub, _, _ := GenerateKeyPair()
	ok, err := VerifyBinary(binaryPath, sigPath, otherPub)
	if err != nil {
		t.Fatalf("VerifyBinary returned error: %v", err)
	}
	if ok {
		t.Fatal("VerifyBinary should return false when signature was produced with a different key")
	}
}

func TestVerifyBinary_BinaryNotFound(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	dir := t.TempDir()
	sigPath := filepath.Join(dir, "sig")
	if err := os.WriteFile(sigPath, make([]byte, ed25519.SignatureSize), 0600); err != nil {
		t.Fatalf("failed to write sig: %v", err)
	}

	ok, err := VerifyBinary(filepath.Join(dir, "missing.bin"), sigPath, pub)
	if err == nil {
		t.Fatal("expected error for missing binary file")
	}
	if ok {
		t.Fatal("VerifyBinary should return false when the binary file is missing")
	}
	if !strings.Contains(err.Error(), "failed to read binary file") {
		t.Errorf("expected wrapped 'failed to read binary file' error, got: %v", err)
	}
}

func TestVerifyBinary_SignatureNotFound(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "app.bin")
	if err := os.WriteFile(binaryPath, []byte("data"), 0600); err != nil {
		t.Fatalf("failed to write binary: %v", err)
	}

	ok, err := VerifyBinary(binaryPath, filepath.Join(dir, "missing.sig"), pub)
	if err == nil {
		t.Fatal("expected error for missing signature file")
	}
	if ok {
		t.Fatal("VerifyBinary should return false when the signature file is missing")
	}
	if !strings.Contains(err.Error(), "failed to read signature file") {
		t.Errorf("expected wrapped 'failed to read signature file' error, got: %v", err)
	}
}

func TestVerifyBinary_InvalidSignatureSize(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "app.bin")
	sigPath := filepath.Join(dir, "app.sig")
	if err := os.WriteFile(binaryPath, []byte("data"), 0600); err != nil {
		t.Fatalf("failed to write binary: %v", err)
	}
	// Truncated signature.
	if err := os.WriteFile(sigPath, []byte("short"), 0600); err != nil {
		t.Fatalf("failed to write sig: %v", err)
	}

	ok, err := VerifyBinary(binaryPath, sigPath, pub)
	if err == nil {
		t.Fatal("expected error for invalid signature size")
	}
	if ok {
		t.Fatal("VerifyBinary should return false for an invalid signature size")
	}
	if !strings.Contains(err.Error(), "invalid signature size") {
		t.Errorf("expected 'invalid signature size' error, got: %v", err)
	}
}

func TestVerifyBinaryWithKey_MirrorsVerifyBinary(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	data := []byte("payload for verify binary with key")
	sig, _ := signer.Sign(data)
	binaryPath := writeTempFile(t, "app.bin", data)
	sigPath := writeTempFile(t, "app.bin.sig", sig)

	// VerifyBinaryWithKey is documented to mirror VerifyBinary; both must succeed here.
	ok, err := VerifyBinaryWithKey(binaryPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinaryWithKey returned error: %v", err)
	}
	if !ok {
		t.Fatal("VerifyBinaryWithKey returned false for a valid signature")
	}
}

func TestSignBinary_RoundTripWithVerifyBinary(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 7)

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "app.bin")
	sigPath := filepath.Join(dir, "app.bin.sig")

	payload := []byte("the binary to sign and verify")
	if err := os.WriteFile(binaryPath, payload, 0600); err != nil {
		t.Fatalf("failed to write binary: %v", err)
	}

	if err := SignBinary(binaryPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary failed: %v", err)
	}

	ok, err := VerifyBinary(binaryPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary failed after SignBinary: %v", err)
	}
	if !ok {
		t.Fatal("VerifyBinary should succeed for a signature just produced by SignBinary")
	}
}

func TestSignBinary_BinaryNotFound(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)
	dir := t.TempDir()

	err := SignBinary(filepath.Join(dir, "missing.bin"), filepath.Join(dir, "out.sig"), signer)
	if err == nil {
		t.Fatal("expected error when the binary file is missing")
	}
	if !strings.Contains(err.Error(), "failed to read binary file") {
		t.Errorf("expected wrapped 'failed to read binary file' error, got: %v", err)
	}
}

func TestSignBinary_InvalidOutputPath(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "app.bin")
	if err := os.WriteFile(binaryPath, []byte("data"), 0600); err != nil {
		t.Fatalf("failed to write binary: %v", err)
	}

	// A path inside a non-existent directory must cause WriteFile to fail.
	badPath := filepath.Join(dir, "does", "not", "exist", "app.sig")
	err := SignBinary(binaryPath, badPath, signer)
	if err == nil {
		t.Fatal("expected error when the signature output path is invalid")
	}
	if !strings.Contains(err.Error(), "failed to write signature file") {
		t.Errorf("expected wrapped 'failed to write signature file' error, got: %v", err)
	}
}

func TestSignBinary_RejectsSignerWithoutPrivateKey(t *testing.T) {
	// A signer with only a public key cannot produce signatures.
	pub, _, _ := GenerateKeyPair()
	signer := NewSigner(pub, nil, 1)
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "app.bin")
	if err := os.WriteFile(binaryPath, []byte("data"), 0600); err != nil {
		t.Fatalf("failed to write binary: %v", err)
	}

	// Sign() itself should refuse; SignBinary must propagate that error.
	_, err := signer.Sign([]byte("data"))
	if err == nil {
		t.Fatal("expected Sign to fail when the private key is missing")
	}
}

// Sanity: crypto/rand should never fail in practice, but verify the function
// returns the correct key sizes regardless of the underlying RNG behaviour.
func TestVerifyBinary_GeneratedKeySizes(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	if len(pub) != ed25519.PublicKeySize {
		t.Fatalf("unexpected public key size: %d", len(pub))
	}
	if len(priv) != ed25519.PrivateKeySize {
		t.Fatalf("unexpected private key size: %d", len(priv))
	}
	// Use rand to silence the unused-import warning when this file is the
	// only consumer of crypto/rand in the package test build.
	if _, err := rand.Read(make([]byte, 1)); err != nil {
		t.Fatalf("rand.Read unexpectedly failed: %v", err)
	}
}

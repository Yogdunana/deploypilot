package signing

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyBinary_Success(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test.bin")
	sigPath := filepath.Join(dir, "test.bin.sig")

	data := []byte("test binary content")
	if err := os.WriteFile(binaryPath, data, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	signer := NewSigner(pub, priv, 1)
	if err := SignBinary(binaryPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary failed: %v", err)
	}

	ok, err := VerifyBinary(binaryPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary failed: %v", err)
	}
	if !ok {
		t.Error("expected valid signature")
	}
}

func TestVerifyBinary_TamperedBinary(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test.bin")
	sigPath := filepath.Join(dir, "test.bin.sig")

	data := []byte("test binary content")
	if err := os.WriteFile(binaryPath, data, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	signer := NewSigner(pub, priv, 1)
	if err := SignBinary(binaryPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary failed: %v", err)
	}

	tamperedData := []byte("tampered content")
	if err := os.WriteFile(binaryPath, tamperedData, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	ok, err := VerifyBinary(binaryPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary failed: %v", err)
	}
	if ok {
		t.Error("expected invalid signature for tampered binary")
	}
}

func TestVerifyBinary_TamperedSignature(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test.bin")
	sigPath := filepath.Join(dir, "test.bin.sig")

	data := []byte("test binary content")
	if err := os.WriteFile(binaryPath, data, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	signer := NewSigner(pub, priv, 1)
	if err := SignBinary(binaryPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary failed: %v", err)
	}

	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	sigData[0] ^= 0xFF
	if err := os.WriteFile(sigPath, sigData, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	ok, err := VerifyBinary(binaryPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary failed: %v", err)
	}
	if ok {
		t.Error("expected invalid signature for tampered signature file")
	}
}

func TestVerifyBinary_WrongPublicKey(t *testing.T) {
	pub1, priv1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	pub2, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test.bin")
	sigPath := filepath.Join(dir, "test.bin.sig")

	data := []byte("test binary content")
	if err := os.WriteFile(binaryPath, data, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	signer := NewSigner(pub1, priv1, 1)
	if err := SignBinary(binaryPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary failed: %v", err)
	}

	ok, err := VerifyBinary(binaryPath, sigPath, pub2)
	if err != nil {
		t.Fatalf("VerifyBinary failed: %v", err)
	}
	if ok {
		t.Error("expected invalid signature with wrong public key")
	}
}

func TestVerifyBinary_MissingBinary(t *testing.T) {
	pub, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	dir := t.TempDir()
	sigPath := filepath.Join(dir, "test.bin.sig")
	if err := os.WriteFile(sigPath, []byte("dummy"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err = VerifyBinary(filepath.Join(dir, "nonexistent.bin"), sigPath, pub)
	if err == nil {
		t.Error("expected error for missing binary file")
	}
}

func TestVerifyBinary_MissingSignature(t *testing.T) {
	pub, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test.bin")
	if err := os.WriteFile(binaryPath, []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err = VerifyBinary(binaryPath, filepath.Join(dir, "nonexistent.sig"), pub)
	if err == nil {
		t.Error("expected error for missing signature file")
	}
}

func TestVerifyBinary_InvalidSignatureSize(t *testing.T) {
	pub, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test.bin")
	sigPath := filepath.Join(dir, "test.bin.sig")

	if err := os.WriteFile(binaryPath, []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := os.WriteFile(sigPath, []byte("short"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err = VerifyBinary(binaryPath, sigPath, pub)
	if err == nil {
		t.Error("expected error for invalid signature size")
	}
}

func TestVerifyBinaryWithKey(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test.bin")
	sigPath := filepath.Join(dir, "test.bin.sig")

	data := []byte("test binary content")
	if err := os.WriteFile(binaryPath, data, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	signer := NewSigner(pub, priv, 1)
	if err := SignBinary(binaryPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary failed: %v", err)
	}

	ok, err := VerifyBinaryWithKey(binaryPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinaryWithKey failed: %v", err)
	}
	if !ok {
		t.Error("expected valid signature")
	}
}

func TestSignBinary_Success(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test.bin")
	sigPath := filepath.Join(dir, "test.bin.sig")

	data := []byte("test binary content")
	if err := os.WriteFile(binaryPath, data, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	signer := NewSigner(pub, priv, 1)
	if err := SignBinary(binaryPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary failed: %v", err)
	}

	if _, err := os.Stat(sigPath); os.IsNotExist(err) {
		t.Error("signature file not created")
	}

	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(sigData) != ed25519.SignatureSize {
		t.Errorf("expected signature size %d, got %d", ed25519.SignatureSize, len(sigData))
	}
}

func TestSignBinary_MissingBinary(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	signer := NewSigner(pub, priv, 1)
	err = SignBinary("/nonexistent/path/to/bin", "/tmp/test.sig", signer)
	if err == nil {
		t.Error("expected error for missing binary file")
	}
}

func TestVerifySelf(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	dir := t.TempDir()
	sigPath := filepath.Join(dir, "test.sig")

	signer := NewSigner(pub, priv, 1)
	data := []byte("test data")
	sig, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if err := os.WriteFile(sigPath, sig, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	ok, err := VerifySelf(sigPath, pub)
	if err != nil {
		t.Logf("VerifySelf error (expected for test binary): %v", err)
	}
	if ok {
		t.Logf("VerifySelf returned true (unexpected for test binary)")
	}
}
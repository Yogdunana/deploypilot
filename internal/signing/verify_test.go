package signing

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyBinary_ValidSignature(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test-binary")
	signaturePath := filepath.Join(dir, "test-binary.sig")

	data := []byte("binary content here")
	if err := os.WriteFile(binaryPath, data, 0644); err != nil {
		t.Fatalf("failed to write binary: %v", err)
	}

	signer := NewSigner(pub, priv, 1)
	sig, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if err := os.WriteFile(signaturePath, sig, 0640); err != nil {
		t.Fatalf("failed to write signature: %v", err)
	}

	valid, err := VerifyBinary(binaryPath, signaturePath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary failed: %v", err)
	}
	if !valid {
		t.Error("expected valid signature")
	}
}

func TestVerifyBinary_InvalidSignature(t *testing.T) {
	pub, _, _ := GenerateKeyPair()

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test-binary")
	signaturePath := filepath.Join(dir, "test-binary.sig")

	data := []byte("binary content here")
	os.WriteFile(binaryPath, data, 0644)

	// Sign with a different key pair
	wrongPub, wrongPriv, _ := GenerateKeyPair()
	wrongSigner := NewSigner(wrongPub, wrongPriv, 1)
	sig, _ := wrongSigner.Sign(data)
	os.WriteFile(signaturePath, sig, 0640)

	valid, err := VerifyBinary(binaryPath, signaturePath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary failed: %v", err)
	}
	if valid {
		t.Error("expected invalid signature for wrong key")
	}
}

func TestVerifyBinary_TamperedBinary(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test-binary")
	signaturePath := filepath.Join(dir, "test-binary.sig")

	data := []byte("original binary content")
	os.WriteFile(binaryPath, data, 0644)

	signer := NewSigner(pub, priv, 1)
	sig, _ := signer.Sign(data)
	os.WriteFile(signaturePath, sig, 0640)

	// Tamper with the binary after signing
	os.WriteFile(binaryPath, []byte("tampered binary content"), 0644)

	valid, err := VerifyBinary(binaryPath, signaturePath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary failed: %v", err)
	}
	if valid {
		t.Error("expected invalid signature for tampered binary")
	}
}

func TestVerifyBinary_BinaryNotFound(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	dir := t.TempDir()
	sigPath := filepath.Join(dir, "test.sig")
	os.WriteFile(sigPath, make([]byte, ed25519.SignatureSize), 0640)

	_, err := VerifyBinary("/nonexistent/binary", sigPath, pub)
	if err == nil {
		t.Error("expected error for non-existent binary")
	}
}

func TestVerifyBinary_SignatureNotFound(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary")
	os.WriteFile(binPath, []byte("data"), 0644)

	_, err := VerifyBinary(binPath, "/nonexistent/sig", pub)
	if err == nil {
		t.Error("expected error for non-existent signature")
	}
}

func TestVerifyBinary_InvalidSignatureSize(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary")
	sigPath := filepath.Join(dir, "binary.sig")

	os.WriteFile(binPath, []byte("data"), 0644)
	os.WriteFile(sigPath, []byte("too short"), 0640)

	valid, err := VerifyBinary(binPath, sigPath, pub)
	if err == nil {
		t.Error("expected error for invalid signature size")
	}
	if valid {
		t.Error("expected invalid for wrong signature size")
	}
}

func TestVerifyBinaryWithKey_ValidSignature(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test-binary")
	signaturePath := filepath.Join(dir, "test-binary.sig")

	data := []byte("test data for VerifyBinaryWithKey")
	os.WriteFile(binaryPath, data, 0644)

	signer := NewSigner(pub, priv, 1)
	sig, _ := signer.Sign(data)
	os.WriteFile(signaturePath, sig, 0640)

	valid, err := VerifyBinaryWithKey(binaryPath, signaturePath, pub)
	if err != nil {
		t.Fatalf("VerifyBinaryWithKey failed: %v", err)
	}
	if !valid {
		t.Error("expected valid signature")
	}
}

func TestSignBinary(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "test-binary")
	signaturePath := filepath.Join(dir, "test-binary.sig")

	data := []byte("binary to sign")
	os.WriteFile(binaryPath, data, 0644)

	signer := NewSigner(pub, priv, 1)
	err := SignBinary(binaryPath, signaturePath, signer)
	if err != nil {
		t.Fatalf("SignBinary failed: %v", err)
	}

	// Verify the signature file was created
	if _, err := os.Stat(signaturePath); os.IsNotExist(err) {
		t.Error("signature file was not created")
	}

	// Verify the signature is valid
	valid, err := VerifyBinary(binaryPath, signaturePath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary failed: %v", err)
	}
	if !valid {
		t.Error("expected generated signature to be valid")
	}
}

func TestSignBinary_BinaryNotFound(t *testing.T) {
	_, priv, _ := GenerateKeyPair()
	signer := NewSigner(nil, priv, 1)

	err := SignBinary("/nonexistent/binary", "/tmp/sig", signer)
	if err == nil {
		t.Error("expected error for non-existent binary")
	}
}

func TestVerifySelf(t *testing.T) {
	pub, _, _ := GenerateKeyPair()

	// Create a fake "binary" to sign
	dir := t.TempDir()
	sigPath := filepath.Join(dir, "self.sig")

	// We can't easily test VerifySelf because it uses os.Executable(),
	// but we can at least verify it doesn't panic
	// This test is limited - it will fail to verify because we're
	// not actually signing the current executable
	_, _ = VerifySelf(sigPath, pub)
}

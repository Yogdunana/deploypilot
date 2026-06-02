package signing

import (
	"bytes"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVerifyBinary_Success covers the happy path: signing a file with
// Signer.Sign and verifying the resulting signature via VerifyBinary
// against the same public key must return (true, nil).
func TestVerifyBinary_Success(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	signer := NewSigner(pub, priv, 1)

	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary.bin")
	sigPath := filepath.Join(dir, "binary.sig")
	if err := os.WriteFile(binPath, []byte("the binary content"), 0644); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	if err := SignBinary(binPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary: %v", err)
	}

	ok, err := VerifyBinary(binPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary: %v", err)
	}
	if !ok {
		t.Error("VerifyBinary returned false for a freshly signed file")
	}
}

// TestVerifyBinary_TamperedBinary is the critical security test: any
// single-byte change to the binary must cause verification to fail.
func TestVerifyBinary_TamperedBinary(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary.bin")
	sigPath := filepath.Join(dir, "binary.sig")
	if err := os.WriteFile(binPath, []byte("original content"), 0644); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	if err := SignBinary(binPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary: %v", err)
	}

	// Modify the binary.
	if err := os.WriteFile(binPath, []byte("tampered content"), 0644); err != nil {
		t.Fatalf("rewrite binary: %v", err)
	}

	ok, err := VerifyBinary(binPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary: %v", err)
	}
	if ok {
		t.Error("VerifyBinary must reject a tampered binary")
	}
}

// TestVerifyBinary_WrongPublicKey confirms that verification fails when
// the wrong public key is supplied.
func TestVerifyBinary_WrongPublicKey(t *testing.T) {
	pub1, priv1, _ := GenerateKeyPair()
	pub2, _, _ := GenerateKeyPair()
	signer := NewSigner(pub1, priv1, 1)

	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary.bin")
	sigPath := filepath.Join(dir, "binary.sig")
	if err := os.WriteFile(binPath, []byte("hello"), 0644); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	if err := SignBinary(binPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary: %v", err)
	}

	ok, err := VerifyBinary(binPath, sigPath, pub2)
	if err != nil {
		t.Fatalf("VerifyBinary: %v", err)
	}
	if ok {
		t.Error("VerifyBinary must reject signature under a different public key")
	}
}

// TestVerifyBinary_MissingBinary verifies the error path when the
// binary file does not exist.
func TestVerifyBinary_MissingBinary(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	_, err := VerifyBinary("/no/such/binary", "/no/such/sig", pub)
	if err == nil {
		t.Fatal("expected error for missing binary, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read binary") {
		t.Errorf("error %q should mention binary read failure", err)
	}
}

// TestVerifyBinary_MissingSignature verifies the error path when the
// signature file does not exist.
func TestVerifyBinary_MissingSignature(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary.bin")
	if err := os.WriteFile(binPath, []byte("hello"), 0644); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	_, err := VerifyBinary(binPath, filepath.Join(dir, "missing.sig"), pub)
	if err == nil {
		t.Fatal("expected error for missing signature, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read signature") {
		t.Errorf("error %q should mention signature read failure", err)
	}
}

// TestVerifyBinary_InvalidSignatureSize covers the size validation
// path: a non-empty signature that is not exactly ed25519.SignatureSize
// bytes must be rejected with a descriptive error *before* reaching the
// crypto layer.
func TestVerifyBinary_InvalidSignatureSize(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary.bin")
	sigPath := filepath.Join(dir, "binary.sig")
	if err := os.WriteFile(binPath, []byte("hello"), 0644); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	// Too short
	if err := os.WriteFile(sigPath, []byte{0x01, 0x02}, 0644); err != nil {
		t.Fatalf("write sig: %v", err)
	}
	_, err := VerifyBinary(binPath, sigPath, pub)
	if err == nil {
		t.Fatal("expected error for short signature, got nil")
	}
	if !strings.Contains(err.Error(), "invalid signature size") {
		t.Errorf("error %q should mention invalid signature size", err)
	}

	// Too long
	if err := os.WriteFile(sigPath, bytes.Repeat([]byte{0x01}, ed25519.SignatureSize+1), 0644); err != nil {
		t.Fatalf("write sig: %v", err)
	}
	_, err = VerifyBinary(binPath, sigPath, pub)
	if err == nil {
		t.Fatal("expected error for long signature, got nil")
	}
	if !strings.Contains(err.Error(), "invalid signature size") {
		t.Errorf("error %q should mention invalid signature size", err)
	}

	// Empty signature file
	if err := os.WriteFile(sigPath, []byte{}, 0644); err != nil {
		t.Fatalf("write sig: %v", err)
	}
	_, err = VerifyBinary(binPath, sigPath, pub)
	if err == nil {
		t.Fatal("expected error for empty signature, got nil")
	}
}

// TestVerifyBinaryWithKey_MatchesVerifyBinary sanity-checks that
// VerifyBinaryWithKey (which is a near-duplicate of VerifyBinary)
// produces the same outcome on the same inputs.
func TestVerifyBinaryWithKey_MatchesVerifyBinary(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary.bin")
	sigPath := filepath.Join(dir, "binary.sig")
	if err := os.WriteFile(binPath, []byte("payload"), 0644); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	if err := SignBinary(binPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary: %v", err)
	}

	ok1, err1 := VerifyBinary(binPath, sigPath, pub)
	ok2, err2 := VerifyBinaryWithKey(binPath, sigPath, pub)
	if err1 != nil || err2 != nil {
		t.Fatalf("errors: VerifyBinary=%v VerifyBinaryWithKey=%v", err1, err2)
	}
	if ok1 != ok2 {
		t.Errorf("VerifyBinary=%v VerifyBinaryWithKey=%v – results must match", ok1, ok2)
	}
	if !ok1 {
		t.Error("expected verification to succeed for a freshly signed file")
	}
}

// TestVerifySelf_NotRunnableOutsideTest enforces that calling
// VerifySelf on the test binary itself fails gracefully (no panic, an
// error or a negative result) because the test binary was not signed.
//
// The test cannot easily produce a valid self-signature, but it can at
// least lock in the contract: VerifySelf does not crash and returns
// (false, _) when the .sig file is missing.
func TestVerifySelf_MissingSignatureFile(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	// Pass a signature path that does not exist; VerifySelf resolves
	// os.Executable() internally so we just supply a bogus sig path.
	ok, err := VerifySelf("/no/such/sig", pub)
	if err == nil {
		t.Fatal("expected error from VerifySelf when signature is missing")
	}
	if ok {
		t.Error("VerifySelf must return false when the signature file is missing")
	}
}

// TestSignBinary_FileErrors covers the three file-related error paths
// of SignBinary: missing binary, missing parent directory for sig.
func TestSignBinary_MissingBinary(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)
	dir := t.TempDir()
	err := SignBinary(filepath.Join(dir, "no_such_binary"),
		filepath.Join(dir, "sig"), signer)
	if err == nil {
		t.Fatal("expected error for missing binary, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read binary") {
		t.Errorf("error %q should mention binary read failure", err)
	}
}

func TestSignBinary_NilSignerPrivateKey(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	// Signer constructed with no private key – SignBinary should still
	// write the signature (because Signer.Sign does the key check), but
	// actually Signer.Sign returns an error if the private key is nil.
	// We document and lock in that behavior.
	signer := &Signer{publicKey: pub, privateKey: nil, keyVersion: 0}
	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary.bin")
	if err := os.WriteFile(binPath, []byte("hello"), 0644); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	err := SignBinary(binPath, filepath.Join(dir, "sig"), signer)
	if err == nil {
		t.Fatal("expected error when signer has no private key, got nil")
	}
	if !strings.Contains(err.Error(), "failed to sign binary") {
		t.Errorf("error %q should mention signing failure", err)
	}
}

// TestSignBinary_SignatureFileCreatedWithSecureMode checks the
// signature file is written with 0640 permissions, matching the
// documented contract.
func TestSignBinary_SignatureFileCreatedWithSecureMode(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root – file mode bits are not enforced")
	}
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)
	dir := t.TempDir()
	binPath := filepath.Join(dir, "binary.bin")
	sigPath := filepath.Join(dir, "binary.sig")
	if err := os.WriteFile(binPath, []byte("content"), 0644); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	if err := SignBinary(binPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary: %v", err)
	}
	info, err := os.Stat(sigPath)
	if err != nil {
		t.Fatalf("stat sig: %v", err)
	}
	if got := info.Mode().Perm(); got != 0640 {
		t.Errorf("signature file mode = %o, want 0640", got)
	}
}

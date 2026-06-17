package signing

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile is a small helper that writes data to a temp file and
// returns its path. The file is automatically cleaned up at test end.
func writeFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

// ===================== VerifyBinary =====================

func TestVerifyBinary_ValidSignature(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	signer := NewSigner(pub, priv, 1)

	binary := []byte("binary payload that we want to sign")
	binPath := writeFile(t, "binary", binary)
	sig, err := signer.Sign(binary)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	sigPath := writeFile(t, "binary.sig", sig)

	ok, err := VerifyBinary(binPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary: %v", err)
	}
	if !ok {
		t.Error("VerifyBinary returned false for a valid signature")
	}
}

func TestVerifyBinary_TamperedBinary(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	original := []byte("original binary")
	sig, _ := signer.Sign(original)
	binPath := writeFile(t, "binary", []byte("TAMPERED binary"))
	sigPath := writeFile(t, "binary.sig", sig)

	ok, err := VerifyBinary(binPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary returned error: %v", err)
	}
	if ok {
		t.Error("VerifyBinary should return false when binary has been tampered with")
	}
}

func TestVerifyBinary_TamperedSignature(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	binary := []byte("binary")
	sig, _ := signer.Sign(binary)
	// Flip a single bit in the signature.
	sig[0] ^= 0x01

	binPath := writeFile(t, "binary", binary)
	sigPath := writeFile(t, "binary.sig", sig)

	ok, err := VerifyBinary(binPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary returned error: %v", err)
	}
	if ok {
		t.Error("VerifyBinary should return false for a tampered signature")
	}
}

func TestVerifyBinary_WrongKey(t *testing.T) {
	_, priv, _ := GenerateKeyPair()
	signer := NewSigner(nil, priv, 1)
	otherPub, _, _ := GenerateKeyPair()

	binary := []byte("binary")
	sig, _ := signer.Sign(binary)
	binPath := writeFile(t, "binary", binary)
	sigPath := writeFile(t, "binary.sig", sig)

	ok, err := VerifyBinary(binPath, sigPath, otherPub)
	if err != nil {
		t.Fatalf("VerifyBinary returned error: %v", err)
	}
	if ok {
		t.Error("VerifyBinary should return false for the wrong public key")
	}
}

func TestVerifyBinary_InvalidSignatureSize(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	binPath := writeFile(t, "binary", []byte("binary"))
	sigPath := writeFile(t, "binary.sig", []byte("too short"))

	_, err := VerifyBinary(binPath, sigPath, pub)
	if err == nil {
		t.Fatal("expected error for invalid signature size")
	}
	if !strings.Contains(err.Error(), "invalid signature size") {
		t.Errorf("expected 'invalid signature size' error, got: %v", err)
	}
}

func TestVerifyBinary_MissingBinary(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	sigPath := writeFile(t, "binary.sig", make([]byte, ed25519.SignatureSize))

	_, err := VerifyBinary("/nonexistent/binary", sigPath, pub)
	if err == nil {
		t.Fatal("expected error for missing binary file")
	}
	if !strings.Contains(err.Error(), "read binary") {
		t.Errorf("expected read-binary error, got: %v", err)
	}
}

func TestVerifyBinary_MissingSignature(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	binPath := writeFile(t, "binary", []byte("data"))

	_, err := VerifyBinary(binPath, "/nonexistent/sig", pub)
	if err == nil {
		t.Fatal("expected error for missing signature file")
	}
	if !strings.Contains(err.Error(), "read signature") {
		t.Errorf("expected read-signature error, got: %v", err)
	}
}

// ===================== VerifyBinaryWithKey =====================

func TestVerifyBinaryWithKey_DelegatesToEd25519(t *testing.T) {
	// VerifyBinaryWithKey should behave identically to VerifyBinary.
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	binary := []byte("verify with key path")
	sig, _ := signer.Sign(binary)
	binPath := writeFile(t, "binary", binary)
	sigPath := writeFile(t, "binary.sig", sig)

	ok, err := VerifyBinaryWithKey(binPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinaryWithKey: %v", err)
	}
	if !ok {
		t.Error("VerifyBinaryWithKey returned false for a valid signature")
	}
}

func TestVerifyBinaryWithKey_RejectsInvalidSize(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	binPath := writeFile(t, "binary", []byte("x"))
	sigPath := writeFile(t, "sig", []byte("short"))

	_, err := VerifyBinaryWithKey(binPath, sigPath, pub)
	if err == nil {
		t.Fatal("expected error for invalid signature size")
	}
}

// ===================== VerifySelf =====================

func TestVerifySelf_ValidSignature(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	// Sign the running test binary as a stand-in for "self".
	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	data, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("read exec: %v", err)
	}
	sig, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	sigPath := writeFile(t, "self.sig", sig)

	ok, err := VerifySelf(sigPath, pub)
	if err != nil {
		t.Fatalf("VerifySelf: %v", err)
	}
	if !ok {
		t.Error("VerifySelf returned false for a valid signature over the current binary")
	}
}

func TestVerifySelf_RejectsWrongSignature(t *testing.T) {
	pub, _, _ := GenerateKeyPair()
	sigPath := writeFile(t, "self.sig", make([]byte, ed25519.SignatureSize))

	ok, err := VerifySelf(sigPath, pub)
	if err != nil {
		t.Fatalf("VerifySelf: %v", err)
	}
	if ok {
		t.Error("VerifySelf returned true for a random signature")
	}
}

// ===================== SignBinary =====================

func TestSignBinary_AndVerifyRoundTrip(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)

	binary := []byte("a binary that should round-trip through SignBinary")
	binPath := writeFile(t, "rt-binary", binary)
	sigPath := filepath.Join(t.TempDir(), "rt-binary.sig")

	if err := SignBinary(binPath, sigPath, signer); err != nil {
		t.Fatalf("SignBinary: %v", err)
	}

	// File permissions: the signature file should be readable by the owner
	// and group (0640 per SignBinary), but not by others.
	info, err := os.Stat(sigPath)
	if err != nil {
		t.Fatalf("stat sig: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Errorf("signature file mode = %o, want 0640", got)
	}

	ok, err := VerifyBinary(binPath, sigPath, pub)
	if err != nil {
		t.Fatalf("VerifyBinary: %v", err)
	}
	if !ok {
		t.Error("verification after SignBinary round-trip failed")
	}
}

func TestSignBinary_MissingBinary(t *testing.T) {
	_, priv, _ := GenerateKeyPair()
	signer := NewSigner(nil, priv, 1)

	sigPath := filepath.Join(t.TempDir(), "missing.sig")
	err := SignBinary("/nonexistent/binary", sigPath, signer)
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
	if !strings.Contains(err.Error(), "read binary") {
		t.Errorf("expected read-binary error, got: %v", err)
	}
}

func TestSignBinary_NilPrivateKey(t *testing.T) {
	signer := NewSigner(nil, nil, 1)
	binPath := writeFile(t, "binary", []byte("x"))
	sigPath := filepath.Join(t.TempDir(), "x.sig")

	err := SignBinary(binPath, sigPath, signer)
	if err == nil {
		t.Fatal("expected error when signer has no private key")
	}
	if !strings.Contains(err.Error(), "sign") {
		t.Errorf("expected sign error, got: %v", err)
	}
}

func TestSignBinary_DestinationUnwritable(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)
	binPath := writeFile(t, "binary", []byte("x"))

	// Point at a path that cannot be created (a non-existent directory).
	bogus := filepath.Join(t.TempDir(), "does", "not", "exist", "sig")
	if err := SignBinary(binPath, bogus, signer); err == nil {
		t.Fatal("expected error for unwritable signature destination")
	}
}

// ===================== Deterministic signature test =====================

func TestVerifyBinary_DeterministicForSameInput(t *testing.T) {
	// Ed25519 signatures are deterministic for a given key+message.
	// This guards against accidentally introducing randomness later.
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 1)
	binary := []byte("deterministic binary")
	sig, _ := signer.Sign(binary)
	sig2, _ := signer.Sign(binary)
	if string(sig) != string(sig2) {
		t.Error("Ed25519 signature should be deterministic for the same input")
	}
}

package signing

import (
	"crypto/ed25519"
	"fmt"
	"os"
)

// VerifyBinary reads a binary file and its signature file, then verifies the signature.
func VerifyBinary(binaryPath, signaturePath string) (bool, error) {
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return false, fmt.Errorf("failed to read binary file: %w", err)
	}

	sig, err := os.ReadFile(signaturePath)
	if err != nil {
		return false, fmt.Errorf("failed to read signature file: %w", err)
	}

	if len(sig) != ed25519.SignatureSize {
		return false, fmt.Errorf("invalid signature size: got %d, want %d", len(sig), ed25519.SignatureSize)
	}

	return ed25519.Verify(data, sig), nil
}

// VerifyBinaryWithKey reads a binary file and its signature file, then verifies
// the signature against the provided public key.
func VerifyBinaryWithKey(binaryPath, signaturePath string, publicKey ed25519.PublicKey) (bool, error) {
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return false, fmt.Errorf("failed to read binary file: %w", err)
	}

	sig, err := os.ReadFile(signaturePath)
	if err != nil {
		return false, fmt.Errorf("failed to read signature file: %w", err)
	}

	if len(sig) != ed25519.SignatureSize {
		return false, fmt.Errorf("invalid signature size: got %d, want %d", len(sig), ed25519.SignatureSize)
	}

	return ed25519.Verify(publicKey, data, sig), nil
}

// VerifySelf verifies the currently running binary against a signature file.
// The signature file is expected to be at binaryPath + ".sig".
func VerifySelf(signaturePath string) (bool, error) {
	execPath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("failed to get executable path: %w", err)
	}
	return VerifyBinary(execPath, signaturePath)
}

// SignBinary reads a binary file and writes its Ed25519 signature to signaturePath.
func SignBinary(binaryPath, signaturePath string, signer *Signer) error {
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to read binary file: %w", err)
	}

	signature, err := signer.Sign(data)
	if err != nil {
		return fmt.Errorf("failed to sign binary: %w", err)
	}

	if err := os.WriteFile(signaturePath, signature, 0644); err != nil {
		return fmt.Errorf("failed to write signature file: %w", err)
	}

	return nil
}

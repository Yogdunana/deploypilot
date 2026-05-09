package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
)

const (
	pemTypePublicKey  = "ED25519 PUBLIC KEY"
	pemTypePrivateKey = "ED25519 PRIVATE KEY"
)

// Signer holds an Ed25519 key pair and a version identifier.
type Signer struct {
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
	keyVersion int
}

// GenerateKeyPair creates a new Ed25519 key pair.
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

// NewSigner creates a Signer from raw public/private key bytes and a version.
func NewSigner(publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey, keyVersion int) *Signer {
	return &Signer{
		publicKey:  publicKey,
		privateKey: privateKey,
		keyVersion: keyVersion,
	}
}

// Sign signs the given data using Ed25519 and returns the signature.
func (s *Signer) Sign(data []byte) ([]byte, error) {
	if s.privateKey == nil {
		return nil, fmt.Errorf("private key is not set")
	}
	signature := ed25519.Sign(s.privateKey, data)
	return signature, nil
}

// Verify checks whether the signature is valid for the given data.
func (s *Signer) Verify(data, signature []byte) bool {
	if s.publicKey == nil {
		return false
	}
	return ed25519.Verify(s.publicKey, data, signature)
}

// PublicKeyBytes returns the raw bytes of the public key.
func (s *Signer) PublicKeyBytes() []byte {
	return s.publicKey
}

// PrivateKeyBytes returns the raw bytes of the private key (seed).
func (s *Signer) PrivateKeyBytes() []byte {
	if s.privateKey == nil {
		return nil
	}
	// ed25519.PrivateKey is the seed prefixed with the public key.
	// Return only the 32-byte seed for storage.
	return s.privateKey.Seed()
}

// Fingerprint returns the first 16 hex characters of the SHA256 hash of the public key.
func (s *Signer) Fingerprint() string {
	if s.publicKey == nil {
		return ""
	}
	hash := sha256.Sum256(s.publicKey)
	return hex.EncodeToString(hash[:])[:16]
}

// KeyVersion returns the key version.
func (s *Signer) KeyVersion() int {
	return s.keyVersion
}

// LoadKeys reads PEM-encoded public and private keys from the given file paths.
func LoadKeys(publicKeyPath, privateKeyPath string) (*Signer, error) {
	pubBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	privBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	pubBlock, _ := pem.Decode(pubBytes)
	if pubBlock == nil {
		return nil, fmt.Errorf("failed to decode public key PEM")
	}
	if len(pubBlock.Bytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: got %d, want %d", len(pubBlock.Bytes), ed25519.PublicKeySize)
	}

	privBlock, _ := pem.Decode(privBytes)
	if privBlock == nil {
		return nil, fmt.Errorf("failed to decode private key PEM")
	}
	if len(privBlock.Bytes) != ed25519.SeedSize {
		return nil, fmt.Errorf("invalid private key seed size: got %d, want %d", len(privBlock.Bytes), ed25519.SeedSize)
	}

	privateKey := ed25519.NewKeyFromSeed(privBlock.Bytes)

	return &Signer{
		publicKey:  pubBlock.Bytes,
		privateKey: privateKey,
	}, nil
}

// SaveKeys writes the public and private keys to PEM files at the given paths.
func (s *Signer) SaveKeys(publicKeyPath, privateKeyPath string) error {
	if s.publicKey == nil {
		return fmt.Errorf("public key is not set")
	}
	if s.privateKey == nil {
		return fmt.Errorf("private key is not set")
	}

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  pemTypePublicKey,
		Bytes: s.publicKey,
	})
	if err := os.WriteFile(publicKeyPath, pubPEM, 0644); err != nil {
		return fmt.Errorf("failed to write public key file: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  pemTypePrivateKey,
		Bytes: s.privateKey.Seed(),
	})
	if err := os.WriteFile(privateKeyPath, privPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key file: %w", err)
	}

	return nil
}

package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SSHKeyPair represents an SSH key pair stored in the database.
type SSHKeyPair struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:200" json:"name"`
	PublicKey   string    `gorm:"type:text" json:"public_key"`
	PrivateKey  string    `gorm:"type:text" json:"-"` // never exposed in JSON
	Fingerprint string    `gorm:"size:100" json:"fingerprint"`
	KeyType     string    `gorm:"size:20;default:rsa" json:"key_type"`
	KeyBits     int       `gorm:"default:4096" json:"key_bits"`
	CreatedAt   time.Time `json:"created_at"`
}

func (SSHKeyPair) TableName() string { return "ssh_key_pairs" }

// SSHAuthorization represents an authorized key on a server.
type SSHAuthorization struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	KeyPairID string    `gorm:"index" json:"key_pair_id"`
	ServerID  string    `gorm:"index" json:"server_id"`
	User      string    `gorm:"size:100;default:root" json:"user"`
	CreatedAt time.Time `json:"created_at"`
}

func (SSHAuthorization) TableName() string { return "ssh_authorizations" }

// SSHService manages SSH key pairs and server authorizations.
type SSHService struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewSSHService creates a new SSHService.
func NewSSHService(db *gorm.DB) *SSHService {
	return &SSHService{
		db:     db,
		logger: slog.Default(),
	}
}

// ========== Key Pair Management ==========

// GenerateKeyPair generates a new RSA SSH key pair.
func (s *SSHService) GenerateKeyPair(name string, bits int) (*SSHKeyPair, error) {
	if bits == 0 {
		bits = 4096
	}
	if bits != 2048 && bits != 4096 {
		return nil, fmt.Errorf("unsupported key size: %d (use 2048 or 4096)", bits)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Encode private key to PEM
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Generate public key in OpenSSH format
	pubKey, err := generateOpenSSHPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	fingerprint := generateFingerprint(pubKey)

	keyPair := &SSHKeyPair{
		ID:          fmt.Sprintf("sshkey-%d", time.Now().UnixNano()),
		Name:        name,
		PublicKey:   pubKey,
		PrivateKey:  string(privPEM),
		Fingerprint: fingerprint,
		KeyType:     "rsa",
		KeyBits:     bits,
	}

	if err := s.db.Create(keyPair).Error; err != nil {
		return nil, fmt.Errorf("failed to save key pair: %w", err)
	}

	return keyPair, nil
}

// ImportPublicKey imports an existing public key.
func (s *SSHService) ImportPublicKey(name, publicKey string) (*SSHKeyPair, error) {
	if !strings.HasPrefix(publicKey, "ssh-") {
		return nil, fmt.Errorf("invalid SSH public key format (must start with 'ssh-')")
	}

	fingerprint := generateFingerprint(publicKey)

	keyPair := &SSHKeyPair{
		ID:          fmt.Sprintf("sshkey-%d", time.Now().UnixNano()),
		Name:        name,
		PublicKey:   publicKey,
		Fingerprint: fingerprint,
		KeyType:     strings.SplitN(publicKey, " ", 2)[0],
		KeyBits:     0,
	}

	if err := s.db.Create(keyPair).Error; err != nil {
		return nil, fmt.Errorf("failed to save public key: %w", err)
	}

	return keyPair, nil
}

// ListKeyPairs returns all stored SSH key pairs.
func (s *SSHService) ListKeyPairs() ([]SSHKeyPair, error) {
	var keys []SSHKeyPair
	err := s.db.Order("created_at DESC").Find(&keys).Error
	return keys, err
}

// GetKeyPair returns a single key pair by ID.
func (s *SSHService) GetKeyPair(id string) (*SSHKeyPair, error) {
	var key SSHKeyPair
	if err := s.db.Where("id = ?", id).First(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

// DeleteKeyPair deletes a key pair and all its authorizations.
func (s *SSHService) DeleteKeyPair(id string) error {
	// Remove authorizations first
	s.db.Where("key_pair_id = ?", id).Delete(&SSHAuthorization{})
	return s.db.Delete(&SSHKeyPair{}, "id = ?", id).Error
}

// ========== Authorization Management ==========

// AuthorizeKey deploys a public key to a server's authorized_keys.
func (s *SSHService) AuthorizeKey(ctx context.Context, keyPairID, serverID, user string) error {
	keyPair, err := s.GetKeyPair(keyPairID)
	if err != nil {
		return fmt.Errorf("key pair not found: %w", err)
	}

	if user == "" {
		user = "root"
	}

	// Check if already authorized
	var existing SSHAuthorization
	if s.db.Where("key_pair_id = ? AND server_id = ? AND user = ?", keyPairID, serverID, user).First(&existing).Error == nil {
		return fmt.Errorf("key already authorized on this server for user %s", user)
	}

	// Deploy via SSH
	exec, err := s.getExecutor(ctx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = exec.Close() }()

	// Add to authorized_keys
	cmd := fmt.Sprintf("mkdir -p /home/%s/.ssh && echo %s >> /home/%s/.ssh/authorized_keys && chmod 600 /home/%s/.ssh/authorized_keys && chown %s:%s /home/%s/.ssh/authorized_keys",
		shellQuote(user), shellQuote(keyPair.PublicKey), shellQuote(user), shellQuote(user), user, user, shellQuote(user))

	// For root user, use /root/.ssh
	if user == "root" {
		cmd = fmt.Sprintf("mkdir -p /root/.ssh && echo %s >> /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys",
			shellQuote(keyPair.PublicKey))
	}

	if _, err := exec.RunCommand(ctx, cmd); err != nil {
		return fmt.Errorf("failed to deploy key: %w", err)
	}

	// Record authorization
	auth := SSHAuthorization{
		ID:        fmt.Sprintf("sshauth-%d", time.Now().UnixNano()),
		KeyPairID: keyPairID,
		ServerID:  serverID,
		User:      user,
	}
	return s.db.Create(&auth).Error
}

// RevokeKey removes a public key from a server's authorized_keys.
func (s *SSHService) RevokeKey(ctx context.Context, keyPairID, serverID, user string) error {
	keyPair, err := s.GetKeyPair(keyPairID)
	if err != nil {
		return fmt.Errorf("key pair not found: %w", err)
	}

	if user == "" {
		user = "root"
	}

	exec, err := s.getExecutor(ctx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = exec.Close() }()

	// Extract the key comment for matching
	parts := strings.Fields(keyPair.PublicKey)
	keyComment := ""
	if len(parts) >= 3 {
		keyComment = parts[2]
	}

	// Remove from authorized_keys using fingerprint or comment
	sshDir := fmt.Sprintf("/home/%s/.ssh", user)
	if user == "root" {
		sshDir = "/root/.ssh"
	}

	cmd := fmt.Sprintf("sed -i '/%s/d' %s/authorized_keys 2>/dev/null || true", shellQuote(keyComment), shellQuote(sshDir))
	if _, err := exec.RunCommand(ctx, cmd); err != nil {
		return fmt.Errorf("failed to revoke key: %w", err)
	}

	// Remove authorization record
	return s.db.Where("key_pair_id = ? AND server_id = ? AND user = ?", keyPairID, serverID, user).
		Delete(&SSHAuthorization{}).Error
}

// ListAuthorizations returns all authorizations for a server.
func (s *SSHService) ListAuthorizations(serverID string) ([]SSHAuthorization, error) {
	var auths []SSHAuthorization
	err := s.db.Where("server_id = ?", serverID).Order("created_at DESC").Find(&auths).Error
	return auths, err
}

// ListAuthorizationsByKey returns all servers a key is authorized on.
func (s *SSHService) ListAuthorizationsByKey(keyPairID string) ([]SSHAuthorization, error) {
	var auths []SSHAuthorization
	err := s.db.Where("key_pair_id = ?", keyPairID).Order("created_at DESC").Find(&auths).Error
	return auths, err
}

// ========== Helpers ==========

func (s *SSHService) getExecutor(ctx context.Context, serverID string) (*sshClientExecutor, error) {
	b := &Bridge{DB: s.db}
	return b.getRemoteExecutor(ctx, serverID)
}

// generateFingerprint creates a simple fingerprint from a public key.
func generateFingerprint(publicKey string) string {
	// Use the comment as a simple identifier
	parts := strings.Fields(publicKey)
	if len(parts) >= 3 {
		return parts[2]
	}
	// Fallback: use first 16 chars of the base64 portion
	if len(parts) >= 2 && len(parts[1]) >= 16 {
		return "fp:" + parts[1][:16]
	}
	return "unknown"
}

// generateOpenSSHPublicKey generates an OpenSSH format public key string.
func generateOpenSSHPublicKey(pub *rsa.PublicKey) (string, error) {
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}

	// Simple OpenSSH public key format
	// ssh-rsa <base64> <comment>
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	// Extract base64 content from PEM
	lines := strings.Split(strings.TrimSpace(string(pubPEM)), "\n")
	var b64Content strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(line, "-----") {
			continue
		}
		b64Content.WriteString(line)
	}

	return fmt.Sprintf("ssh-rsa %s deploypilot-generated", b64Content.String()), nil
}

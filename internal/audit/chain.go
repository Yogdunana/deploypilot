package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// ChainVerificationResult holds the result of verifying a single chain link.
type ChainVerificationResult struct {
	RecordID     string `json:"record_id"`
	Valid        bool   `json:"valid"`
	ExpectedHash string `json:"expected_hash"`
	ActualHash   string `json:"actual_hash"`
}

// AuditChain provides hash chain verification for audit log integrity.
// Each audit record's hash is computed from the previous hash combined with
// the record's own fields, creating a tamper-evident chain.
type AuditChain struct {
	db        *gorm.DB
	secretKey []byte
}

// NewAuditChain creates a new AuditChain instance.
func NewAuditChain(db *gorm.DB, secretKey []byte) *AuditChain {
	return &AuditChain{db: db, secretKey: secretKey}
}

// ComputeHash computes an HMAC-SHA256 hash for a chain link.
// The hash is derived from prevHash + record.ID + record.Action + record.Timestamp.
func (ac *AuditChain) ComputeHash(prevHash string, record *model.AuditLog) string {
	data := fmt.Sprintf("%s|%s|%s|%s",
		prevHash,
		record.ID,
		record.Action,
		record.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	)
	mac := hmac.New(sha256.New, ac.secretKey)
	_, _ = mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// AppendHash stores the hash for a new audit record in the chain.
// It looks up the previous record's hash to maintain chain continuity.
func (ac *AuditChain) AppendHash(recordID string, hash string) error {
	// Find the previous audit record's hash
	var prevHash string
	var prevEntry model.AuditHash

	// Look for the most recent hash entry before this recordID
	if err := ac.db.Where("audit_id < ?", recordID).
		Order("audit_id DESC").
		First(&prevEntry).Error; err == nil {
		prevHash = prevEntry.Hash
	}

	entry := &model.AuditHash{
		AuditID:      recordID,
		Hash:         hash,
		PreviousHash: prevHash,
	}

	return ac.db.Create(entry).Error
}

// GetRecordHash retrieves the stored hash for an audit record.
func (ac *AuditChain) GetRecordHash(recordID string) (string, error) {
	var entry model.AuditHash
	if err := ac.db.Where("audit_id = ?", recordID).First(&entry).Error; err != nil {
		return "", err
	}
	return entry.Hash, nil
}

// VerifyChain walks through all audit records and verifies the hash chain integrity.
// It returns a list of verification results for each record in the chain.
func (ac *AuditChain) VerifyChain() ([]ChainVerificationResult, error) {
	// Get all audit logs ordered by ID (insertion order)
	var logs []model.AuditLog
	if err := ac.db.Order("id ASC").Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}

	if len(logs) == 0 {
		return []ChainVerificationResult{}, nil
	}

	// Get all hash entries
	var hashes []model.AuditHash
	if err := ac.db.Order("audit_id ASC").Find(&hashes).Error; err != nil {
		return nil, fmt.Errorf("failed to query audit hashes: %w", err)
	}

	// Build a map of auditID -> AuditHash
	hashMap := make(map[string]model.AuditHash, len(hashes))
	for _, h := range hashes {
		hashMap[h.AuditID] = h
	}

	results := make([]ChainVerificationResult, 0, len(logs))
	prevHash := "" // genesis hash is empty

	for _, log := range logs {
		expectedHash := ac.ComputeHash(prevHash, &log)

		storedHash, hasHash := hashMap[log.ID]
		actualHash := ""
		if hasHash {
			actualHash = storedHash.Hash
		}

		valid := hasHash && (actualHash == expectedHash)

		results = append(results, ChainVerificationResult{
			RecordID:     log.ID,
			Valid:        valid,
			ExpectedHash: expectedHash,
			ActualHash:   actualHash,
		})

		// Advance the chain: use the expected hash as the next prevHash
		// This ensures that even if the stored hash was tampered with,
		// we detect it because subsequent hashes won't match
		if hasHash {
			prevHash = storedHash.Hash
		} else {
			prevHash = expectedHash
		}
	}

	return results, nil
}

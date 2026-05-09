package backup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// Config holds database backup configuration.
type Config struct {
	Enabled        bool          `mapstructure:"enabled"`
	Interval       time.Duration `mapstructure:"interval"`        // backup interval (default: 6h)
	RetentionCount int           `mapstructure:"retention_count"` // max backup files to keep (default: 10)
	RetentionDays  int           `mapstructure:"retention_days"`  // max days to keep backups (default: 30)
	BackupDir      string        `mapstructure:"backup_dir"`      // directory to store backups (default: ./data/backups)
}

// DefaultConfig returns sensible defaults for backup configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:        true,
		Interval:       6 * time.Hour,
		RetentionCount: 10,
		RetentionDays:  30,
		BackupDir:      "./data/backups",
	}
}

// Record represents a database backup record.
type Record struct {
	ID            string    `gorm:"primaryKey" json:"id"`
	Type          string    `json:"type"`                // "database" or "container"
	AppID         string    `gorm:"index" json:"app_id,omitempty"` // empty for system DB backups
	Status        string    `json:"status"`              // "completed", "failed"
	FilePath      string    `json:"file_path"`
	FileSize      int64     `json:"file_size"`
	Trigger       string    `json:"trigger"`             // "manual", "scheduled", "pre_deploy"
	Error         string    `json:"error,omitempty"`
	StorageType   string    `gorm:"column:storage_type" json:"storage_type,omitempty"`   // "local", "s3", "oss", "cos", "minio"
	StoragePath   string    `gorm:"column:storage_path" json:"storage_path,omitempty"`   // cloud storage key
	StorageBucket string    `gorm:"column:storage_bucket" json:"storage_bucket,omitempty"` // cloud bucket name
	FileChecksum  string    `gorm:"column:file_checksum" json:"file_checksum,omitempty"` // SHA-256 checksum
	Encrypted     bool      `gorm:"column:encrypted" json:"encrypted,omitempty"`          // whether backup is encrypted
	CreatedAt     time.Time `json:"created_at"`
}

func (Record) TableName() string { return "backup_records" }

// Service manages database backups with scheduling and retention.
type Service struct {
	mu            sync.Mutex
	config        Config
	db            *gorm.DB
	dsn           string
	dbType        string
	cancel        context.CancelFunc
	running       bool
	storage       StorageProvider // optional: cloud storage backend
	encryptKey    []byte          // optional: 32-byte AES-256 key for backup encryption
}

// New creates a new backup service.
func New(cfg Config, db *gorm.DB, dbType, dsn string) *Service {
	if cfg.BackupDir == "" {
		cfg.BackupDir = "./data/backups"
	}
	return &Service{
		config: cfg,
		db:     db,
		dsn:    dsn,
		dbType: dbType,
	}
}

// Start begins the automatic backup scheduler.
func (s *Service) Start() {
	if !s.config.Enabled {
		slog.Info("database auto-backup is disabled")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true

	// Ensure backup directory exists
	if err := os.MkdirAll(s.config.BackupDir, 0750); err != nil {
		slog.Error("failed to create backup directory", "path", s.config.BackupDir, "error", err)
		return
	}

	// Run initial backup after a short delay (30s)
	go func() {
		select {
		case <-time.After(30 * time.Second):
			slog.Info("running initial database backup")
			if _, err := s.CreateBackup(ctx, "scheduled"); err != nil {
				slog.Error("initial backup failed", "error", err)
			}
		case <-ctx.Done():
			return
		}

		// Schedule periodic backups
		ticker := time.NewTicker(s.config.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				slog.Info("running scheduled database backup")
				if _, err := s.CreateBackup(ctx, "scheduled"); err != nil {
					slog.Error("scheduled backup failed", "error", err)
				}
			case <-ctx.Done():
				slog.Info("backup scheduler stopped")
				return
			}
		}
	}()

	slog.Info("database auto-backup started", "interval", s.config.Interval,
		"retention_count", s.config.RetentionCount, "retention_days", s.config.RetentionDays,
		"backup_dir", s.config.BackupDir)
}

// Stop gracefully stops the backup scheduler.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.running = false
}

// IsRunning returns whether the backup scheduler is active.
func (s *Service) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// GetConfig returns the current backup configuration.
func (s *Service) GetConfig() Config {
	return s.config
}

// SetStorage sets the cloud storage provider for uploading backups.
func (s *Service) SetStorage(provider StorageProvider) {
	s.storage = provider
}

// SetEncryptionKey sets the AES-256 encryption key for backup encryption.
// The key must be exactly 32 bytes.
func (s *Service) SetEncryptionKey(key []byte) {
	if len(key) == 32 {
		s.encryptKey = key
	}
}

// ListRecords returns backup records from the database.
func (s *Service) ListRecords(limit int) ([]Record, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var records []Record
	err := s.db.Order("created_at DESC").Limit(limit).Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list backup records: %w", err)
	}
	return records, nil
}

// ListByApp returns backup records for a specific application.
func (s *Service) ListByApp(appID string, limit int) ([]Record, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var records []Record
	err := s.db.Where("app_id = ?", appID).Order("created_at DESC").Limit(limit).Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list backup records for app: %w", err)
	}
	return records, nil
}

// DeleteRecord deletes a backup record and its file.
func (s *Service) DeleteRecord(id string) error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	var record Record
	if err := s.db.Where("id = ?", id).First(&record).Error; err != nil {
		return fmt.Errorf("backup record not found: %w", err)
	}

	// Delete the backup file
	if record.FilePath != "" {
		if err := os.Remove(record.FilePath); err != nil && !os.IsNotExist(err) {
			slog.Warn("failed to delete backup file", "path", record.FilePath, "error", err)
		}
	}

	// Delete the database record
	if err := s.db.Delete(&record).Error; err != nil {
		return fmt.Errorf("failed to delete backup record: %w", err)
	}

	return nil
}

// GetStatus returns the current backup service status.
func (s *Service) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"enabled":         s.config.Enabled,
		"running":         s.IsRunning(),
		"interval":        s.config.Interval.String(),
		"retention_count": s.config.RetentionCount,
		"retention_days":  s.config.RetentionDays,
		"backup_dir":      s.config.BackupDir,
		"db_type":         s.dbType,
	}

	// Count existing backup files
	if entries, err := os.ReadDir(s.config.BackupDir); err == nil {
		count := 0
		var totalSize int64
		for _, e := range entries {
			if !e.IsDir() && strings.HasPrefix(e.Name(), "db-") {
				count++
				if info, err := e.Info(); err == nil {
					totalSize += info.Size()
				}
			}
		}
		status["backup_count"] = count
		status["total_size"] = totalSize
	}

	return status
}

// saveRecord persists a backup record to the database.
func (s *Service) saveRecord(record *Record) {
	if s.db == nil {
		return
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	if err := s.db.Create(record).Error; err != nil {
		slog.Error("failed to save backup record", "error", err)
	}
}

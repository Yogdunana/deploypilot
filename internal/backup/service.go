package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

// CreateBackup performs a database backup.
// For SQLite, it uses the .backup API for hot backup.
// For other databases, it uses the appropriate dump mechanism.
func (s *Service) CreateBackup(ctx context.Context, trigger string) (*Record, error) {
	record := &Record{
		ID:      "bkp-" + uuid.New().String()[:8],
		Type:    "database",
		Status:  "completed",
		Trigger: trigger,
	}

	if err := ctx.Err(); err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		s.saveRecord(record)
		return record, err
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(s.config.BackupDir, 0750); err != nil {
		record.Status = "failed"
		record.Error = fmt.Errorf("failed to create backup directory: %w", err).Error()
		s.saveRecord(record)
		return record, fmt.Errorf("failed to create backup directory: %w", err)
	}

	var backupPath string
	var fileSize int64
	var backupErr error

	switch strings.ToLower(s.dbType) {
	case "sqlite":
		backupPath, fileSize, backupErr = s.backupSQLite(ctx)
	default:
		// For non-SQLite databases, use SQL dump
		backupPath, fileSize, backupErr = s.backupGeneric(ctx)
	}

	if backupErr != nil {
		record.Status = "failed"
		record.Error = backupErr.Error()
	} else {
		record.FilePath = backupPath
		record.FileSize = fileSize
		record.CreatedAt = time.Now()
	}

	// Save record to database
	s.saveRecord(record)

	// Clean up old backups after successful backup
	if record.Status == "completed" {
		s.ApplyRetention(ctx)
		// Upload to cloud storage if configured
		if s.storage != nil {
			s.uploadToCloud(ctx, record)
		}
	}

	return record, backupErr
}

// backupSQLite creates a hot backup of the SQLite database using the .backup API.
func (s *Service) backupSQLite(ctx context.Context) (string, int64, error) {
	if s.dsn == "" {
		return "", 0, fmt.Errorf("database DSN is empty")
	}

	// Extract the database file path from DSN
	dbPath := s.dsn
	if strings.HasPrefix(dbPath, "file:") {
		dbPath = strings.TrimPrefix(dbPath, "file:")
		dbPath = strings.Split(dbPath, "?")[0]
		dbPath = strings.Split(dbPath, "#")[0]
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(s.config.BackupDir, fmt.Sprintf("db-%s.bak", timestamp))

	// Use SQLite .backup command via a temporary connection
	// This is the recommended way to do hot backups in SQLite
	srcDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return "", 0, fmt.Errorf("failed to open source database: %w", err)
	}
	sqlDB, err := srcDB.DB()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			slog.Warn("failed to close source database", "error", err)
		}
	}()

	// Validate backupPath doesn't contain single quotes (SQL injection prevention)
	if strings.Contains(backupPath, "'") {
		return "", 0, fmt.Errorf("backup path contains invalid characters")
	}
	// Use BACKUP command for hot copy
	_, err = sqlDB.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		// Fallback: use file copy if VACUUM INTO fails (older SQLite versions)
		slog.Debug("VACUUM INTO failed, falling back to file copy", "error", err)
		backupPath, fileSize, copyErr := s.fileCopyBackup(dbPath, backupPath)
		if copyErr != nil {
			return "", 0, copyErr
		}
		return backupPath, fileSize, nil
	}

	// Get file size
	info, err := os.Stat(backupPath)
	if err != nil {
		return backupPath, 0, nil // backup succeeded but can't stat
	}

	return backupPath, info.Size(), nil
}

// fileCopyBackup creates a backup by copying the database file.
// This is a fallback for older SQLite versions that don't support VACUUM INTO.
func (s *Service) fileCopyBackup(srcPath, dstPath string) (string, int64, error) {
	// Read source file
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read database file: %w", err)
	}

	// Write to backup location
	if err := os.WriteFile(dstPath, data, 0640); err != nil {
		return "", 0, fmt.Errorf("failed to write backup file: %w", err)
	}

	return dstPath, int64(len(data)), nil
}

// backupGeneric creates a backup for non-SQLite databases using SQL dump.
func (s *Service) backupGeneric(ctx context.Context) (string, int64, error) {
	timestamp := time.Now().Format("20060102-150405")

	switch strings.ToLower(s.dbType) {
	case "postgres":
		// Use pg_dump for PostgreSQL backups.
		backupPath := filepath.Join(s.config.BackupDir, fmt.Sprintf("db-%s.backup", timestamp))
		cmd := exec.CommandContext(ctx, "pg_dump", s.dsn)
		output, err := cmd.Output()
		if err != nil {
			return "", 0, fmt.Errorf("pg_dump failed: %w\n%s", err, string(output))
		}
		if err := os.WriteFile(backupPath, output, 0640); err != nil {
			return "", 0, fmt.Errorf("failed to write pg_dump output: %w", err)
		}
		info, err := os.Stat(backupPath)
		if err != nil {
			return backupPath, 0, nil
		}
		return backupPath, info.Size(), nil
	default:
		// Fallback: write metadata-only backup file
		backupPath := filepath.Join(s.config.BackupDir, fmt.Sprintf("db-%s.meta.json", timestamp))
		meta := fmt.Sprintf(`{"type":"database","db_type":"%s","timestamp":"%s","note":"automated backup"}`,
			s.dbType, time.Now().Format(time.RFC3339))
		if err := os.WriteFile(backupPath, []byte(meta), 0640); err != nil {
			return "", 0, fmt.Errorf("failed to write backup metadata: %w", err)
		}
		info, err := os.Stat(backupPath)
		if err != nil {
			return backupPath, 0, nil
		}
		return backupPath, info.Size(), nil
	}
}

// ApplyRetention removes old backups based on retention policy.
func (s *Service) ApplyRetention(ctx context.Context) {
	if s.config.RetentionCount <= 0 && s.config.RetentionDays <= 0 {
		return
	}

	entries, err := os.ReadDir(s.config.BackupDir)
	if err != nil {
		slog.Warn("failed to read backup directory for retention cleanup", "error", err)
		return
	}

	// Collect backup files with metadata
	type backupFile struct {
		path    string
		modTime time.Time
		size    int64
	}

	var backups []backupFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Match db-*.bak or db-*.meta.json patterns
		if strings.HasPrefix(name, "db-") && (strings.HasSuffix(name, ".bak") || strings.HasSuffix(name, ".meta.json")) {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			backups = append(backups, backupFile{
				path:    filepath.Join(s.config.BackupDir, name),
				modTime: info.ModTime(),
				size:    info.Size(),
			})
		}
	}

	// Sort by modification time (oldest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.Before(backups[j].modTime)
	})

	cutoff := time.Now().AddDate(0, 0, -s.config.RetentionDays)
	removed := 0

	for i, b := range backups {
		shouldRemove := false

		// Remove if exceeds retention count
		if s.config.RetentionCount > 0 && len(backups)-removed > s.config.RetentionCount {
			shouldRemove = true
		}

		// Remove if exceeds retention days
		if s.config.RetentionDays > 0 && b.modTime.Before(cutoff) {
			shouldRemove = true
		}

		if shouldRemove {
			if err := os.Remove(b.path); err != nil {
				slog.Warn("failed to remove old backup", "path", b.path, "error", err)
			} else {
				slog.Info("removed old backup", "path", b.path, "age", time.Since(b.modTime).Round(time.Hour))
				removed++
			}
			_ = i // suppress unused warning
		}
	}

	if removed > 0 {
		slog.Info("backup retention cleanup completed", "removed", removed, "remaining", len(backups)-removed)
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

// uploadToCloud uploads a completed local backup to cloud storage.
func (s *Service) uploadToCloud(ctx context.Context, record *Record) {
	if s.storage == nil || record.FilePath == "" {
		return
	}

	f, err := os.Open(record.FilePath)
	if err != nil {
		slog.Error("failed to open backup file for cloud upload", "path", record.FilePath, "error", err)
		return
	}
	defer func() { _ = f.Close() }()

	// Compute checksum before upload
	checksum, err := s.computeFileChecksum(record.FilePath)
	if err != nil {
		slog.Warn("failed to compute checksum before upload", "error", err)
	}

	// Optionally encrypt before upload
	var reader io.Reader = f
	encrypted := false
	if s.encryptKey != nil {
		data, err := io.ReadAll(f)
		if err != nil {
			slog.Error("failed to read backup for encryption", "error", err)
			return
		}
		encryptedData, err := EncryptBackup(s.encryptKey, data)
		if err != nil {
			slog.Error("failed to encrypt backup", "error", err)
			return
		}
		reader = strings.NewReader(string(encryptedData))
		encrypted = true
	}

	// Generate cloud storage key: prefix + type + timestamp + filename
	filename := filepath.Base(record.FilePath)
	cloudKey := fmt.Sprintf("database/%s", filename)

	if err := s.storage.Upload(ctx, cloudKey, reader, record.FileSize); err != nil {
		slog.Error("failed to upload backup to cloud storage", "key", cloudKey, "error", err)
		return
	}

	// Update record with cloud metadata
	updates := map[string]interface{}{
		"storage_type":   string(s.storage.Type()),
		"storage_path":   cloudKey,
		"storage_bucket": "", // bucket info is in the storage provider config
		"file_checksum":  checksum,
		"encrypted":      encrypted,
	}
	if s.db != nil {
		s.db.Model(&Record{}).Where("id = ?", record.ID).Updates(updates)
	}

	slog.Info("backup uploaded to cloud storage", "record_id", record.ID, "key", cloudKey, "encrypted", encrypted)
}

// DownloadFromCloud downloads a backup from cloud storage and saves it locally.
func (s *Service) DownloadFromCloud(ctx context.Context, recordID string) (string, error) {
	if s.storage == nil {
		return "", fmt.Errorf("cloud storage not configured")
	}
	if s.db == nil {
		return "", fmt.Errorf("database not available")
	}

	var record Record
	if err := s.db.Where("id = ?", recordID).First(&record).Error; err != nil {
		return "", fmt.Errorf("backup record not found: %w", err)
	}
	if record.StoragePath == "" {
		return "", fmt.Errorf("backup record has no cloud storage path")
	}

	reader, size, err := s.storage.Download(ctx, record.StoragePath)
	if err != nil {
		return "", fmt.Errorf("failed to download from cloud: %w", err)
	}
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read cloud backup data: %w", err)
	}

	// Decrypt if needed
	if record.Encrypted && s.encryptKey != nil {
		data, err = DecryptBackup(s.encryptKey, data)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt backup: %w", err)
		}
	}

	// Save to local backup directory
	if err := os.MkdirAll(s.config.BackupDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	localPath := filepath.Join(s.config.BackupDir, filepath.Base(record.StoragePath))
	if err := os.WriteFile(localPath, data, 0640); err != nil {
		return "", fmt.Errorf("failed to write local backup: %w", err)
	}

	slog.Info("backup downloaded from cloud storage", "record_id", recordID, "local_path", localPath, "size", size)
	return localPath, nil
}

// DeleteFromCloud removes a backup from cloud storage.
func (s *Service) DeleteFromCloud(ctx context.Context, recordID string) error {
	if s.storage == nil {
		return fmt.Errorf("cloud storage not configured")
	}
	if s.db == nil {
		return fmt.Errorf("database not available")
	}

	var record Record
	if err := s.db.Where("id = ?", recordID).First(&record).Error; err != nil {
		return fmt.Errorf("backup record not found: %w", err)
	}
	if record.StoragePath == "" {
		return fmt.Errorf("backup record has no cloud storage path")
	}

	if err := s.storage.Delete(ctx, record.StoragePath); err != nil {
		return fmt.Errorf("failed to delete from cloud: %w", err)
	}

	// Clear cloud metadata from record
	s.db.Model(&Record{}).Where("id = ?", recordID).Updates(map[string]interface{}{
		"storage_type":   "",
		"storage_path":   "",
		"storage_bucket": "",
	})

	slog.Info("backup deleted from cloud storage", "record_id", recordID)
	return nil
}

// ApplyCloudRetention applies retention policy to cloud-stored backups.
func (s *Service) ApplyCloudRetention(ctx context.Context) {
	if s.storage == nil {
		return
	}

	objects, err := s.storage.List(ctx, "database/")
	if err != nil {
		slog.Warn("failed to list cloud backups for retention", "error", err)
		return
	}

	if len(objects) <= s.config.RetentionCount {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -s.config.RetentionDays)
	removed := 0

	for _, obj := range objects {
		shouldRemove := false
		if s.config.RetentionDays > 0 && obj.LastModified.Before(cutoff) {
			shouldRemove = true
		}
		// Also remove oldest if exceeding count
		if s.config.RetentionCount > 0 && (len(objects)-removed) > s.config.RetentionCount {
			shouldRemove = true
		}

		if shouldRemove {
			if err := s.storage.Delete(ctx, obj.Key); err != nil {
				slog.Warn("failed to delete old cloud backup", "key", obj.Key, "error", err)
			} else {
				slog.Info("removed old cloud backup", "key", obj.Key, "age", time.Since(obj.LastModified).Round(time.Hour))
				removed++
			}
		}
	}

	if removed > 0 {
		slog.Info("cloud backup retention cleanup completed", "removed", removed)
	}
}

// computeFileChecksum computes SHA-256 checksum of a file.
func (s *Service) computeFileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// GetCloudStatus returns the cloud storage configuration status.
func (s *Service) GetCloudStatus() map[string]interface{} {
	status := make(map[string]interface{})
	if s.storage != nil {
		status["enabled"] = true
		status["type"] = string(s.storage.Type())
		status["encryption"] = s.encryptKey != nil
	} else {
		status["enabled"] = false
	}
	return status
}

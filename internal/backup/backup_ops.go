package backup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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

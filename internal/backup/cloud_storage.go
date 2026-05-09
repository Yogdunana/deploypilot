package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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


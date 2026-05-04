package backup

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

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

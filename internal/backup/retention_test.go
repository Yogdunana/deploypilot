package backup

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestApplyRetention_NoRetentionPolicy(t *testing.T) {
	tempDir := t.TempDir()

	for i := 0; i < 5; i++ {
		filePath := filepath.Join(tempDir, "db-2024010"+strconv.Itoa(i)+"-000000.bak")
		err := os.WriteFile(filePath, []byte("test data"), 0640)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	s := &Service{
		config: Config{
			BackupDir:       tempDir,
			RetentionCount: 0,
			RetentionDays:  0,
		},
	}

	s.ApplyRetention(nil)

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read backup dir: %v", err)
	}

	if len(entries) != 5 {
		t.Errorf("expected 5 files after no retention, got %d", len(entries))
	}
}

func TestApplyRetention_RetentionCount(t *testing.T) {
	tempDir := t.TempDir()

	for i := 0; i < 10; i++ {
		filePath := filepath.Join(tempDir, "db-2024010"+strconv.Itoa(i)+"-000000.bak")
		err := os.WriteFile(filePath, []byte("test data"), 0640)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	s := &Service{
		config: Config{
			BackupDir:       tempDir,
			RetentionCount: 5,
			RetentionDays:  0,
		},
	}

	s.ApplyRetention(nil)

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read backup dir: %v", err)
	}

	if len(entries) != 5 {
		t.Errorf("expected 5 files after retention count=5, got %d", len(entries))
	}
}

func TestApplyRetention_RetentionDays(t *testing.T) {
	tempDir := t.TempDir()

	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	newTime := time.Now().Add(-1 * 24 * time.Hour)

	for i := 0; i < 5; i++ {
		filePath := filepath.Join(tempDir, "db-2024010"+strconv.Itoa(i)+"-000000.bak")
		err := os.WriteFile(filePath, []byte("test data"), 0640)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		var modTime time.Time
		if i < 3 {
			modTime = oldTime
		} else {
			modTime = newTime
		}
		err = os.Chtimes(filePath, modTime, modTime)
		if err != nil {
			t.Fatalf("failed to set modification time: %v", err)
		}
	}

	s := &Service{
		config: Config{
			BackupDir:       tempDir,
			RetentionCount: 0,
			RetentionDays:  14,
		},
	}

	s.ApplyRetention(nil)

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read backup dir: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 files after retention days=14, got %d", len(entries))
	}
}

func TestApplyRetention_BothPolicies(t *testing.T) {
	tempDir := t.TempDir()

	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	newTime := time.Now().Add(-1 * 24 * time.Hour)

	for i := 0; i < 10; i++ {
		filePath := filepath.Join(tempDir, "db-2024010"+strconv.Itoa(i)+"-000000.bak")
		err := os.WriteFile(filePath, []byte("test data"), 0640)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		var modTime time.Time
		if i < 7 {
			modTime = oldTime
		} else {
			modTime = newTime
		}
		err = os.Chtimes(filePath, modTime, modTime)
		if err != nil {
			t.Fatalf("failed to set modification time: %v", err)
		}
	}

	s := &Service{
		config: Config{
			BackupDir:       tempDir,
			RetentionCount: 5,
			RetentionDays:  14,
		},
	}

	s.ApplyRetention(nil)

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read backup dir: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 files after both policies (3 new enough, limited by count=5), got %d", len(entries))
	}
}

func TestApplyRetention_InvalidBackupDir(t *testing.T) {
	s := &Service{
		config: Config{
			BackupDir:       "/nonexistent/path",
			RetentionCount: 5,
			RetentionDays:  14,
		},
	}

	s.ApplyRetention(nil)
}

func TestApplyRetention_OnlyBackupFiles(t *testing.T) {
	tempDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tempDir, "db-20240101-000000.bak"), []byte("backup1"), 0640)
	if err != nil {
		t.Fatalf("failed to create backup file: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "db-20240102-000000.meta.json"), []byte("backup2"), 0640)
	if err != nil {
		t.Fatalf("failed to create backup file: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "log.txt"), []byte("not a backup"), 0640)
	if err != nil {
		t.Fatalf("failed to create non-backup file: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "other.db"), []byte("not a backup"), 0640)
	if err != nil {
		t.Fatalf("failed to create non-backup file: %v", err)
	}

	s := &Service{
		config: Config{
			BackupDir:       tempDir,
			RetentionCount: 1,
			RetentionDays:  0,
		},
	}

	s.ApplyRetention(nil)

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read backup dir: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 files (1 backup + 2 non-backup), got %d", len(entries))
	}
}

func TestApplyRetention_EmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	s := &Service{
		config: Config{
			BackupDir:       tempDir,
			RetentionCount: 5,
			RetentionDays:  14,
		},
	}

	s.ApplyRetention(nil)

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read backup dir: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 files in empty dir, got %d", len(entries))
	}
}
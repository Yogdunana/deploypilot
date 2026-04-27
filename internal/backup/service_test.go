package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestService(t *testing.T) (*Service, func()) {
	t.Helper()

	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Create backup_records table
	db.Exec(`CREATE TABLE IF NOT EXISTS backup_records (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL DEFAULT 'database',
		app_id TEXT,
		status TEXT NOT NULL DEFAULT 'completed',
		file_path TEXT,
		file_size INTEGER DEFAULT 0,
		trigger TEXT DEFAULT 'manual',
		error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	cfg := Config{
		Enabled:        true,
		Interval:       time.Hour,
		RetentionCount: 5,
		RetentionDays:  7,
		BackupDir:      dir,
	}

	svc := New(cfg, db, "sqlite", "")
	sqlDB, _ := db.DB()
	cleanup := func() {
		svc.Stop()
		sqlDB.Close()
	}

	return svc, cleanup
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Enabled {
		t.Error("expected enabled to be true")
	}
	if cfg.Interval != 6*time.Hour {
		t.Errorf("expected interval 6h, got %v", cfg.Interval)
	}
	if cfg.RetentionCount != 10 {
		t.Errorf("expected retention_count 10, got %d", cfg.RetentionCount)
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("expected retention_days 30, got %d", cfg.RetentionDays)
	}
}

func TestNewService(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.config.BackupDir == "" {
		t.Error("expected non-empty backup dir")
	}
}

func TestStartStop(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	if svc.IsRunning() {
		t.Error("expected service to not be running initially")
	}

	svc.Start()
	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	if !svc.IsRunning() {
		t.Error("expected service to be running after Start()")
	}

	svc.Stop()
	time.Sleep(50 * time.Millisecond)

	if svc.IsRunning() {
		t.Error("expected service to not be running after Stop()")
	}
}

func TestStartDisabled(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	svc.config.Enabled = false
	svc.Start()
	time.Sleep(50 * time.Millisecond)

	if svc.IsRunning() {
		t.Error("expected disabled service to not start")
	}
}

func TestCreateBackup_WithRealDB(t *testing.T) {
	dir := t.TempDir()

	// Create a real SQLite database file
	dbPath := filepath.Join(dir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// Create backup_records table in the real DB
	db.Exec(`CREATE TABLE IF NOT EXISTS backup_records (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL DEFAULT 'database',
		app_id TEXT,
		status TEXT NOT NULL DEFAULT 'completed',
		file_path TEXT,
		file_size INTEGER DEFAULT 0,
		trigger TEXT DEFAULT 'manual',
		error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	backupDir := filepath.Join(dir, "backups")
	cfg := Config{
		Enabled:        true,
		Interval:       time.Hour,
		RetentionCount: 5,
		RetentionDays:  7,
		BackupDir:      backupDir,
	}

	svc := New(cfg, db, "sqlite", dbPath)
	defer svc.Stop()

	record, err := svc.CreateBackup(context.Background(), "manual")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if record.Status != "completed" {
		t.Errorf("expected status completed, got %s", record.Status)
	}
	if record.Trigger != "manual" {
		t.Errorf("expected trigger manual, got %s", record.Trigger)
	}
	if record.FilePath == "" {
		t.Error("expected non-empty file path")
	}
	if record.FileSize <= 0 {
		t.Errorf("expected positive file size, got %d", record.FileSize)
	}

	// Verify backup file exists
	if _, err := os.Stat(record.FilePath); os.IsNotExist(err) {
		t.Errorf("backup file does not exist: %s", record.FilePath)
	}
}

func TestCreateBackup_CancelledContext(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	record, err := svc.CreateBackup(ctx, "manual")
	if err == nil {
		t.Error("expected error from cancelled context")
	}
	if record.Status != "failed" {
		t.Errorf("expected status failed, got %s", record.Status)
	}
}

func TestApplyRetention(t *testing.T) {
	dir := t.TempDir()

	// Create 8 backup files
	for i := 0; i < 8; i++ {
		path := filepath.Join(dir, "db-test.bak")
		// Use different timestamps by modifying file mod time
		content := []byte("test backup data")
		if err := os.WriteFile(path, content, 0640); err != nil {
			t.Fatalf("failed to create backup file: %v", err)
		}
		// Set mod time to past
		pastTime := time.Now().Add(-time.Duration(i+1) * time.Hour)
		os.Chtimes(path, pastTime, pastTime)
	}

	cfg := Config{
		Enabled:        true,
		Interval:       time.Hour,
		RetentionCount: 3,
		RetentionDays:  1,
		BackupDir:      dir,
	}

	svc := New(cfg, nil, "sqlite", "")
	svc.ApplyRetention(context.Background())

	// Should have removed 5 files (8 - 3 retention count)
	entries, _ := os.ReadDir(dir)
	count := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".bak" {
			count++
		}
	}
	if count > 3 {
		t.Errorf("expected at most 3 backup files, got %d", count)
	}
}

func TestListRecords(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	// Insert some records
	svc.saveRecord(&Record{ID: "bkp-1", Type: "database", Status: "completed", Trigger: "manual", CreatedAt: time.Now()})
	svc.saveRecord(&Record{ID: "bkp-2", Type: "database", Status: "completed", Trigger: "scheduled", CreatedAt: time.Now().Add(-time.Hour)})

	records, err := svc.ListRecords(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
	// Should be ordered by created_at DESC
	if records[0].ID != "bkp-1" {
		t.Errorf("expected first record bkp-1, got %s", records[0].ID)
	}
}

func TestDeleteRecord(t *testing.T) {
	dir := t.TempDir()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	db.Exec(`CREATE TABLE IF NOT EXISTS backup_records (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL DEFAULT 'database',
		app_id TEXT,
		status TEXT NOT NULL DEFAULT 'completed',
		file_path TEXT,
		file_size INTEGER DEFAULT 0,
		trigger TEXT DEFAULT 'manual',
		error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// Create a backup file
	backupPath := filepath.Join(dir, "test-backup.bak")
	os.WriteFile(backupPath, []byte("test"), 0640)

	cfg := Config{BackupDir: dir}
	svc := New(cfg, db, "sqlite", "")

	// Insert record
	svc.saveRecord(&Record{ID: "bkp-del", Type: "database", Status: "completed", FilePath: backupPath, CreatedAt: time.Now()})

	// Delete it
	err := svc.DeleteRecord("bkp-del")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("expected backup file to be deleted")
	}

	// Verify record is deleted
	var count int64
	db.Table("backup_records").Where("id = ?", "bkp-del").Count(&count)
	if count != 0 {
		t.Error("expected record to be deleted from database")
	}
}

func TestGetStatus(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	status := svc.GetStatus()

	if status["enabled"] != true {
		t.Error("expected enabled to be true")
	}
	if status["running"] != false {
		t.Error("expected running to be false initially")
	}
	if status["backup_dir"] == "" {
		t.Error("expected non-empty backup_dir")
	}
}

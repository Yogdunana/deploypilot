package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/mcp"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Create tables
	db.Exec(`CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY,
		tenant_id TEXT,
		server_id TEXT,
		app_name TEXT,
		app_id TEXT,
		container_name TEXT,
		image TEXT,
		previous_image TEXT,
		deploy_type TEXT DEFAULT 'deploy',
		config_snapshot TEXT,
		status TEXT DEFAULT 'deploying',
		preflight_code TEXT,
		preflight_message TEXT,
		preflight_checks TEXT,
		error_message TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS apps (
		id TEXT PRIMARY KEY,
		tenant_id TEXT,
		server_id TEXT,
		name TEXT NOT NULL,
		repo_url TEXT NOT NULL,
		branch TEXT DEFAULT 'main',
		domain TEXT,
		tech_stack TEXT DEFAULT 'docker',
		deploy_mode TEXT DEFAULT 'api',
		status TEXT DEFAULT 'pending',
		current_version TEXT,
		container_name TEXT,
		env_vars TEXT,
		resource_limits TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	return db
}

func TestFindPreviousSuccessfulImage(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Exec("DROP TABLE deployments") }()

	b := &Bridge{DB: db}

	// Insert deployment records
	now := time.Now()
	records := []model.DeploymentRecord{
		{ID: "dep-1", ContainerName: "myapp", Image: "nginx:1.24", Status: "success", CreatedAt: now.Add(-30 * time.Minute), UpdatedAt: now.Add(-30 * time.Minute)},
		{ID: "dep-2", ContainerName: "myapp", Image: "nginx:1.25", Status: "success", CreatedAt: now.Add(-20 * time.Minute), UpdatedAt: now.Add(-20 * time.Minute)},
		{ID: "dep-3", ContainerName: "myapp", Image: "nginx:1.26", Status: "failed", CreatedAt: now.Add(-10 * time.Minute), UpdatedAt: now.Add(-10 * time.Minute)},
	}
	for _, r := range records {
		db.Create(&r)
	}

	// Should return the most recent SUCCESSFUL image (1.25), not the failed one (1.26)
	img, err := b.findPreviousSuccessfulImage(context.Background(), "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img != "nginx:1.25" {
		t.Errorf("expected nginx:1.25, got %s", img)
	}
}

func TestFindPreviousSuccessfulImage_NoRecords(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Exec("DROP TABLE deployments") }()

	b := &Bridge{DB: db}

	_, err := b.findPreviousSuccessfulImage(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent container")
	}
}

func TestBuildRollbackConfig_FromAppRecord(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_ = db.Exec("DROP TABLE deployments")
		_ = db.Exec("DROP TABLE apps")
	}()

	// Insert app with env vars and resource limits
	envVars := `{"DB_HOST":"localhost","DB_PORT":"5432"}`
	resourceLimits := `{"cpu":"0.5","memory":"512m"}`
	db.Exec(`INSERT INTO apps (id, name, repo_url, container_name, env_vars, resource_limits, domain) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"app-1", "myapp", "https://github.com/test/app", "myapp", envVars, resourceLimits, "example.com")

	b := &Bridge{DB: db}

	cfg, err := b.buildRollbackConfig(context.Background(), "myapp", "nginx:1.24")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Image != "nginx:1.24" {
		t.Errorf("expected image nginx:1.24, got %s", cfg.Image)
	}
	if cfg.AppName != "myapp" {
		t.Errorf("expected app name myapp, got %s", cfg.AppName)
	}
	if cfg.EnvVars == nil || cfg.EnvVars["DB_HOST"] != "localhost" {
		t.Errorf("expected env vars to be restored, got %v", cfg.EnvVars)
	}
	if cfg.CPU != "0.5" || cfg.Memory != "512m" {
		t.Errorf("expected resource limits restored, cpu=%s memory=%s", cfg.CPU, cfg.Memory)
	}
	// Should have default ports when domain is set
	if cfg.Ports != "80:80, 443:443" {
		t.Errorf("expected default ports for domain, got %s", cfg.Ports)
	}
}

func TestBuildRollbackConfig_FromConfigSnapshot(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_ = db.Exec("DROP TABLE deployments")
		_ = db.Exec("DROP TABLE apps")
	}()

	// Insert app
	db.Exec(`INSERT INTO apps (id, name, repo_url, container_name) VALUES (?, ?, ?, ?)`,
		"app-1", "myapp", "https://github.com/test/app", "myapp")

	// Insert a successful deployment with config snapshot
	snapshot := mcp.DeployConfig{
		Image:         "nginx:1.24",
		ContainerName: "myapp",
		Ports:         "8080:80",
		EnvVars:       map[string]string{"API_KEY": "secret"},
		Network:       "mynet",
		Volumes:       "/data:/app/data",
		CPU:           "1.0",
		Memory:        "1g",
		RestartPolicy: "always",
	}
	snapJSON, _ := json.Marshal(snapshot)
	now := time.Now()
	db.Exec(`INSERT INTO deployments (id, container_name, image, status, config_snapshot, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"dep-1", "myapp", "nginx:1.24", "success", string(snapJSON), now.Format(time.RFC3339), now.Format(time.RFC3339))

	b := &Bridge{DB: db}

	cfg, err := b.buildRollbackConfig(context.Background(), "myapp", "nginx:1.23")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Config snapshot should take precedence
	if cfg.Ports != "8080:80" {
		t.Errorf("expected ports from snapshot 8080:80, got %s", cfg.Ports)
	}
	if cfg.Network != "mynet" {
		t.Errorf("expected network from snapshot mynet, got %s", cfg.Network)
	}
	if cfg.Volumes != "/data:/app/data" {
		t.Errorf("expected volumes from snapshot, got %s", cfg.Volumes)
	}
	if cfg.RestartPolicy != "always" {
		t.Errorf("expected restart policy from snapshot always, got %s", cfg.RestartPolicy)
	}
}

func TestBuildRollbackConfig_NoAppRecord(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Exec("DROP TABLE apps") }()

	b := &Bridge{DB: db}

	cfg, err := b.buildRollbackConfig(context.Background(), "unknown", "nginx:1.24")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return minimal config
	if cfg.Image != "nginx:1.24" {
		t.Errorf("expected image nginx:1.24, got %s", cfg.Image)
	}
	if cfg.RestartPolicy != "unless-stopped" {
		t.Errorf("expected default restart policy, got %s", cfg.RestartPolicy)
	}
}

func TestSaveRollbackRecord(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Exec("DROP TABLE deployments") }()

	b := &Bridge{DB: db}

	cfg := mcp.DeployConfig{
		Image:         "nginx:1.24",
		ContainerName: "myapp",
		AppName:       "myapp",
		Ports:         "80:80",
		EnvVars:       map[string]string{"KEY": "val"},
	}

	b.saveRollbackRecord(context.Background(), cfg, "nginx:1.25", "rollback", "success", "")

	var record model.DeploymentRecord
	if err := db.Where("container_name = ?", "myapp").First(&record).Error; err != nil {
		t.Fatalf("failed to find rollback record: %v", err)
	}

	if record.Image != "nginx:1.24" {
		t.Errorf("expected image nginx:1.24, got %s", record.Image)
	}
	if record.PreviousImage != "nginx:1.25" {
		t.Errorf("expected previous_image nginx:1.25, got %s", record.PreviousImage)
	}
	if record.DeployType != "rollback" {
		t.Errorf("expected deploy_type rollback, got %s", record.DeployType)
	}
	if record.Status != "success" {
		t.Errorf("expected status success, got %s", record.Status)
	}
	if record.ConfigSnapshot == "" {
		t.Error("expected config_snapshot to be populated")
	}
}

func TestGetDeploymentHistory(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Exec("DROP TABLE deployments") }()

	b := &Bridge{DB: db}
	now := time.Now()

	// Insert records in chronological order
	for i := 0; i < 5; i++ {
		db.Exec(`INSERT INTO deployments (id, container_name, image, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			"dep-"+string(rune('1'+i)), "myapp", "nginx:1.2"+string(rune('0'+i)), "success",
			now.Add(time.Duration(i)*time.Minute).Format(time.RFC3339),
			now.Add(time.Duration(i)*time.Minute).Format(time.RFC3339))
	}

	records, err := b.GetDeploymentHistory(context.Background(), "myapp", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
	// Should be ordered by most recent first (1.24, 1.23, 1.22)
	if records[0].Image != "nginx:1.24" {
		t.Errorf("expected first record image nginx:1.24, got %s", records[0].Image)
	}
}

func TestGetAppDeploymentHistory(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_ = db.Exec("DROP TABLE deployments")
		_ = db.Exec("DROP TABLE apps")
	}()

	b := &Bridge{DB: db}
	now := time.Now()

	// Insert records for different apps
	db.Exec(`INSERT INTO deployments (id, container_name, app_id, image, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"dep-1", "app1", "app-id-1", "nginx:1.24", "success", now.Format(time.RFC3339), now.Format(time.RFC3339))
	db.Exec(`INSERT INTO deployments (id, container_name, app_id, image, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"dep-2", "app2", "app-id-2", "redis:7", "success", now.Format(time.RFC3339), now.Format(time.RFC3339))
	db.Exec(`INSERT INTO deployments (id, container_name, app_id, image, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"dep-3", "app1", "app-id-1", "nginx:1.23", "success", now.Add(-time.Hour).Format(time.RFC3339), now.Add(-time.Hour).Format(time.RFC3339))

	records, err := b.GetAppDeploymentHistory(context.Background(), "app-id-1", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records for app-id-1, got %d", len(records))
	}
}

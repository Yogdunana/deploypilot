package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/database"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/mcp"
)

func TestSaveDeploymentRecord_PreflightFailed(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	b := &Bridge{DB: db}
	cfg := mcp.DeployConfig{
		Image:         "nginx:alpine",
		ContainerName: "test-pf-save",
		ServerID:      "srv-1",
	}
	pfResult := &PreflightResult{
		Passed:  false,
		Code:    PreflightDockerUnavailable,
		Message: "Docker not found",
		Checks: []PreflightCheck{
			{Name: "Docker", Passed: false, Message: "not found", Suggestion: "install docker"},
		},
	}

	b.saveDeploymentRecord(context.Background(), cfg, "preflight_failed", pfResult)

	var record model.DeploymentRecord
	err = db.Where("container_name = ?", "test-pf-save").First(&record).Error
	if err != nil {
		t.Fatalf("record not found: %v", err)
	}
	if record.Status != "preflight_failed" {
		t.Errorf("expected status preflight_failed, got %s", record.Status)
	}
	if record.PreflightCode != "REMOTE_DOCKER_UNAVAILABLE" {
		t.Errorf("expected code REMOTE_DOCKER_UNAVAILABLE, got %s", record.PreflightCode)
	}
	if record.PreflightMessage != "Docker not found" {
		t.Errorf("expected message 'Docker not found', got %s", record.PreflightMessage)
	}
	// Verify checks JSON
	var checks []PreflightCheck
	if err := json.Unmarshal([]byte(record.PreflightChecks), &checks); err != nil {
		t.Fatalf("failed to unmarshal checks: %v", err)
	}
	if len(checks) != 1 || checks[0].Name != "Docker" {
		t.Errorf("unexpected checks: %+v", checks)
	}
}

func TestSaveDeploymentRecord_Success(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	b := &Bridge{DB: db}
	cfg := mcp.DeployConfig{
		Image:         "nginx:alpine",
		ContainerName: "test-save-ok",
	}

	b.saveDeploymentRecord(context.Background(), cfg, "success", nil)

	var record model.DeploymentRecord
	err = db.Where("container_name = ?", "test-save-ok").First(&record).Error
	if err != nil {
		t.Fatalf("record not found: %v", err)
	}
	if record.Status != "success" {
		t.Errorf("expected status success, got %s", record.Status)
	}
	if record.PreflightCode != "" {
		t.Errorf("expected empty preflight code for success, got %s", record.PreflightCode)
	}
}

func TestGetLatestDeploymentRecord(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	b := &Bridge{DB: db}

	// Save two records
	b.saveDeploymentRecord(context.Background(), mcp.DeployConfig{
		ContainerName: "multi-rec", Image: "nginx:v1",
	}, "success", nil)
	b.saveDeploymentRecord(context.Background(), mcp.DeployConfig{
		ContainerName: "multi-rec", Image: "nginx:v2",
	}, "preflight_failed", &PreflightResult{
		Passed: false, Code: PreflightTCPUnreachable, Message: "timeout",
	})

	record, err := b.GetLatestDeploymentRecord(context.Background(), "multi-rec")
	if err != nil {
		t.Fatalf("failed to get latest record: %v", err)
	}
	if record.Status != "preflight_failed" {
		t.Errorf("expected latest status preflight_failed, got %s", record.Status)
	}
	if record.Image != "nginx:v2" {
		t.Errorf("expected latest image nginx:v2, got %s", record.Image)
	}
}

func TestGetLatestDeploymentRecord_NotFound(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	b := &Bridge{DB: db}
	_, err = b.GetLatestDeploymentRecord(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent record")
	}
}

func TestSaveDeploymentRecord_DBError(t *testing.T) {
    // Use a nil DB to trigger error path
    b := &Bridge{DB: nil}
    cfg := mcp.DeployConfig{
        Image:         "nginx:alpine",
        ContainerName: "test-db-err",
    }
    // Should not panic, just log the error
    b.saveDeploymentRecord(context.Background(), cfg, "success", nil)
}

func TestLogPreflightResult(t *testing.T) {
    result := &PreflightResult{
        Passed:  false,
        Code:    PreflightTCPUnreachable,
        Message: "timeout",
        Checks: []PreflightCheck{
            {Name: "TCP", Passed: false, Message: "connection refused", Suggestion: "check firewall"},
        },
    }
    // Should not panic
    logPreflightResult("test-container", result)
}

func TestGenerateID(t *testing.T) {
    id1 := generateID()
    id2 := generateID()
    if id1 == id2 {
        t.Error("generateID should return unique IDs")
    }
    if id1 == "" {
        t.Error("generateID should not return empty string")
    }
}

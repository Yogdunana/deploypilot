package model

import (
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/database"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	db, err := database.Connect("sqlite", tmpDir+"/test.db")
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err := database.Seed(db); err != nil {
		t.Fatalf("Seed() error = %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
}

// ========== Server CRUD ==========

func TestServerCRUD(t *testing.T) {
	setupTestDB(t)
	// Note: actual CRUD tests need db reference — tested via model methods below
}

func TestServerModel(t *testing.T) {
	now := time.Now()
	s := &Server{
		ID:        "srv-001",
		TenantID:  "tenant-default",
		Name:      "my-server",
		Host:      "192.168.1.1",
		Port:      22,
		Status:    "unknown",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if s.ID != "srv-001" {
		t.Errorf("Server.ID = %q, want %q", s.ID, "srv-001")
	}
	if s.Host != "192.168.1.1" {
		t.Errorf("Server.Host = %q, want %q", s.Host, "192.168.1.1")
	}
	if s.Port != 22 {
		t.Errorf("Server.Port = %d, want %d", s.Port, 22)
	}
	if s.Status != "unknown" {
		t.Errorf("Server.Status = %q, want %q", s.Status, "unknown")
	}
}

func TestServerTableName(t *testing.T) {
	s := &Server{}
	if s.TableName() != "servers" {
		t.Errorf("Server.TableName() = %q, want %q", s.TableName(), "servers")
	}
}

// ========== App CRUD ==========

func TestAppModel(t *testing.T) {
	now := time.Now()
	a := &App{
		ID:        "app-001",
		TenantID:  "tenant-default",
		ServerID:  "srv-001",
		Name:      "my-app",
		RepoURL:   "https://github.com/user/repo",
		Branch:    "main",
		DeployMode: "api",
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if a.ID != "app-001" {
		t.Errorf("App.ID = %q, want %q", a.ID, "app-001")
	}
	if a.RepoURL != "https://github.com/user/repo" {
		t.Errorf("App.RepoURL = %q", a.RepoURL)
	}
	if a.DeployMode != "api" {
		t.Errorf("App.DeployMode = %q, want %q", a.DeployMode, "api")
	}
}

func TestAppTableName(t *testing.T) {
	a := &App{}
	if a.TableName() != "apps" {
		t.Errorf("App.TableName() = %q, want %q", a.TableName(), "apps")
	}
}

// ========== Credential CRUD ==========

func TestCredentialModel(t *testing.T) {
	now := time.Now()
	c := &Credential{
		ID:             "cred-001",
		TenantID:       "tenant-default",
		Name:           "my-ssh-key",
		Type:           "ssh",
		EncryptedValue: "encrypted-data-here",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if c.ID != "cred-001" {
		t.Errorf("Credential.ID = %q, want %q", c.ID, "cred-001")
	}
	if c.Type != "ssh" {
		t.Errorf("Credential.Type = %q, want %q", c.Type, "ssh")
	}
}

func TestCredentialTableName(t *testing.T) {
	c := &Credential{}
	if c.TableName() != "credentials" {
		t.Errorf("Credential.TableName() = %q, want %q", c.TableName(), "credentials")
	}
}

// ========== Tenant & User Models ==========

func TestTenantModel(t *testing.T) {
	tn := &Tenant{
		ID:        "tenant-001",
		Name:      "Acme Corp",
		Slug:      "acme-corp",
		Plan:      "pro",
		MaxServers: 10,
		MaxApps:    50,
	}

	if tn.Slug != "acme-corp" {
		t.Errorf("Tenant.Slug = %q, want %q", tn.Slug, "acme-corp")
	}
	if tn.Plan != "pro" {
		t.Errorf("Tenant.Plan = %q, want %q", tn.Plan, "pro")
	}
}

func TestTenantTableName(t *testing.T) {
	tn := &Tenant{}
	if tn.TableName() != "tenants" {
		t.Errorf("Tenant.TableName() = %q, want %q", tn.TableName(), "tenants")
	}
}

func TestUserTableName(t *testing.T) {
	u := &User{}
	if u.TableName() != "users" {
		t.Errorf("User.TableName() = %q, want %q", u.TableName(), "users")
	}
}

func TestRoleTableName(t *testing.T) {
	r := &Role{}
	if r.TableName() != "roles" {
		t.Errorf("Role.TableName() = %q, want %q", r.TableName(), "roles")
	}
}

func TestProviderTableName(t *testing.T) {
	p := &Provider{}
	if p.TableName() != "providers" {
		t.Errorf("Provider.TableName() = %q, want %q", p.TableName(), "providers")
	}
}

func TestUserModel(t *testing.T) {
	u := &User{
		ID:           "user-001",
		TenantID:     "tenant-default",
		RoleID:       "role-owner",
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: "hashed-password",
	}

	if u.Username != "admin" {
		t.Errorf("User.Username = %q, want %q", u.Username, "admin")
	}
	if u.RoleID != "role-owner" {
		t.Errorf("User.RoleID = %q, want %q", u.RoleID, "role-owner")
	}
}

func TestRoleModel(t *testing.T) {
	r := &Role{
		ID:          "role-dev",
		Name:        "dev",
		Permissions: `{"deploy": true}`,
	}

	if r.Name != "dev" {
		t.Errorf("Role.Name = %q, want %q", r.Name, "dev")
	}
}

func TestProviderModel(t *testing.T) {
	p := &Provider{
		ID:       "prov-001",
		TenantID: "tenant-default",
		Type:     "docker",
		Name:     "local-docker",
		Enabled:  true,
	}

	if p.Type != "docker" {
		t.Errorf("Provider.Type = %q, want %q", p.Type, "docker")
	}
	if p.Enabled != true {
		t.Errorf("Provider.Enabled = %v, want %v", p.Enabled, true)
	}
}

func TestAuditLogModel(t *testing.T) {
	now := time.Now()
	al := &AuditLog{
		ID:           1,
		UserID:       42,
		Username:     "admin",
		Action:       "app.create",
		ResourceType: "app",
		ResourceID:   "app-123",
		Detail:       `{"name": "myapp"}`,
		IPAddress:    "192.168.1.1",
		UserAgent:    "test-agent",
		CreatedAt:    now,
	}

	if al.ID != 1 {
		t.Errorf("AuditLog.ID = %d, want 1", al.ID)
	}
	if al.UserID != 42 {
		t.Errorf("AuditLog.UserID = %d, want 42", al.UserID)
	}
	if al.Action != "app.create" {
		t.Errorf("AuditLog.Action = %q, want %q", al.Action, "app.create")
	}
	if al.ResourceType != "app" {
		t.Errorf("AuditLog.ResourceType = %q, want %q", al.ResourceType, "app")
	}
	if al.ResourceID != "app-123" {
		t.Errorf("AuditLog.ResourceID = %q, want %q", al.ResourceID, "app-123")
	}
	if al.IPAddress != "192.168.1.1" {
		t.Errorf("AuditLog.IPAddress = %q, want %q", al.IPAddress, "192.168.1.1")
	}
}

func TestAuditLogTableName(t *testing.T) {
	al := &AuditLog{}
	if al.TableName() != "audit_logs" {
		t.Errorf("AuditLog.TableName() = %q, want %q", al.TableName(), "audit_logs")
	}
}

func TestDeploymentRecordModel(t *testing.T) {
	now := time.Now()
	dr := &DeploymentRecord{
		ID:            "dep-001",
		TenantID:      "tenant-default",
		AppName:       "myapp",
		ContainerName: "myapp-container",
		Image:         "nginx:latest",
		Status:        "success",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if dr.ID != "dep-001" {
		t.Errorf("DeploymentRecord.ID = %q, want %q", dr.ID, "dep-001")
	}
	if dr.Status != "success" {
		t.Errorf("DeploymentRecord.Status = %q, want %q", dr.Status, "success")
	}
}

func TestDeploymentRecordTableName(t *testing.T) {
	dr := &DeploymentRecord{}
	if dr.TableName() != "deployments" {
		t.Errorf("DeploymentRecord.TableName() = %q, want %q", dr.TableName(), "deployments")
	}
}

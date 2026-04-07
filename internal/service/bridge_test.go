package service

import (
	"context"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/database"
	"github.com/Yogdunana/deploypilot/internal/mcp"
	"gorm.io/gorm"
)

// mockExecutor implements deployer.CommandExecutor for testing.
type mockExecutor struct {
	output map[string]string // exact cmd → output
	err    map[string]error  // exact cmd → error
}

func (m *mockExecutor) RunCommand(_ context.Context, cmd string) (string, error) {
	if m.err != nil && m.err[cmd] != nil {
		return "", m.err[cmd]
	}
	if m.output != nil {
		if out, ok := m.output[cmd]; ok {
			return out, nil
		}
	}
	return "", nil
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	// Seed may fail on duplicate key — that's fine
	_ = database.Seed(db)
	return db
}

func newTestBridge(t *testing.T) (*Bridge, *mockExecutor) {
	t.Helper()
	db := setupTestDB(t)
	exec := &mockExecutor{
		output: map[string]string{
			"docker ps -a --format '{{.Names}}'": "",
			"echo ok":                              "ok",
			"docker version --format '{{.Server.Version}}' 2>/dev/null": "24.0",
			"docker info --format '{{.NCPU}}' 2>/dev/null":             "4",
			"docker info --format '{{.MemTotal}}' 2>/dev/null":          "8192000000",
			"uname -a": "Linux test 5.15 x86_64",
			"cat /etc/os-release 2>/dev/null | head -5": "PRETTY_NAME=\"Ubuntu 22.04\"",
		},
	}
	return NewBridge(db, exec), exec
}

// ===================== CRUD Tests =====================

func TestCreateApp(t *testing.T) {
	b, _ := newTestBridge(t)
	id, err := b.CreateApp(context.Background(), mcp.CreateAppConfig{
		Name:    "test-app",
		RepoURL: "https://github.com/test/test",
	})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty app ID")
	}
}

func TestListApps_Empty(t *testing.T) {
	b, _ := newTestBridge(t)
	apps, err := b.ListApps(context.Background())
	if err != nil {
		t.Fatalf("ListApps failed: %v", err)
	}
	if len(apps) != 0 {
		t.Fatalf("expected 0 apps, got %d", len(apps))
	}
}

func TestListApps(t *testing.T) {
	b, _ := newTestBridge(t)
	b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "app1", RepoURL: "https://a.com/a"})
	b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "app2", RepoURL: "https://b.com/b"})

	apps, err := b.ListApps(context.Background())
	if err != nil {
		t.Fatalf("ListApps failed: %v", err)
	}
	if len(apps) < 2 {
		t.Fatalf("expected >=2 apps, got %d", len(apps))
	}
}

func TestGetAppDetail(t *testing.T) {
	b, _ := newTestBridge(t)
	id, _ := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "detail-test", RepoURL: "https://x.com/x"})

	detail, err := b.GetAppDetail(context.Background(), id)
	if err != nil {
		t.Fatalf("GetAppDetail failed: %v", err)
	}
	m, ok := detail.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["name"] != "detail-test" {
		t.Fatalf("expected name=detail-test, got %v", m["name"])
	}
}

func TestGetAppDetail_NotFound(t *testing.T) {
	b, _ := newTestBridge(t)
	_, err := b.GetAppDetail(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent app")
	}
}

func TestUpdateApp(t *testing.T) {
	b, _ := newTestBridge(t)
	id, _ := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "update-test", RepoURL: "https://x.com/x"})

	result, err := b.UpdateApp(context.Background(), id, map[string]interface{}{"name": "updated-name"})
	if err != nil {
		t.Fatalf("UpdateApp failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["name"] != "updated-name" {
		t.Fatalf("expected name=updated-name, got %v", m["name"])
	}
}

func TestDeleteApp(t *testing.T) {
	b, _ := newTestBridge(t)
	id, _ := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "delete-test", RepoURL: "https://x.com/x"})

	err := b.DeleteApp(context.Background(), id)
	if err != nil {
		t.Fatalf("DeleteApp failed: %v", err)
	}

	_, err = b.GetAppDetail(context.Background(), id)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDeleteApp_NotFound(t *testing.T) {
	b, _ := newTestBridge(t)
	err := b.DeleteApp(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent app")
	}
}

func TestAddServer(t *testing.T) {
	b, _ := newTestBridge(t)
	si, err := b.AddServer(context.Background(), "test-server", "192.168.1.1", 22, "root")
	if err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}
	if si.ID == "" {
		t.Fatal("expected non-empty server ID")
	}
	if si.Host != "192.168.1.1" {
		t.Fatalf("expected host=192.168.1.1, got %s", si.Host)
	}
	if si.Status != "connected" {
		t.Fatalf("expected status=connected, got %s", si.Status)
	}
}

func TestListServers(t *testing.T) {
	b, _ := newTestBridge(t)
	b.AddServer(context.Background(), "s1", "10.0.0.1", 22, "root")
	b.AddServer(context.Background(), "s2", "10.0.0.2", 22, "root")

	servers, err := b.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(servers) < 2 {
		t.Fatalf("expected >=2 servers, got %d", len(servers))
	}
}

func TestRemoveServer(t *testing.T) {
	b, _ := newTestBridge(t)
	si, _ := b.AddServer(context.Background(), "rm-test", "10.0.0.3", 22, "root")

	err := b.RemoveServer(context.Background(), si.ID)
	if err != nil {
		t.Fatalf("RemoveServer failed: %v", err)
	}

	err = b.RemoveServer(context.Background(), si.ID)
	if err == nil {
		t.Fatal("expected error removing nonexistent server")
	}
}

func TestUpdateServer(t *testing.T) {
	b, _ := newTestBridge(t)
	si, _ := b.AddServer(context.Background(), "upd-server", "10.0.0.4", 22, "root")

	result, err := b.UpdateServer(context.Background(), si.ID, map[string]interface{}{"host": "10.0.0.99"})
	if err != nil {
		t.Fatalf("UpdateServer failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["host"] != "10.0.0.99" {
		t.Fatalf("expected host=10.0.0.99, got %v", m["host"])
	}
}

func TestCreateCredential(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.CreateCredential(context.Background(), "tenant-default", "my-cred", "ssh_key", "secret-value")
	if err != nil {
		t.Fatalf("CreateCredential failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["name"] != "my-cred" {
		t.Fatalf("expected name=my-cred, got %v", m["name"])
	}
}

func TestListCredentials(t *testing.T) {
	b, _ := newTestBridge(t)
	b.CreateCredential(context.Background(), "tenant-default", "cred1", "ssh", "val1")
	b.CreateCredential(context.Background(), "tenant-default", "cred2", "ssh", "val2")

	result, err := b.ListCredentials(context.Background(), "tenant-default")
	if err != nil {
		t.Fatalf("ListCredentials failed: %v", err)
	}
	list, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("expected []map")
	}
	if len(list) < 2 {
		t.Fatalf("expected >=2 credentials, got %d", len(list))
	}
	// Verify values are masked
	for _, c := range list {
		if _, hasVal := c["encrypted_value"]; hasVal {
			t.Fatal("credential values should be masked in list output")
		}
	}
}

func TestDeleteCredential(t *testing.T) {
	b, _ := newTestBridge(t)
	result, _ := b.CreateCredential(context.Background(), "tenant-default", "del-cred", "ssh", "val")
	id := result.(map[string]interface{})["id"].(string)

	err := b.DeleteCredential(context.Background(), id)
	if err != nil {
		t.Fatalf("DeleteCredential failed: %v", err)
	}

	err = b.DeleteCredential(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error deleting nonexistent credential")
	}
}

func TestUpdateCredential(t *testing.T) {
	b, _ := newTestBridge(t)
	result, _ := b.CreateCredential(context.Background(), "tenant-default", "upd-cred", "ssh", "old")
	id := result.(map[string]interface{})["id"].(string)

	updated, err := b.UpdateCredential(context.Background(), id, "new-value")
	if err != nil {
		t.Fatalf("UpdateCredential failed: %v", err)
	}
	m, ok := updated.(map[string]interface{})
	if !ok || m["status"] != "updated" {
		t.Fatal("expected status=updated")
	}
}

// ===================== Docker Delegation Tests =====================

func TestDeploy(t *testing.T) {
	b, exec := newTestBridge(t)
	// Mock the docker commands that DockerDeployer generates
	inspectOut := "abc123|test-deploy|nginx:alpine|running|2026-04-07T00:00:00Z"
	exec.output["docker pull nginx:alpine"] = "Downloaded"
	exec.output["docker rm -f test-deploy 2>/dev/null || true"] = ""
	exec.output["docker run -d --name test-deploy --restart unless-stopped nginx:alpine"] = "abc123"
	exec.output["docker inspect --format '{{.Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}|{{.Created}}' test-deploy 2>/dev/null"] = inspectOut

	cs, err := b.Deploy(context.Background(), mcp.DeployConfig{
		Image:         "nginx:alpine",
		ContainerName: "test-deploy",
		RestartPolicy: "unless-stopped",
	})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if cs.Status != "running" {
		t.Fatalf("expected status=running, got %s", cs.Status)
	}
}

func TestGetContainerStatus(t *testing.T) {
	b, exec := newTestBridge(t)
	inspectOut := "abc123|test-app|nginx:alpine|running|2026-04-07T00:00:00Z"
	exec.output["docker inspect --format '{{.Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}|{{.Created}}' test-app 2>/dev/null"] = inspectOut

	cs, err := b.GetContainerStatus(context.Background(), "test-app")
	if err != nil {
		t.Fatalf("GetContainerStatus failed: %v", err)
	}
	if cs.Name != "test-app" {
		t.Fatalf("expected name=test-app, got %s", cs.Name)
	}
}

func TestStop(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.output["docker stop test-stop"] = "test-stop"

	err := b.Stop(context.Background(), "test-stop")
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestGetContainerLogs(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.output["docker logs --tail 50 test-logs 2>&1"] = "line1\nline2\nline3"

	logs, err := b.GetContainerLogs(context.Background(), "test-logs", 50)
	if err != nil {
		t.Fatalf("GetContainerLogs failed: %v", err)
	}
	if logs != "line1\nline2\nline3" {
		t.Fatalf("unexpected logs: %s", logs)
	}
}

func TestHealthCheck(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.output["curl -sf --max-time 5 http://localhost:80"] = "ok"

	result, err := b.HealthCheck(context.Background(), "http://localhost:80", "http")
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok || m["status"] != "healthy" {
		t.Fatalf("expected healthy, got %v", m)
	}
}

// ===================== Info/Static Tests =====================

func TestListTemplates(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.ListTemplates(context.Background())
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}
	list, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("expected []map")
	}
	if len(list) != 9 {
		t.Fatalf("expected 9 templates, got %d", len(list))
	}
}

func TestGetTemplate(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.GetTemplate(context.Background(), "go")
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok || m["name"] != "Go" {
		t.Fatalf("expected Go template, got %v", m)
	}
}

func TestGetTemplate_NotFound(t *testing.T) {
	b, _ := newTestBridge(t)
	_, err := b.GetTemplate(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}

func TestCheckSystemUpdate(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.CheckSystemUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckSystemUpdate failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok || m["update_available"] != false {
		t.Fatalf("expected update_available=false, got %v", m)
	}
}

func TestDetectEnv(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.DetectEnv(context.Background(), 2, nil, nil)
	if err != nil {
		t.Fatalf("DetectEnv failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if _, hasOS := m["os"]; !hasOS {
		t.Fatal("expected os field")
	}
	if _, hasDocker := m["docker_version"]; !hasDocker {
		t.Fatal("expected docker_version field")
	}
}

func TestDetectEnv_Level3(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.DetectEnv(context.Background(), 3, []int{80, 443}, nil)
	if err != nil {
		t.Fatalf("DetectEnv failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if _, hasPorts := m["ports"]; !hasPorts {
		t.Fatal("expected ports field at level 3")
	}
}

// ===================== Stub Tests =====================

func TestDNSStubs(t *testing.T) {
	b, _ := newTestBridge(t)
	ctx := context.Background()

	// DNSCreateRecord
	res, err := b.DNSCreateRecord(ctx, "example.com", "A", "www", "1.2.3.4")
	if err != nil {
		t.Fatalf("DNSCreateRecord failed: %v", err)
	}
	m, ok := res.(map[string]interface{})
	if !ok || m["status"] != "not_implemented" {
		t.Fatalf("expected not_implemented, got %v", m)
	}

	// DNSDeleteRecord
	err = b.DNSDeleteRecord(ctx, "some-id")
	if err != nil {
		t.Fatalf("DNSDeleteRecord failed: %v", err)
	}

	// DNSListRecords
	res, err = b.DNSListRecords(ctx, "example.com")
	if err != nil {
		t.Fatalf("DNSListRecords failed: %v", err)
	}
	m = res.(map[string]interface{})
	if m["status"] != "not_implemented" {
		t.Fatalf("expected not_implemented, got %v", m)
	}

	// UpdateDNSRecord
	res, err = b.UpdateDNSRecord(ctx, "example.com", "www", "A", "5.6.7.8")
	if err != nil {
		t.Fatalf("UpdateDNSRecord failed: %v", err)
	}
	m = res.(map[string]interface{})
	if m["status"] != "not_implemented" {
		t.Fatalf("expected not_implemented, got %v", m)
	}
}

func TestSendNotification(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.SendNotification(context.Background(), "deploy", "myapp", "server1", "success", "deployed ok")
	if err != nil {
		t.Fatalf("SendNotification failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok || m["status"] != "logged" {
		t.Fatalf("expected logged, got %v", m)
	}
}

func TestRestore(t *testing.T) {
	b, _ := newTestBridge(t)
	cs, err := b.Restore(context.Background(), "backup-123")
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	if cs.Status != "not_implemented" {
		t.Fatalf("expected not_implemented, got %s", cs.Status)
	}
}

func TestGetTaskStatus(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.GetTaskStatus(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetTaskStatus failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok || m["status"] != "pending" {
		t.Fatalf("expected pending, got %v", m)
	}
}

func TestListTasks(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.ListTasks(context.Background(), 10, "all")
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok || m["status"] != "not_implemented" {
		t.Fatalf("expected not_implemented, got %v", m)
	}
}

// ===================== Batch Tests =====================

func TestBatchBackup(t *testing.T) {
	b, _ := newTestBridge(t)
	id1, _ := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "batch-app-1", RepoURL: "https://a.com/a"})
	id2, _ := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "batch-app-2", RepoURL: "https://b.com/b"})

	result, err := b.BatchBackup(context.Background(), []string{id1, id2})
	if err != nil {
		t.Fatalf("BatchBackup failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok || m["total"] != 2 {
		t.Fatalf("expected total=2, got %v", m)
	}
}

func TestBatchDNS(t *testing.T) {
	b, _ := newTestBridge(t)
	records := []map[string]interface{}{
		{"domain": "example.com", "type": "A", "subdomain": "www", "value": "1.2.3.4"},
	}
	result, err := b.BatchDNS(context.Background(), records)
	if err != nil {
		t.Fatalf("BatchDNS failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok || m["total"] != 1 {
		t.Fatalf("expected total=1, got %v", m)
	}
}

// ===================== Other Tests =====================

func TestTestServer(t *testing.T) {
	b, _ := newTestBridge(t)
	si, _ := b.AddServer(context.Background(), "test-srv", "10.0.0.5", 22, "root")

	result, err := b.TestServer(context.Background(), si.ID)
	if err != nil {
		t.Fatalf("TestServer failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok || m["status"] != "reachable" {
		t.Fatalf("expected reachable, got %v", m)
	}
}

func TestTestServer_NotFound(t *testing.T) {
	b, _ := newTestBridge(t)
	_, err := b.TestServer(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}

func TestCheckDeployReadiness(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.CheckDeployReadiness(context.Background(), map[string]interface{}{
		"ports": "8080",
	})
	if err != nil {
		t.Fatalf("CheckDeployReadiness failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if _, hasDocker := m["checks"]; !hasDocker {
		t.Fatal("expected checks field")
	}
}

func TestSearchAppLogs(t *testing.T) {
	b, exec := newTestBridge(t)
	id, _ := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "log-app", RepoURL: "https://x.com/x"})
	exec.output["docker logs --tail 2000 log-app 2>&1"] = "hello world\nerror: something\ninfo: ok\nerror: another"

	result, err := b.SearchAppLogs(context.Background(), id, "error", 10)
	if err != nil {
		t.Fatalf("SearchAppLogs failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["match_count"] != 2 {
		t.Fatalf("expected 2 matches, got %v", m["match_count"])
	}
}

func TestBackup(t *testing.T) {
	b, _ := newTestBridge(t)
	id, _ := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "backup-app", RepoURL: "https://x.com/x"})

	backupID, err := b.Backup(context.Background(), id)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	if backupID == "" {
		t.Fatal("expected non-empty backup ID")
	}
}

func TestBackup_NotFound(t *testing.T) {
	b, _ := newTestBridge(t)
	_, err := b.Backup(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent app")
	}
}

// ===================== Helper Tests =====================

func TestToString(t *testing.T) {
	tests := []struct {
		input interface{}
		want  string
	}{
		{nil, ""},
		{"hello", "hello"},
		{[]byte("bytes"), "bytes"},
		{42, "42"},
	}
	for _, tt := range tests {
		got := toString(tt.input)
		if got != tt.want {
			t.Errorf("toString(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		input interface{}
		want  int
	}{
		{nil, 0},
		{42, 42},
		{int64(99), 99},
		{float64(3.7), 3},
		{"not-int", 0},
	}
	for _, tt := range tests {
		got := toInt(tt.input)
		if got != tt.want {
			t.Errorf("toInt(%v) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestDefaultVal(t *testing.T) {
	if defaultVal("", "fallback") != "fallback" {
		t.Error("expected fallback for empty string")
	}
	if defaultVal("value", "fallback") != "value" {
		t.Error("expected value for non-empty string")
	}
}

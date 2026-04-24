package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
			"echo ok":                            "ok",
			"docker version --format '{{.Server.Version}}' 2>/dev/null": "24.0",
			"docker info --format '{{.NCPU}}' 2>/dev/null":              "4",
			"docker info --format '{{.MemTotal}}' 2>/dev/null":          "8192000000",
			"uname -a": "Linux test 5.15 x86_64",
			"cat /etc/os-release 2>/dev/null | head -5": "PRETTY_NAME=\"Ubuntu 22.04\"",
		},
		err: map[string]error{},
	}
	return NewBridge(db, exec, []byte("01234567890123456789012345678901")), exec
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
	result, err := b.CreateCredential(context.Background(), "tenant-default", "del-cred", "ssh", "val")
	if err != nil {
		t.Fatalf("CreateCredential failed: %v", err)
	}
	id := result.(map[string]interface{})["id"].(string)

	err = b.DeleteCredential(context.Background(), id)
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
	result, err := b.CreateCredential(context.Background(), "tenant-default", "upd-cred", "ssh", "old")
	if err != nil {
		t.Fatalf("CreateCredential failed: %v", err)
	}
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

	// DNSCreateRecord - no DNS provider configured, returns error status
	res, err := b.DNSCreateRecord(ctx, "example.com", "A", "www", "1.2.3.4")
	if err != nil {
		t.Fatalf("DNSCreateRecord failed: %v", err)
	}
	m, ok := res.(map[string]interface{})
	if !ok || m["status"] != "error" {
		t.Fatalf("expected error status, got %v", m)
	}

	// DNSDeleteRecord - no DNS provider configured, returns error
	err = b.DNSDeleteRecord(ctx, "some-id")
	if err == nil {
		t.Fatal("expected error for DNSDeleteRecord without provider")
	}

	// DNSListRecords - no DNS provider configured, returns error status
	res, err = b.DNSListRecords(ctx, "example.com")
	if err != nil {
		t.Fatalf("DNSListRecords failed: %v", err)
	}
	m = res.(map[string]interface{})
	if m["status"] != "error" {
		t.Fatalf("expected error status, got %v", m)
	}

	// UpdateDNSRecord - no DNS provider configured, returns error status
	res, err = b.UpdateDNSRecord(ctx, "example.com", "www", "A", "5.6.7.8")
	if err != nil {
		t.Fatalf("UpdateDNSRecord failed: %v", err)
	}
	m = res.(map[string]interface{})
	if m["status"] != "error" {
		t.Fatalf("expected error status, got %v", m)
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
	// No backup mapping exists, should return error
	_, err := b.Restore(context.Background(), "backup-123")
	if err == nil {
		t.Fatal("expected error for nonexistent backup")
	}
}

func TestGetTaskStatus(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.GetTaskStatus(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetTaskStatus failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok || m["status"] != "not_found" {
		t.Fatalf("expected not_found, got %v", m)
	}
}

func TestListTasks(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.ListTasks(context.Background(), 10, "all")
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok || m["status"] != "success" {
		t.Fatalf("expected success, got %v", m)
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
	// Each record should have error status since no DNS provider is configured
	results, ok := m["results"].([]map[string]interface{})
	if !ok || len(results) != 1 {
		t.Fatalf("expected 1 result, got %v", m["results"])
	}
	if results[0]["status"] != "error" {
		t.Fatalf("expected error status for record without provider, got %v", results[0]["status"])
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

func TestTestServer_Suggestions(t *testing.T) {
	b, exec := newTestBridge(t)
	// Make executor fail to simulate unreachable server
	exec.err = map[string]error{"echo ok": fmt.Errorf("connection refused")}
	srv, _ := b.AddServer(context.Background(), "unreachable-srv", "10.0.0.99", 22, "root")

	result, err := b.TestServer(context.Background(), srv.ID)
	if err != nil {
		t.Fatalf("TestServer failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["status"] != "unreachable" {
		t.Fatalf("expected unreachable, got %v", m["status"])
	}
	suggestions, ok := m["suggestions"].([]string)
	if !ok || len(suggestions) == 0 {
		t.Fatal("expected suggestions for unreachable server")
	}
}

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

func TestDeploy_PreflightFail_DockerUnavailable(t *testing.T) {
	b, _ := newTestBridge(t)
	// Make executor fail on docker version (simulates Docker not installed)
	exec := b.Executor.(*mockExecutor)
	exec.output["docker version --format '{{.Server.Version}}'"] = ""
	exec.err["docker version --format '{{.Server.Version}}'"] = fmt.Errorf("command not found")

	_, err := b.Deploy(context.Background(), mcp.DeployConfig{
		Image:         "nginx:alpine",
		ContainerName: "preflight-fail-test",
	})
	if err == nil {
		t.Fatal("expected preflight error")
	}
	var pfErr *PreflightError
	if !errors.As(err, &pfErr) {
		t.Fatalf("expected PreflightError, got: %T: %v", err, err)
	}
	if pfErr.Code != PreflightDockerUnavailable {
		t.Errorf("expected code %s, got %s", PreflightDockerUnavailable, pfErr.Code)
	}
	if pfErr.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestDeploy_PreflightPass_Local(t *testing.T) {
	b, _ := newTestBridge(t)
	exec := b.Executor.(*mockExecutor)
	exec.output["docker version --format '{{.Server.Version}}'"] = "24.0.7"
	exec.output["docker pull nginx:alpine"] = "pulled"
	exec.output["docker run -d --name preflight-ok-test --restart no nginx:alpine"] = "abc123"

	_, err := b.Deploy(context.Background(), mcp.DeployConfig{
		Image:         "nginx:alpine",
		ContainerName: "preflight-ok-test",
	})
	// May still fail at deploy stage (mock doesn't handle all docker commands),
	// but should NOT be a preflight error
	if err != nil {
		var pfErr *PreflightError
		if errors.As(err, &pfErr) {
			t.Fatalf("should not be a preflight error, got: %s", pfErr.Code)
		}
		// Non-preflight error is OK (mock limitations)
	}
}

func TestPreflightError_Methods(t *testing.T) {
	pfErr := &PreflightError{
		Code:    PreflightDockerUnavailable,
		Message: "Docker not found",
		Checks:  []PreflightCheck{{Name: "Docker", Passed: false, Suggestion: "Install Docker"}},
	}
	if pfErr.PreflightCode() != string(PreflightDockerUnavailable) {
		t.Errorf("expected code %s, got %s", PreflightDockerUnavailable, pfErr.PreflightCode())
	}
	if pfErr.PreflightMessage() != "Docker not found" {
		t.Errorf("expected message 'Docker not found', got %s", pfErr.PreflightMessage())
	}
	checks := pfErr.PreflightChecks()
	if checks == nil {
		t.Fatal("expected non-nil checks")
	}
}

func TestRemove_Success(t *testing.T) {
	b, _ := newTestBridge(t)
	exec := b.Executor.(*mockExecutor)
	exec.output["docker rm -f test-rm"] = "test-rm"
	err := b.Remove(context.Background(), "test-rm")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
}

func TestDetectEnv_Level1(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.DetectEnv(context.Background(), 1, nil, nil)
	if err != nil {
		t.Fatalf("DetectEnv failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if _, ok := m["os"]; !ok {
		t.Error("expected 'os' key in result")
	}
}

// ===================== Additional Coverage Tests =====================

func TestDeploy_ServerNotFound(t *testing.T) {
	b, _ := newTestBridge(t)
	_, err := b.Deploy(context.Background(), mcp.DeployConfig{
		Image:         "nginx:alpine",
		ContainerName: "test-srv-not-found",
		ServerID:      "nonexistent-server-id",
	})
	if err == nil {
		t.Fatal("expected error when server not found")
	}
	if err != nil && !strings.Contains(err.Error(), "server not found") {
		t.Errorf("expected 'server not found' error, got: %v", err)
	}
}

func TestDeploy_DeployFailure(t *testing.T) {
	b, exec := newTestBridge(t)
	// Mock preflight to pass, but deploy to fail
	exec.output["docker version --format '{{.Server.Version}}'"] = "24.0"
	exec.output["docker pull nginx:alpine"] = "Downloaded"
	exec.output["docker rm -f deploy-fail-test 2>/dev/null || true"] = ""
	exec.err["docker run -d --name deploy-fail-test --restart unless-stopped nginx:alpine"] = fmt.Errorf("port already in use")

	_, err := b.Deploy(context.Background(), mcp.DeployConfig{
		Image:         "nginx:alpine",
		ContainerName: "deploy-fail-test",
	})
	if err == nil {
		t.Fatal("expected deploy error")
	}
	var pfErr *PreflightError
	if errors.As(err, &pfErr) {
		t.Fatalf("should not be a preflight error, got: %s", pfErr.Code)
	}
}

func TestDeploy_DeploySuccess_FullConfig(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.output["docker version --format '{{.Server.Version}}'"] = "24.0"
	exec.output["docker pull nginx:alpine"] = "Downloaded"
	exec.output["docker rm -f full-cfg-test 2>/dev/null || true"] = ""
	exec.output["docker run -d --name full-cfg-test --restart unless-stopped -p 8080:80 -e FOO=bar -l app=test nginx:alpine"] = "container-id-123"
	exec.output["docker inspect --format '{{.Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}|{{.Created}}' full-cfg-test 2>/dev/null"] = "container-id-123|full-cfg-test|nginx:alpine|running|2026-04-07T00:00:00Z"

	cs, err := b.Deploy(context.Background(), mcp.DeployConfig{
		Image:         "nginx:alpine",
		ContainerName: "full-cfg-test",
		Ports:         "8080:80",
		EnvVars:       map[string]string{"FOO": "bar"},
		Labels:        map[string]string{"app": "test"},
		RestartPolicy: "unless-stopped",
	})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if cs.ID != "container-id-123" {
		t.Errorf("expected container-id-123, got %s", cs.ID)
	}
	if cs.Status != "running" {
		t.Errorf("expected running, got %s", cs.Status)
	}
}

func TestGetContainerStatus_Error(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.err["docker inspect --format '{{.Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}|{{.Created}}' nonexistent 2>/dev/null"] = fmt.Errorf("not found")

	_, err := b.GetContainerStatus(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent container")
	}
}

func TestListApps_WithEnvVars(t *testing.T) {
	b, _ := newTestBridge(t)
	// Create an app, then manually update it with env_vars
	id, _ := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "env-app", RepoURL: "https://x.com/x"})
	b.DB.Table("apps").Where("id = ?", id).Update("env_vars", `{"key":"value"}`)

	apps, err := b.ListApps(context.Background())
	if err != nil {
		t.Fatalf("ListApps failed: %v", err)
	}
	found := false
	for _, app := range apps {
		if app.Name == "env-app" {
			found = true
			if app.Labels == nil || app.Labels["key"] != "value" {
				t.Errorf("expected env_vars parsed into Labels, got %v", app.Labels)
			}
		}
	}
	if !found {
		t.Fatal("env-app not found in list")
	}
}

func TestListApps_WithInvalidEnvVars(t *testing.T) {
	b, _ := newTestBridge(t)
	id, _ := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "bad-env-app", RepoURL: "https://x.com/x"})
	b.DB.Table("apps").Where("id = ?", id).Update("env_vars", "not-json")

	apps, err := b.ListApps(context.Background())
	if err != nil {
		t.Fatalf("ListApps failed: %v", err)
	}
	// Should not panic, just skip the invalid JSON
	for _, app := range apps {
		if app.Name == "bad-env-app" {
			// Labels should be nil since JSON parse fails
			if app.Labels != nil {
				t.Errorf("expected nil Labels for invalid env_vars, got %v", app.Labels)
			}
		}
	}
}

func TestDeleteApp_WithContainer(t *testing.T) {
	b, exec := newTestBridge(t)
	id, _ := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "del-with-container", RepoURL: "https://x.com/x"})
	// Set container_name on the app
	b.DB.Table("apps").Where("id = ?", id).Update("container_name", "del-container")
	exec.output["docker stop del-container"] = ""
	exec.output["docker rm -f del-container"] = ""

	err := b.DeleteApp(context.Background(), id)
	if err != nil {
		t.Fatalf("DeleteApp failed: %v", err)
	}
}

func TestDeleteApp_DBDeleteError(t *testing.T) {
	b, _ := newTestBridge(t)
	// Try to delete an app that doesn't exist (already handled by Take, but test the flow)
	err := b.DeleteApp(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent app")
	}
}

func TestDetectEnv_Level4(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.output["timeout 2 bash -c 'echo > /dev/tcp/localhost:80' 2>/dev/null && echo ok || echo fail"] = "ok"

	result, err := b.DetectEnv(context.Background(), 4, []int{80}, []string{"localhost:80"})
	if err != nil {
		t.Fatalf("DetectEnv failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if _, hasPorts := m["ports"]; !hasPorts {
		t.Fatal("expected ports at level 4")
	}
	if _, hasServices := m["services"]; !hasServices {
		t.Fatal("expected services at level 4")
	}
}

func TestDetectEnv_Level4_ServiceFail(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.output["timeout 2 bash -c 'echo > /dev/tcp/unreachable:9999' 2>/dev/null && echo ok || echo fail"] = "fail"

	result, err := b.DetectEnv(context.Background(), 4, nil, []string{"unreachable:9999"})
	if err != nil {
		t.Fatalf("DetectEnv failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	services := m["services"].(map[string]bool)
	if services["unreachable:9999"] {
		t.Error("expected service to be false (unreachable)")
	}
}

func TestDetectEnv_Level4_EmptyService(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.DetectEnv(context.Background(), 4, nil, []string{"  ", ""})
	if err != nil {
		t.Fatalf("DetectEnv failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	services := m["services"].(map[string]bool)
	if len(services) != 0 {
		t.Errorf("expected 0 services for empty/whitespace input, got %d", len(services))
	}
}

func TestDetectEnv_Level2_Error(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.err["docker version --format '{{.Server.Version}}' 2>/dev/null"] = fmt.Errorf("not found")
	exec.err["docker info --format '{{.NCPU}}' 2>/dev/null"] = fmt.Errorf("not found")
	exec.err["docker info --format '{{.MemTotal}}' 2>/dev/null"] = fmt.Errorf("not found")

	result, err := b.DetectEnv(context.Background(), 2, nil, nil)
	if err != nil {
		t.Fatalf("DetectEnv failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	// Docker info should be absent when commands fail
	if _, has := m["docker_version"]; has {
		t.Error("expected no docker_version when command fails")
	}
}

func TestDetectEnv_Level3_PortInUse(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.output["ss -tlnp 2>/dev/null | grep ':80 ' || true"] = "LISTEN  0  128  *:80  *:*"

	result, err := b.DetectEnv(context.Background(), 3, []int{80}, nil)
	if err != nil {
		t.Fatalf("DetectEnv failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	ports := m["ports"].(map[string]bool)
	if !ports["80"] {
		t.Error("expected port 80 to be in use")
	}
}

func TestCreateCredential_DifferentTypes(t *testing.T) {
	b, _ := newTestBridge(t)
	types := []string{"ssh_key", "password", "token", "api_key"}
	for _, credType := range types {
		result, err := b.CreateCredential(context.Background(), "tenant-default", "cred-"+credType, credType, "secret-"+credType)
		if err != nil {
			t.Fatalf("CreateCredential(%s) failed: %v", credType, err)
		}
		m, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map for %s", credType)
		}
		if m["type"] != credType {
			t.Errorf("expected type=%s, got %v", credType, m["type"])
		}
	}
}

func TestCreateCredential_EncryptionError(t *testing.T) {
	b, _ := newTestBridge(t)
	// Use a bridge with a nil encryption key to trigger encryption error
	b.EncryptionKey = nil
	_, err := b.CreateCredential(context.Background(), "tenant-default", "bad-cred", "ssh_key", "value")
	if err == nil {
		t.Fatal("expected error with nil encryption key")
	}
}

func TestUpdateApp_NotFound(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.UpdateApp(context.Background(), "nonexistent-app-id", map[string]interface{}{"name": "new-name"})
	if err != nil {
		t.Fatalf("UpdateApp should not error on update even if not found: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["status"] != "updated" {
		t.Errorf("expected status=updated for fallback, got %v", m["status"])
	}
}

func TestUpdateCredential_NotFound(t *testing.T) {
	b, _ := newTestBridge(t)
	_, err := b.UpdateCredential(context.Background(), "nonexistent-cred-id", "new-value")
	if err != nil {
		t.Fatalf("UpdateCredential should succeed even if not found: %v", err)
	}
}

func TestUpdateCredential_EncryptionError(t *testing.T) {
	b, _ := newTestBridge(t)
	b.EncryptionKey = nil
	_, err := b.UpdateCredential(context.Background(), "some-id", "value")
	if err == nil {
		t.Fatal("expected error with nil encryption key")
	}
}

func TestUpdateServer_NotFound(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.UpdateServer(context.Background(), "nonexistent-server-id", map[string]interface{}{"host": "new-host"})
	if err != nil {
		t.Fatalf("UpdateServer should not error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["status"] != "updated" {
		t.Errorf("expected status=updated for fallback, got %v", m["status"])
	}
}

func TestCheckDeployReadiness_DockerUnavailable(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.err["docker version --format '{{.Server.Version}}' 2>/dev/null"] = fmt.Errorf("command not found")

	result, err := b.CheckDeployReadiness(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("CheckDeployReadiness failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["ready"] != false {
		t.Error("expected ready=false when docker unavailable")
	}
}

func TestCheckDeployReadiness_PortInUse(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.output["ss -tlnp 2>/dev/null | grep ':8080 ' || true"] = "LISTEN  0  128  *:8080  *:*"

	result, err := b.CheckDeployReadiness(context.Background(), map[string]interface{}{
		"ports": "8080:80",
	})
	if err != nil {
		t.Fatalf("CheckDeployReadiness failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["ready"] != false {
		t.Error("expected ready=false when port in use")
	}
}

func TestCheckDeployReadiness_PortWithColon(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.output["ss -tlnp 2>/dev/null | grep ':9090 ' || true"] = ""

	result, err := b.CheckDeployReadiness(context.Background(), map[string]interface{}{
		"ports": "9090:80",
	})
	if err != nil {
		t.Fatalf("CheckDeployReadiness failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["ready"] != true {
		t.Error("expected ready=true when port is free")
	}
}

func TestCheckDeployReadiness_NoPorts(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.CheckDeployReadiness(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("CheckDeployReadiness failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["ready"] != true {
		t.Error("expected ready=true when no ports to check and docker available")
	}
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.err["curl -sf --max-time 5 http://localhost:9999"] = fmt.Errorf("connection refused")

	result, err := b.HealthCheck(context.Background(), "http://localhost:9999", "http")
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["status"] != "unhealthy" {
		t.Errorf("expected unhealthy, got %v", m["status"])
	}
}

func TestHealthCheck_TCP(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.output["timeout 5 bash -c 'echo > /dev/tcp/localhost:3306' 2>/dev/null"] = ""

	result, err := b.HealthCheck(context.Background(), "tcp://localhost:3306", "tcp")
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["status"] != "healthy" {
		t.Errorf("expected healthy, got %v", m["status"])
	}
}

func TestHealthCheck_DefaultType(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.output["echo ok"] = "ok"

	result, err := b.HealthCheck(context.Background(), "http://localhost:80", "")
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["type"] != "http" {
		t.Errorf("expected default type http, got %v", m["type"])
	}
}

func TestAddServer_Unreachable(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.err["echo ok"] = fmt.Errorf("connection refused")

	si, err := b.AddServer(context.Background(), "unreachable", "10.0.0.99", 22, "root")
	if err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}
	if si.Status != "unknown" {
		t.Errorf("expected status=unknown, got %s", si.Status)
	}
}

func TestListServers_Empty(t *testing.T) {
	b, _ := newTestBridge(t)
	servers, err := b.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(servers) != 0 {
		t.Fatalf("expected 0 servers, got %d", len(servers))
	}
}

func TestListCredentials_Empty(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.ListCredentials(context.Background(), "nonexistent-tenant")
	if err != nil {
		t.Fatalf("ListCredentials failed: %v", err)
	}
	list, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("expected []map")
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 credentials, got %d", len(list))
	}
}

func TestSearchAppLogs_NotFound(t *testing.T) {
	b, _ := newTestBridge(t)
	_, err := b.SearchAppLogs(context.Background(), "nonexistent-app-id", "error", 10)
	if err == nil {
		t.Fatal("expected error for nonexistent app")
	}
}

func TestSearchAppLogs_ContainerNameFallback(t *testing.T) {
	b, exec := newTestBridge(t)
	id, _ := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "fallback-app", RepoURL: "https://x.com/x"})
	// container_name is empty, so it should fall back to name
	exec.output["docker logs --tail 2000 fallback-app 2>&1"] = "error line\ninfo line"

	result, err := b.SearchAppLogs(context.Background(), id, "error", 10)
	if err != nil {
		t.Fatalf("SearchAppLogs failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["match_count"] != 1 {
		t.Errorf("expected 1 match, got %v", m["match_count"])
	}
}

func TestSearchAppLogs_Limit(t *testing.T) {
	b, exec := newTestBridge(t)
	id, _ := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "limit-app", RepoURL: "https://x.com/x"})
	exec.output["docker logs --tail 2000 limit-app 2>&1"] = "error1\nerror2\nerror3"

	result, err := b.SearchAppLogs(context.Background(), id, "error", 2)
	if err != nil {
		t.Fatalf("SearchAppLogs failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["match_count"] != 2 {
		t.Errorf("expected 2 matches (limited), got %v", m["match_count"])
	}
}

func TestBackup_WithContainerName(t *testing.T) {
	b, exec := newTestBridge(t)
	id, _ := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "backup-cn-app", RepoURL: "https://x.com/x"})
	b.DB.Table("apps").Where("id = ?", id).Update("container_name", "backup-cn")
	exec.output["docker exec backup-cn sh -c 'tar czf - /app /data 2>/dev/null' > /tmp/backup-backup-cn-*.tar.gz 2>/dev/null || echo 'no_backup_paths'"] = "no_backup_paths"

	backupID, err := b.Backup(context.Background(), id)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	if backupID == "" {
		t.Fatal("expected non-empty backup ID")
	}
}

func TestBatchBackup_NotFound(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.BatchBackup(context.Background(), []string{"nonexistent"})
	if err != nil {
		t.Fatalf("BatchBackup failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["total"] != 1 {
		t.Errorf("expected total=1, got %v", m["total"])
	}
}

func TestCreateApp_WithOptions(t *testing.T) {
	b, _ := newTestBridge(t)
	id, err := b.CreateApp(context.Background(), mcp.CreateAppConfig{
		Name:       "full-app",
		RepoURL:    "https://github.com/test/test",
		Branch:     "develop",
		Domain:     "test.example.com",
		TechStack:  "node",
		DeployMode: "docker",
		ServerID:   "srv-1",
	})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty app ID")
	}
	// Verify the record
	detail, err := b.GetAppDetail(context.Background(), id)
	if err != nil {
		t.Fatalf("GetAppDetail failed: %v", err)
	}
	m := detail.(map[string]interface{})
	if m["branch"] != "develop" {
		t.Errorf("expected branch=develop, got %v", m["branch"])
	}
	if m["tech_stack"] != "node" {
		t.Errorf("expected tech_stack=node, got %v", m["tech_stack"])
	}
}

func TestToStringOrDefault(t *testing.T) {
	tests := []struct {
		input interface{}
		def   string
		want  string
	}{
		{nil, "default", "default"},
		{"hello", "default", "hello"},
		{"", "default", "default"},
		{42, "default", "42"},
	}
	for _, tt := range tests {
		got := toStringOrDefault(tt.input, tt.def)
		if got != tt.want {
			t.Errorf("toStringOrDefault(%v, %q) = %q, want %q", tt.input, tt.def, got, tt.want)
		}
	}
}

func TestDeploy_WithServerID(t *testing.T) {
	b, _ := newTestBridge(t)
	// Add a server first
	si, _ := b.AddServer(context.Background(), "remote-server", "10.0.0.1", 22, "root")
	// Deploy with server ID - will fail at getRemoteExecutor since we can't SSH
	_, err := b.Deploy(context.Background(), mcp.DeployConfig{
		Image:         "nginx:alpine",
		ContainerName: "remote-deploy-test",
		ServerID:      si.ID,
	})
	// Should fail at SSH connection, not at server lookup
	if err == nil {
		t.Fatal("expected error for remote deploy without real SSH")
	}
}

func TestDeploy_PreflightFail_PortConflict(t *testing.T) {
	b, exec := newTestBridge(t)
	exec.output["docker version --format '{{.Server.Version}}'"] = "24.0"
	exec.output["ss -tlnp 2>/dev/null | grep ':8080 ' || true"] = "LISTEN  0  128  *:8080  *:*"

	_, err := b.Deploy(context.Background(), mcp.DeployConfig{
		Image:         "nginx:alpine",
		ContainerName: "port-conflict-test",
		Ports:         "8080:80",
	})
	if err == nil {
		t.Fatal("expected preflight error for port conflict")
	}
	var pfErr *PreflightError
	if !errors.As(err, &pfErr) {
		t.Fatalf("expected PreflightError, got: %T: %v", err, err)
	}
	if pfErr.Code != PreflightPortInUse {
		t.Errorf("expected code %s, got %s", PreflightPortInUse, pfErr.Code)
	}
}

func TestDetectEnv_Level0(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.DetectEnv(context.Background(), 0, nil, nil)
	if err != nil {
		t.Fatalf("DetectEnv failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if len(m) != 0 {
		t.Errorf("expected empty map for level 0, got %v", m)
	}
}

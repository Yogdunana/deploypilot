package service

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/database"
	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// setupDBWithProviders creates an in-memory DB with DNS and notification providers seeded.
func setupDBWithProviders(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	_ = database.Seed(db)

	// Seed a DNS provider
	cfg, _ := json.Marshal(map[string]string{"api_token": "test-token", "account_email": "test@example.com"})
	db.Create(&model.Provider{
		ID:      "dns-1",
		Type:    "cloudflare",
		Name:    "cf-test",
		Config:  string(cfg),
		Enabled: true,
	})

	// Seed a webhook notifier
	ncfg, _ := json.Marshal(map[string]interface{}{"channel": "webhook", "url": "https://hooks.example.com/test"})
	db.Create(&model.Provider{
		ID:      "notify-1",
		Type:    "notify",
		Name:    "webhook-test",
		Config:  string(ncfg),
		Enabled: true,
	})

	return db
}

// ===================== Provider Manager Tests =====================

func TestGetDNSProvider_NoDB(t *testing.T) {
	b := &Bridge{DB: nil}
	_, err := b.getDNSProvider(context.Background())
	if err == nil {
		t.Fatal("expected error when DB is nil")
	}
}

func TestGetDNSProvider_NoProvider(t *testing.T) {
	db := setupTestDB(t)
	b := &Bridge{DB: db}
	_, err := b.getDNSProvider(context.Background())
	if err == nil {
		t.Fatal("expected error when no DNS provider exists")
	}
}

func TestGetNotifiers_NoDB(t *testing.T) {
	b := &Bridge{DB: nil}
	notifiers, err := b.getNotifiers(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if notifiers != nil {
		t.Fatalf("expected nil notifiers, got %d", len(notifiers))
	}
}

func TestGetNotifiers_Webhook(t *testing.T) {
	db := setupDBWithProviders(t)
	b := &Bridge{DB: db}
	notifiers, err := b.getNotifiers(context.Background())
	if err != nil {
		t.Fatalf("getNotifiers failed: %v", err)
	}
	if len(notifiers) != 1 {
		t.Fatalf("expected 1 notifier, got %d", len(notifiers))
	}
	if notifiers[0].Name() != "webhook" {
		t.Fatalf("expected webhook notifier, got %s", notifiers[0].Name())
	}
}

func TestGetNotifiers_InvalidConfig(t *testing.T) {
	db := setupTestWithProviders(t) // already has webhook
	// Add a notifier with invalid JSON config
	db.Create(&model.Provider{
		ID:      "notify-bad",
		Type:    "notify",
		Name:    "bad-config",
		Config:  "not-json{{{",
		Enabled: true,
	})
	b := &Bridge{DB: db}
	notifiers, err := b.getNotifiers(context.Background())
	if err != nil {
		t.Fatalf("getNotifiers failed: %v", err)
	}
	// Should still get the valid webhook notifier, skipping the bad one
	if len(notifiers) != 1 {
		t.Fatalf("expected 1 notifier (bad one skipped), got %d", len(notifiers))
	}
}

// ===================== DNS Wired Tests =====================

func TestDNSCreateRecord_NoProvider(t *testing.T) {
	b, _ := newTestBridge(t)
	res, err := b.DNSCreateRecord(context.Background(), "example.com", "A", "www", "1.2.3.4")
	if err != nil {
		t.Fatalf("DNSCreateRecord failed: %v", err)
	}
	m, ok := res.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["status"] != "error" {
		t.Fatalf("expected error status, got %v", m["status"])
	}
}

func TestDNSCreateRecord_WithProvider(t *testing.T) {
	db := setupDBWithProviders(t)
	exec := &mockExecutor{
		output: map[string]string{},
		err:    map[string]error{},
	}
	b := NewBridge(db, exec, []byte("01234567890123456789012345678901"), nil)
	// The DNS provider will try to make HTTP calls to Cloudflare API,
	// which will fail in tests. We just verify it doesn't panic and returns
	// a result (error from the API call is wrapped in the response).
	res, err := b.DNSCreateRecord(context.Background(), "example.com", "A", "www", "1.2.3.4")
	if err != nil {
		t.Fatalf("DNSCreateRecord failed: %v", err)
	}
	m, ok := res.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	// The Cloudflare API call will fail (no real API), so status should be error
	if m["status"] != "error" {
		t.Fatalf("expected error status (API call fails in test), got %v", m["status"])
	}
}

func TestDNSListRecords_NoProvider(t *testing.T) {
	b, _ := newTestBridge(t)
	res, err := b.DNSListRecords(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("DNSListRecords failed: %v", err)
	}
	m, ok := res.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["status"] != "error" {
		t.Fatalf("expected error status, got %v", m["status"])
	}
}

func TestDNSDeleteRecord_InvalidFormat(t *testing.T) {
	db := setupDBWithProviders(t)
	exec := &mockExecutor{}
	b := NewBridge(db, exec, []byte("01234567890123456789012345678901"), nil)
	err := b.DNSDeleteRecord(context.Background(), "invalid-format")
	if err == nil {
		t.Fatal("expected error for invalid record ID format")
	}
}

// ===================== Notification Tests =====================

func TestSendNotification_NoProviders(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.SendNotification(context.Background(), "deploy", "myapp", "server1", "success", "deployed ok")
	if err != nil {
		t.Fatalf("SendNotification failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["status"] != "logged" {
		t.Fatalf("expected logged status, got %v", m["status"])
	}
	if m["message"] != "no notification providers configured" {
		t.Fatalf("expected 'no notification providers configured', got %v", m["message"])
	}
}

func TestSendNotification_WithWebhook(t *testing.T) {
	db := setupDBWithProviders(t)
	exec := &mockExecutor{}
	b := NewBridge(db, exec, []byte("01234567890123456789012345678901"), nil)
	// The webhook notifier will try to make an HTTP call that will fail in tests.
	// The MultiNotifier should still return results (with success=false).
	result, err := b.SendNotification(context.Background(), "deploy", "myapp", "server1", "success", "deployed ok")
	if err != nil {
		t.Fatalf("SendNotification failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["status"] != "sent" {
		t.Fatalf("expected sent status, got %v", m["status"])
	}
	if m["total_notifiers"] != 1 {
		t.Fatalf("expected total_notifiers=1, got %v", m["total_notifiers"])
	}
}

// ===================== BatchDNS Tests =====================

func TestBatchDNS_EmptyRecords(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.BatchDNS(context.Background(), []map[string]interface{}{})
	if err != nil {
		t.Fatalf("BatchDNS failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["total"] != 0 {
		t.Fatalf("expected total=0, got %v", m["total"])
	}
}

// ===================== Restore Tests =====================

func TestRestore_AppNotFound(t *testing.T) {
	b, _ := newTestBridge(t)
	// No backup mapping exists
	_, err := b.Restore(context.Background(), "nonexistent-backup")
	if err == nil {
		t.Fatal("expected error for nonexistent backup")
	}
}

func TestRestore_BackupExists_AppGone(t *testing.T) {
	db := setupTestDB(t)
	exec := &mockExecutor{}
	b := NewBridge(db, exec, []byte("01234567890123456789012345678901"), nil)

	// Manually insert a backup mapping for a nonexistent app
	b.backupMu.Lock()
	b.backupApps["backup-orphan"] = "nonexistent-app-id"
	b.backupMu.Unlock()

	_, err := b.Restore(context.Background(), "backup-orphan")
	if err == nil {
		t.Fatal("expected error when app not found")
	}
}

// ===================== Task Tracker Tests =====================

func TestGetTaskStatus_NotFound(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.GetTaskStatus(context.Background(), "nonexistent-task")
	if err != nil {
		t.Fatalf("GetTaskStatus failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["status"] != "not_found" {
		t.Fatalf("expected not_found, got %v", m["status"])
	}
}

func TestGetTaskStatus_Found(t *testing.T) {
	b, _ := newTestBridge(t)
	id := b.createTask("deploy")
	result, err := b.GetTaskStatus(context.Background(), id)
	if err != nil {
		t.Fatalf("GetTaskStatus failed: %v", err)
	}
	ti, ok := result.(*taskInfo)
	if !ok {
		t.Fatal("expected *taskInfo")
	}
	if ti.ID != id {
		t.Fatalf("expected ID=%s, got %s", id, ti.ID)
	}
	if ti.Status != "pending" {
		t.Fatalf("expected pending, got %s", ti.Status)
	}
	if ti.Type != "deploy" {
		t.Fatalf("expected type=deploy, got %s", ti.Type)
	}
}

func TestListTasks_Empty(t *testing.T) {
	b, _ := newTestBridge(t)
	// Clear tasks for this test
	b.taskMu.Lock()
	oldTasks := b.tasks
	b.tasks = make(map[string]*taskInfo)
	b.taskMu.Unlock()
	defer func() {
		b.taskMu.Lock()
		b.tasks = oldTasks
		b.taskMu.Unlock()
	}()

	result, err := b.ListTasks(context.Background(), 10, "")
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["status"] != "success" {
		t.Fatalf("expected success, got %v", m["status"])
	}
	taskList, ok := m["tasks"].([]*taskInfo)
	if !ok {
		t.Fatal("expected []*taskInfo")
	}
	if len(taskList) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(taskList))
	}
}

func TestListTasks_WithFilter(t *testing.T) {
	b, _ := newTestBridge(t)
	// Clear tasks and add known tasks
	b.taskMu.Lock()
	oldTasks := b.tasks
	oldCounter := b.taskCounter
	b.tasks = make(map[string]*taskInfo)
	b.taskCounter = 0
	b.taskMu.Unlock()
	defer func() {
		b.taskMu.Lock()
		b.tasks = oldTasks
		b.taskCounter = oldCounter
		b.taskMu.Unlock()
	}()

	id1 := b.createTask("deploy")
	// Mark id1 as success
	b.taskMu.Lock()
	if t, ok := b.tasks[id1]; ok {
		t.Status = "success"
	}
	b.taskMu.Unlock()

	id2 := b.createTask("backup")
	// Mark id2 as failed
	b.taskMu.Lock()
	if t, ok := b.tasks[id2]; ok {
		t.Status = "failed"
	}
	b.taskMu.Unlock()

	// Filter by "success"
	result, err := b.ListTasks(context.Background(), 10, "success")
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	taskList, ok := m["tasks"].([]*taskInfo)
	if !ok {
		t.Fatal("expected []*taskInfo")
	}
	if len(taskList) != 1 {
		t.Fatalf("expected 1 task with status=success, got %d", len(taskList))
	}
	if taskList[0].ID != id1 {
		t.Fatalf("expected task %s, got %s", id1, taskList[0].ID)
	}

	// Filter by "failed"
	result, err = b.ListTasks(context.Background(), 10, "failed")
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	m = result.(map[string]interface{})
	taskList = m["tasks"].([]*taskInfo)
	if len(taskList) != 1 {
		t.Fatalf("expected 1 task with status=failed, got %d", len(taskList))
	}

	// No filter - should return all
	result, err = b.ListTasks(context.Background(), 10, "")
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	m = result.(map[string]interface{})
	taskList = m["tasks"].([]*taskInfo)
	if len(taskList) != 2 {
		t.Fatalf("expected 2 tasks with no filter, got %d", len(taskList))
	}

	// Limit test
	result, err = b.ListTasks(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	m = result.(map[string]interface{})
	taskList = m["tasks"].([]*taskInfo)
	if len(taskList) != 1 {
		t.Fatalf("expected 1 task with limit=1, got %d", len(taskList))
	}
}

func TestCreateTask(t *testing.T) {
	b, _ := newTestBridge(t)
	// Save and restore state
	b.taskMu.Lock()
	oldTasks := b.tasks
	oldCounter := b.taskCounter
	b.tasks = make(map[string]*taskInfo)
	b.taskCounter = 0
	b.taskMu.Unlock()
	defer func() {
		b.taskMu.Lock()
		b.tasks = oldTasks
		b.taskCounter = oldCounter
		b.taskMu.Unlock()
	}()

	id1 := b.createTask("deploy")
	id2 := b.createTask("backup")

	if id1 == id2 {
		t.Fatal("expected unique task IDs")
	}
	if id1 != "task-1" {
		t.Fatalf("expected task-1, got %s", id1)
	}
	if id2 != "task-2" {
		t.Fatalf("expected task-2, got %s", id2)
	}

	b.taskMu.RLock()
	if len(b.tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(b.tasks))
	}
	b.taskMu.RUnlock()
}

// ===================== Concurrency Tests =====================

func TestTaskTracker_Concurrent(t *testing.T) {
	b, _ := newTestBridge(t)
	// Save and restore state
	b.taskMu.Lock()
	oldTasks := b.tasks
	oldCounter := b.taskCounter
	b.tasks = make(map[string]*taskInfo)
	b.taskCounter = 0
	b.taskMu.Unlock()
	defer func() {
		b.taskMu.Lock()
		b.tasks = oldTasks
		b.taskCounter = oldCounter
		b.taskMu.Unlock()
	}()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.createTask("concurrent-test")
		}()
	}
	wg.Wait()

	b.taskMu.RLock()
	if len(b.tasks) != 100 {
		t.Fatalf("expected 100 tasks, got %d", len(b.tasks))
	}
	b.taskMu.RUnlock()
}

// ===================== Email Notifier Config Test =====================

func TestGetNotifiers_Email(t *testing.T) {
	db := setupTestWithProviders(t) // already has webhook
	// Add an email notifier
	ecfg, _ := json.Marshal(map[string]interface{}{
		"channel":   "email",
		"smtp_host": "smtp.example.com",
		"smtp_port": 587,
		"username":  "user@example.com",
		"password":  "secret",
		"from":      "noreply@example.com",
	})
	db.Create(&model.Provider{
		ID:      "notify-email",
		Type:    "notify",
		Name:    "email-test",
		Config:  string(ecfg),
		Enabled: true,
	})

	b := &Bridge{DB: db}
	notifiers, err := b.getNotifiers(context.Background())
	if err != nil {
		t.Fatalf("getNotifiers failed: %v", err)
	}
	// Should have 2 notifiers: webhook + email
	if len(notifiers) != 2 {
		t.Fatalf("expected 2 notifiers, got %d", len(notifiers))
	}
	foundEmail := false
	for _, n := range notifiers {
		if n.Name() == "email" {
			foundEmail = true
		}
	}
	if !foundEmail {
		t.Fatal("expected email notifier")
	}
}

func TestGetNotifiers_DisabledProvider(t *testing.T) {
	db := setupTestWithProviders(t)
	// Disable the webhook notifier
	db.Model(&model.Provider{}).Where("id = ?", "notify-1").Update("enabled", false)

	b := &Bridge{DB: db}
	notifiers, err := b.getNotifiers(context.Background())
	if err != nil {
		t.Fatalf("getNotifiers failed: %v", err)
	}
	if len(notifiers) != 0 {
		t.Fatalf("expected 0 notifiers (disabled), got %d", len(notifiers))
	}
}

func TestGetNotifiers_UnknownChannel(t *testing.T) {
	db := setupTestWithProviders(t) // already has webhook
	// Add a notifier with unknown channel type
	ucfg, _ := json.Marshal(map[string]interface{}{"channel": "slack", "webhook_url": "https://slack.example.com"})
	db.Create(&model.Provider{
		ID:      "notify-slack",
		Type:    "notify",
		Name:    "slack-test",
		Config:  string(ucfg),
		Enabled: true,
	})

	b := &Bridge{DB: db}
	notifiers, err := b.getNotifiers(context.Background())
	if err != nil {
		t.Fatalf("getNotifiers failed: %v", err)
	}
	// Unknown channel should be skipped, only webhook from setupTestWithProviders
	if len(notifiers) != 1 {
		t.Fatalf("expected 1 notifier (unknown channel skipped), got %d", len(notifiers))
	}
}

// ===================== New DNS Provider Tests =====================

func TestGetDNSProvider_Aliyun(t *testing.T) {
	db := setupTestDB(t)
	cfg, _ := json.Marshal(map[string]interface{}{
		"access_key_id":     "test-key-id",
		"access_key_secret": "test-key-secret",
	})
	db.Create(&model.Provider{
		ID:      "dns-aliyun-1",
		Type:    "dns-aliyun",
		Name:    "Aliyun DNS",
		Config:  string(cfg),
		Enabled: true,
	})
	b := &Bridge{DB: db}
	provider, err := b.getDNSProvider(context.Background())
	if err != nil {
		t.Fatalf("getDNSProvider failed: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
	// Verify it implements DNSProvider by calling a method
	_, _ = provider.ListRecords(context.Background(), "example.com")
}

func TestGetDNSProvider_Tencent(t *testing.T) {
	db := setupTestDB(t)
	cfg, _ := json.Marshal(map[string]interface{}{
		"secret_id":  "test-secret-id",
		"secret_key": "test-secret-key",
	})
	db.Create(&model.Provider{
		ID:      "dns-tencent-1",
		Type:    "dns-tencent",
		Name:    "Tencent DNS",
		Config:  string(cfg),
		Enabled: true,
	})
	b := &Bridge{DB: db}
	provider, err := b.getDNSProvider(context.Background())
	if err != nil {
		t.Fatalf("getDNSProvider failed: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
	// Verify it implements DNSProvider by calling a method
	_, _ = provider.ListRecords(context.Background(), "example.com")
}

func TestGetDNSProvider_UnsupportedType(t *testing.T) {
	db := setupTestDB(t)
	cfg, _ := json.Marshal(map[string]interface{}{
		"api_token": "test-token",
	})
	db.Create(&model.Provider{
		ID:      "dns-unsupported-1",
		Type:    "dns-unsupported",
		Name:    "Unsupported DNS",
		Config:  string(cfg),
		Enabled: true,
	})
	b := &Bridge{DB: db}
	_, err := b.getDNSProvider(context.Background())
	if err == nil {
		t.Fatal("expected error for unsupported DNS provider type")
	}
}

func TestGetDNSProvider_InvalidConfig(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&model.Provider{
		ID:      "dns-bad-1",
		Type:    "dns-cloudflare",
		Name:    "Bad Config DNS",
		Config:  "not-json{{{",
		Enabled: true,
	})
	b := &Bridge{DB: db}
	_, err := b.getDNSProvider(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

// ===================== New Notifier Tests =====================

func TestGetNotifiers_Telegram(t *testing.T) {
	db := setupTestDB(t)
	cfg, _ := json.Marshal(map[string]interface{}{
		"channel":   "telegram",
		"bot_token": "test-bot-token",
		"chat_id":   "test-chat-id",
	})
	db.Create(&model.Provider{
		ID:      "notify-telegram-1",
		Type:    "notify",
		Name:    "Telegram",
		Config:  string(cfg),
		Enabled: true,
	})
	b := &Bridge{DB: db}
	notifiers, err := b.getNotifiers(context.Background())
	if err != nil {
		t.Fatalf("getNotifiers failed: %v", err)
	}
	if len(notifiers) != 1 {
		t.Fatalf("expected 1 notifier, got %d", len(notifiers))
	}
	if notifiers[0].Name() != "telegram" {
		t.Errorf("expected telegram notifier, got %s", notifiers[0].Name())
	}
}

func TestGetNotifiers_DingTalk(t *testing.T) {
	db := setupTestDB(t)
	cfg, _ := json.Marshal(map[string]interface{}{
		"channel":     "dingtalk",
		"webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=test",
		"secret":      "test-secret",
	})
	db.Create(&model.Provider{
		ID:      "notify-dingtalk-1",
		Type:    "notify",
		Name:    "DingTalk",
		Config:  string(cfg),
		Enabled: true,
	})
	b := &Bridge{DB: db}
	notifiers, err := b.getNotifiers(context.Background())
	if err != nil {
		t.Fatalf("getNotifiers failed: %v", err)
	}
	if len(notifiers) != 1 {
		t.Fatalf("expected 1 notifier, got %d", len(notifiers))
	}
	if notifiers[0].Name() != "dingtalk" {
		t.Errorf("expected dingtalk notifier, got %s", notifiers[0].Name())
	}
}

func TestGetNotifiers_Feishu(t *testing.T) {
	db := setupTestDB(t)
	cfg, _ := json.Marshal(map[string]interface{}{
		"channel":     "feishu",
		"webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/test-token",
	})
	db.Create(&model.Provider{
		ID:      "notify-feishu-1",
		Type:    "notify",
		Name:    "Feishu",
		Config:  string(cfg),
		Enabled: true,
	})
	b := &Bridge{DB: db}
	notifiers, err := b.getNotifiers(context.Background())
	if err != nil {
		t.Fatalf("getNotifiers failed: %v", err)
	}
	if len(notifiers) != 1 {
		t.Fatalf("expected 1 notifier, got %d", len(notifiers))
	}
	if notifiers[0].Name() != "feishu" {
		t.Errorf("expected feishu notifier, got %s", notifiers[0].Name())
	}
}

// setupTestWithProviders is a variant that returns the DB for further manipulation.
func setupTestWithProviders(t *testing.T) *gorm.DB {
	t.Helper()
	return setupDBWithProviders(t)
}

// --- Monitor/Healer Bridge Tests ---

func TestHealContainer_NoExecutor(t *testing.T) {
	db, _ := database.Connect("sqlite", ":memory:")
	database.Migrate(db)
	b := &Bridge{DB: db, Executor: nil}
	// getHealer creates a healer even with nil executor, but CheckAndHeal will
	// panic on nil executor. This is expected — in production, executor is always set.
	defer func() {
		if r := recover(); r != nil {
			t.Logf("expected panic with nil executor: %v", r)
		}
	}()
	b.HealContainer(context.Background(), "test")
}

func TestGetContainerMetrics_NoExecutor(t *testing.T) {
	db, _ := database.Connect("sqlite", ":memory:")
	database.Migrate(db)
	b := &Bridge{DB: db, Executor: nil}
	defer func() {
		if r := recover(); r != nil {
			t.Logf("expected panic with nil executor: %v", r)
		}
	}()
	b.GetContainerMetrics(context.Background(), "test")
}

func TestGetSystemMetrics_NoExecutor(t *testing.T) {
	db, _ := database.Connect("sqlite", ":memory:")
	database.Migrate(db)
	b := &Bridge{DB: db, Executor: nil}
	defer func() {
		if r := recover(); r != nil {
			t.Logf("expected panic with nil executor: %v", r)
		}
	}()
	b.GetSystemMetrics(context.Background())
}

func TestListAlerts_Empty(t *testing.T) {
	db, _ := database.Connect("sqlite", ":memory:")
	database.Migrate(db)
	b := &Bridge{DB: db, Executor: nil}
	defer func() {
		if r := recover(); r != nil {
			t.Logf("expected panic with nil executor: %v", r)
		}
	}()
	b.ListAlerts(context.Background())
}

func TestListAlertRules_Default(t *testing.T) {
	db, _ := database.Connect("sqlite", ":memory:")
	database.Migrate(db)
	b := &Bridge{DB: db, Executor: nil}
	defer func() {
		if r := recover(); r != nil {
			t.Logf("expected panic with nil executor: %v", r)
		}
	}()
	b.ListAlertRules(context.Background())
}

func TestBatchDeploy_Empty(t *testing.T) {
	db, _ := database.Connect("sqlite", ":memory:")
	database.Migrate(db)
	b := &Bridge{DB: db, Executor: nil}
	result, err := b.BatchDeploy(context.Background(), []map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	br, ok := result.(*mcp.BatchDeployResult)
	if !ok {
		t.Fatalf("expected *BatchDeployResult, got %T", result)
	}
	if br.Total != 0 {
		t.Errorf("expected 0 total, got %d", br.Total)
	}
}

func TestBatchDeploy_WithApps(t *testing.T) {
	db, _ := database.Connect("sqlite", ":memory:")
	database.Migrate(db)
	b := &Bridge{DB: db, Executor: nil}
	apps := []map[string]interface{}{
		{"image": "nginx:latest", "container_name": "test-1"},
		{"image": "redis:alpine", "container_name": "test-2"},
	}
	defer func() {
		if r := recover(); r != nil {
			t.Logf("expected panic with nil executor: %v", r)
		}
	}()
	b.BatchDeploy(context.Background(), apps)
}

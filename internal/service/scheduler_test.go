package service

import (
	"context"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/database"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

func setupSchedulerTest(t *testing.T) (*Scheduler, *Bridge, *mockExecutor) {
	t.Helper()
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}

	exec := &mockExecutor{
		output: map[string]string{
			"docker version --format '{{.Server.Version}}' 2>/dev/null": "24.0",
			"echo ok": "ok",
		},
		err: map[string]error{},
	}

	bridge := NewBridge(db, exec, []byte("01234567890123456789012345678901"), nil)
	scheduler := NewScheduler(db, bridge)

	t.Cleanup(func() {
		scheduler.Stop()
	})

	return scheduler, bridge, exec
}

func boolPtr(b bool) *bool {
	return &b
}

func clearScheduledTasks(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("DELETE FROM scheduled_tasks").Error; err != nil {
		t.Fatalf("failed to clear scheduled tasks: %v", err)
	}
}

func TestNewScheduler(t *testing.T) {
	scheduler, _, _ := setupSchedulerTest(t)
	if scheduler == nil {
		t.Fatal("NewScheduler returned nil")
	}
	if scheduler.entryMap == nil {
		t.Fatal("expected non-nil entryMap")
	}
}

func TestScheduler_AddTask(t *testing.T) {
	scheduler, _, _ := setupSchedulerTest(t)
	defer scheduler.Stop()

	enabled := true
	task := model.ScheduledTask{
		ID:       "task-1",
		TenantID: "tenant-default",
		Name:     "Test Task",
		CronExpr: "* * * * * *",
		TaskType: "shell",
		Command:  "echo hello",
		Enabled:  &enabled,
	}

	err := scheduler.AddTask(context.Background(), task)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	scheduler.Start(context.Background())
	defer scheduler.Stop()

	scheduler.mu.RLock()
	_, exists := scheduler.entryMap["task-1"]
	scheduler.mu.RUnlock()
	if !exists {
		t.Error("expected task to be added to entryMap")
	}
}

func TestScheduler_AddTask_Disabled(t *testing.T) {
	scheduler, bridge, _ := setupSchedulerTest(t)

	var countBefore int64
	bridge.DB.Model(&model.ScheduledTask{}).Count(&countBefore)
	t.Logf("Tasks before clear: %d", countBefore)

	clearScheduledTasks(t, bridge.DB)

	var countAfter int64
	bridge.DB.Model(&model.ScheduledTask{}).Count(&countAfter)
	t.Logf("Tasks after clear: %d", countAfter)

	task := model.ScheduledTask{
		ID:       "task-disabled",
		TenantID: "tenant-default",
		Name:     "Disabled Task",
		CronExpr: "* * * * * *",
		TaskType: "shell",
		Command:  "echo hello",
		Enabled:  boolPtr(false),
	}

	err := scheduler.AddTask(context.Background(), task)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	var savedTask model.ScheduledTask
	bridge.DB.First(&savedTask, "id = ?", "task-disabled")
	t.Logf("Saved task enabled status: %v", savedTask.Enabled)

	scheduler.Start(context.Background())
	defer scheduler.Stop()

	scheduler.mu.RLock()
	_, exists := scheduler.entryMap["task-disabled"]
	scheduler.mu.RUnlock()
	if exists {
		t.Error("expected disabled task not to be in entryMap")
	}
}

func TestScheduler_RemoveTask(t *testing.T) {
	scheduler, _, _ := setupSchedulerTest(t)
	scheduler.Start(context.Background())
	defer scheduler.Stop()

	task := model.ScheduledTask{
		ID:       "task-remove",
		TenantID: "tenant-default",
		Name:     "Remove Task",
		CronExpr: "* * * * * *",
		TaskType: "shell",
		Command:  "echo hello",
		Enabled:  boolPtr(true),
	}

	err := scheduler.AddTask(context.Background(), task)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	scheduler.mu.RLock()
	_, exists := scheduler.entryMap["task-remove"]
	scheduler.mu.RUnlock()
	if !exists {
		t.Fatal("expected task to exist before removal")
	}

	err = scheduler.RemoveTask("task-remove")
	if err != nil {
		t.Fatalf("RemoveTask failed: %v", err)
	}

	scheduler.mu.RLock()
	_, exists = scheduler.entryMap["task-remove"]
	scheduler.mu.RUnlock()
	if exists {
		t.Error("expected task to be removed from entryMap")
	}
}

func TestScheduler_RemoveTask_NotFound(t *testing.T) {
	scheduler, _, _ := setupSchedulerTest(t)

	err := scheduler.RemoveTask("nonexistent")
	if err != nil {
		t.Fatalf("RemoveTask should not error for nonexistent task: %v", err)
	}
}

func TestScheduler_ToggleTask(t *testing.T) {
	scheduler, _, _ := setupSchedulerTest(t)
	scheduler.Start(context.Background())
	defer scheduler.Stop()

	task := model.ScheduledTask{
		ID:       "task-toggle",
		TenantID: "tenant-default",
		Name:     "Toggle Task",
		CronExpr: "* * * * * *",
		TaskType: "shell",
		Command:  "echo hello",
		Enabled:  boolPtr(true),
	}

	err := scheduler.AddTask(context.Background(), task)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	err = scheduler.ToggleTask(context.Background(), "task-toggle", false)
	if err != nil {
		t.Fatalf("ToggleTask failed: %v", err)
	}

	scheduler.mu.RLock()
	_, exists := scheduler.entryMap["task-toggle"]
	scheduler.mu.RUnlock()
	if exists {
		t.Error("expected task to be removed after disabling")
	}

	err = scheduler.ToggleTask(context.Background(), "task-toggle", true)
	if err != nil {
		t.Fatalf("ToggleTask failed: %v", err)
	}

	scheduler.mu.RLock()
	_, exists = scheduler.entryMap["task-toggle"]
	scheduler.mu.RUnlock()
	if !exists {
		t.Error("expected task to be added after enabling")
	}
}

func TestScheduler_ListTasks(t *testing.T) {
	scheduler, _, _ := setupSchedulerTest(t)

	task1 := model.ScheduledTask{
		ID:       "task-list-1",
		TenantID: "tenant-default",
		Name:     "Task 1",
		CronExpr: "* * * * * *",
		TaskType: "shell",
		Command:  "echo 1",
	}
	task2 := model.ScheduledTask{
		ID:       "task-list-2",
		TenantID: "tenant-default",
		Name:     "Task 2",
		CronExpr: "* * * * * *",
		TaskType: "shell",
		Command:  "echo 2",
	}

	scheduler.AddTask(context.Background(), task1)
	scheduler.AddTask(context.Background(), task2)

	tasks, err := scheduler.ListTasks()
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestScheduler_ListTasks_Empty(t *testing.T) {
	scheduler, _, _ := setupSchedulerTest(t)

	tasks, err := scheduler.ListTasks()
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestScheduler_GetTaskExecutions(t *testing.T) {
	scheduler, bridge, _ := setupSchedulerTest(t)

	task := model.ScheduledTask{
		ID:       "task-exec",
		TenantID: "tenant-default",
		Name:     "Exec Task",
		CronExpr: "* * * * * *",
		TaskType: "shell",
		Command:  "echo test",
		Enabled:  boolPtr(false),
	}

	err := scheduler.AddTask(context.Background(), task)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Manually create an execution record
	bridge.DB.Create(&model.TaskExecution{
		ID:        "exec-1",
		TaskID:    "task-exec",
		TenantID:  "tenant-default",
		Status:    "success",
		Output:    "test",
		StartedAt: time.Now(),
		EndedAt:   time.Now(),
		Duration:  100,
	})

	executions, err := scheduler.GetTaskExecutions("task-exec", 10)
	if err != nil {
		t.Fatalf("GetTaskExecutions failed: %v", err)
	}
	if len(executions) != 1 {
		t.Errorf("expected 1 execution, got %d", len(executions))
	}
}

func TestScheduler_GetTaskExecutions_Limit(t *testing.T) {
	scheduler, bridge, _ := setupSchedulerTest(t)

	task := model.ScheduledTask{
		ID:       "task-limit",
		TenantID: "tenant-default",
		Name:     "Limit Task",
		CronExpr: "* * * * * *",
		TaskType: "shell",
		Command:  "echo test",
		Enabled:  boolPtr(false),
	}

	scheduler.AddTask(context.Background(), task)

	for i := 0; i < 5; i++ {
		bridge.DB.Create(&model.TaskExecution{
			ID:        "exec-limit-" + string(rune('0'+i)),
			TaskID:    "task-limit",
			TenantID:  "tenant-default",
			Status:    "success",
			StartedAt: time.Now().Add(time.Duration(i) * time.Minute),
			EndedAt:   time.Now().Add(time.Duration(i) * time.Minute),
			Duration:  100,
		})
	}

	executions, err := scheduler.GetTaskExecutions("task-limit", 2)
	if err != nil {
		t.Fatalf("GetTaskExecutions failed: %v", err)
	}
	if len(executions) != 2 {
		t.Errorf("expected 2 executions (limited), got %d", len(executions))
	}
}

func TestScheduler_GetTaskExecutions_NoLimit(t *testing.T) {
	scheduler, bridge, _ := setupSchedulerTest(t)

	task := model.ScheduledTask{
		ID:       "task-nolimit",
		TenantID: "tenant-default",
		Name:     "NoLimit Task",
		CronExpr: "* * * * * *",
		TaskType: "shell",
		Command:  "echo test",
		Enabled:  boolPtr(false),
	}

	scheduler.AddTask(context.Background(), task)

	for i := 0; i < 3; i++ {
		bridge.DB.Create(&model.TaskExecution{
			ID:        "exec-nolimit-" + string(rune('0'+i)),
			TaskID:    "task-nolimit",
			TenantID:  "tenant-default",
			Status:    "success",
			StartedAt: time.Now().Add(time.Duration(i) * time.Minute),
			EndedAt:   time.Now().Add(time.Duration(i) * time.Minute),
			Duration:  100,
		})
	}

	executions, err := scheduler.GetTaskExecutions("task-nolimit", 0)
	if err != nil {
		t.Fatalf("GetTaskExecutions failed: %v", err)
	}
	if len(executions) != 3 {
		t.Errorf("expected 3 executions (no limit), got %d", len(executions))
	}
}

func TestScheduler_Start_WithTasks(t *testing.T) {
	scheduler, bridge, _ := setupSchedulerTest(t)
	defer scheduler.Stop()

	// Create a task directly in DB
	task := model.ScheduledTask{
		ID:       "task-start",
		TenantID: "tenant-default",
		Name:     "Start Task",
		CronExpr: "@every 1h",
		TaskType: "shell",
		Command:  "echo start",
		Enabled:  boolPtr(true),
	}
	bridge.DB.Create(&task)

	scheduler.Start(context.Background())

	scheduler.mu.RLock()
	_, exists := scheduler.entryMap["task-start"]
	scheduler.mu.RUnlock()
	if !exists {
		t.Error("expected task loaded from DB to be in entryMap")
	}
}

func TestScheduler_Start_DBError(t *testing.T) {
	db, _ := database.Connect("sqlite", ":memory:")
	exec := &mockExecutor{}
	bridge := NewBridge(db, exec, []byte("01234567890123456789012345678901"), nil)
	
	scheduler := NewScheduler(nil, bridge)
	
	scheduler.Start(context.Background())
	defer scheduler.Stop()
}

func TestScheduler_ToggleTask_NotFound(t *testing.T) {
	scheduler, _, _ := setupSchedulerTest(t)

	err := scheduler.ToggleTask(context.Background(), "nonexistent", true)
	if err == nil {
		t.Error("expected error when toggling nonexistent task")
	}
}

func TestScheduler_AddTask_DBError(t *testing.T) {
	scheduler := NewScheduler(nil, nil)

	task := model.ScheduledTask{
		ID:       "task-db-error",
		TenantID: "tenant-default",
		Name:     "DB Error Task",
		CronExpr: "* * * * * *",
		TaskType: "shell",
		Command:  "echo hello",
	}

	err := scheduler.AddTask(context.Background(), task)
	if err == nil {
		t.Error("expected error when DB is nil")
	}
}

func TestScheduler_ExecuteTask_Shell(t *testing.T) {
	scheduler, bridge, exec := setupSchedulerTest(t)
	exec.output["echo test-command"] = "test-output"

	task := model.ScheduledTask{
		ID:       "task-exec-shell",
		TenantID: "tenant-default",
		Name:     "Shell Task",
		CronExpr: "* * * * * *",
		TaskType: "shell",
		Command:  "echo test-command",
		Enabled:  boolPtr(false),
	}

	scheduler.AddTask(context.Background(), task)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	scheduler.executeTask(ctx, task)

	var execution model.TaskExecution
	bridge.DB.Where("task_id = ?", "task-exec-shell").First(&execution)
	if execution.Status != "success" {
		t.Errorf("expected success status, got %s", execution.Status)
	}
	if execution.Output != "test-output" {
		t.Errorf("expected 'test-output', got %q", execution.Output)
	}
}

func TestScheduler_ExecuteTask_ShellError(t *testing.T) {
	scheduler, bridge, exec := setupSchedulerTest(t)
	exec.err["echo fail"] = gorm.ErrInvalidData

	task := model.ScheduledTask{
		ID:       "task-exec-fail",
		TenantID: "tenant-default",
		Name:     "Fail Task",
		CronExpr: "* * * * * *",
		TaskType: "shell",
		Command:  "echo fail",
		Enabled:  boolPtr(false),
	}

	scheduler.AddTask(context.Background(), task)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	scheduler.executeTask(ctx, task)

	var execution model.TaskExecution
	bridge.DB.Where("task_id = ?", "task-exec-fail").First(&execution)
	if execution.Status != "failed" {
		t.Errorf("expected failed status, got %s", execution.Status)
	}
}

func TestScheduler_ExecuteTask_UnknownType(t *testing.T) {
	scheduler, bridge, _ := setupSchedulerTest(t)

	task := model.ScheduledTask{
		ID:       "task-exec-unknown",
		TenantID: "tenant-default",
		Name:     "Unknown Task",
		CronExpr: "* * * * * *",
		TaskType: "unknown",
		Command:  "echo hello",
		Enabled:  boolPtr(false),
	}

	scheduler.AddTask(context.Background(), task)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	scheduler.executeTask(ctx, task)

	var execution model.TaskExecution
	bridge.DB.Where("task_id = ?", "task-exec-unknown").First(&execution)
	if execution.Status != "failed" {
		t.Errorf("expected failed status for unknown task type, got %s", execution.Status)
	}
}

func TestScheduler_ExecuteHealthCheck(t *testing.T) {
	scheduler, bridge, exec := setupSchedulerTest(t)
	exec.output[`docker inspect --format='{{.State.Health.Status}}' test-container 2>/dev/null || docker inspect --format='{{.State.Status}}' test-container`] = "healthy"

	task := model.ScheduledTask{
		ID:       "task-health",
		TenantID: "tenant-default",
		Name:     "Health Task",
		CronExpr: "* * * * * *",
		TaskType: "health_check",
		Command:  "test-container",
		Enabled:  boolPtr(false),
	}

	scheduler.AddTask(context.Background(), task)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	scheduler.executeTask(ctx, task)

	var execution model.TaskExecution
	bridge.DB.Where("task_id = ?", "task-health").First(&execution)
	if execution.Status != "success" {
		t.Errorf("expected success status, got %s", execution.Status)
	}
}

func TestScheduler_ExecuteHealthCheck_NoContainer(t *testing.T) {
	scheduler, bridge, _ := setupSchedulerTest(t)

	task := model.ScheduledTask{
		ID:       "task-health-empty",
		TenantID: "tenant-default",
		Name:     "Health Empty Task",
		CronExpr: "* * * * * *",
		TaskType: "health_check",
		Command:  "",
		Enabled:  boolPtr(false),
	}

	scheduler.AddTask(context.Background(), task)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	scheduler.executeTask(ctx, task)

	var execution model.TaskExecution
	bridge.DB.Where("task_id = ?", "task-health-empty").First(&execution)
	if execution.Status != "failed" {
		t.Errorf("expected failed status for empty container name, got %s", execution.Status)
	}
}

func TestScheduler_ExecuteLogCleanup(t *testing.T) {
	scheduler, _, _ := setupSchedulerTest(t)

	task := model.ScheduledTask{
		ID:       "task-cleanup",
		TenantID: "tenant-default",
		Name:     "Cleanup Task",
		CronExpr: "* * * * * *",
		TaskType: "log_cleanup",
		Command:  "",
		Enabled:  boolPtr(false),
	}

	scheduler.AddTask(context.Background(), task)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	scheduler.executeTask(ctx, task)

	var execution model.TaskExecution
	scheduler.bridge.DB.Where("task_id = ?", "task-cleanup").First(&execution)
	if execution.Status != "success" {
		t.Errorf("expected success status, got %s", execution.Status)
	}
}

func TestScheduler_AddTask_InvalidCron(t *testing.T) {
	scheduler, _, _ := setupSchedulerTest(t)
	scheduler.Start(context.Background())
	defer scheduler.Stop()

	task := model.ScheduledTask{
		ID:       "task-invalid-cron",
		TenantID: "tenant-default",
		Name:     "Invalid Cron",
		CronExpr: "invalid-cron-expr",
		TaskType: "shell",
		Command:  "echo hello",
		Enabled:  boolPtr(true),
	}

	err := scheduler.AddTask(context.Background(), task)
	if err != nil {
		t.Fatalf("AddTask should not fail for invalid cron: %v", err)
	}

	scheduler.mu.RLock()
	_, exists := scheduler.entryMap["task-invalid-cron"]
	scheduler.mu.RUnlock()
	if exists {
		t.Error("expected invalid cron task not to be scheduled")
	}
}
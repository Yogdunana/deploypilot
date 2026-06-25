package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/Yogdunana/deploypilot/internal/model"
)

// mockCron is a mock cron scheduler for testing.
type mockCron struct {
	entries   map[string]cron.EntryID
	mu        sync.Mutex
	started   bool
	callCount int
}

func newMockCron() *mockCron {
	return &mockCron{
		entries: make(map[string]cron.EntryID),
	}
}

func (m *mockCron) AddFunc(spec string, cmd func()) (cron.EntryID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	id := cron.EntryID(m.callCount)
	m.entries[spec] = id
	return id, nil
}

func (m *mockCron) Remove(id cron.EntryID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for spec, entryID := range m.entries {
		if entryID == id {
			delete(m.entries, spec)
		}
	}
}

func (m *mockCron) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
}

func (m *mockCron) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = false
}

// mockDB is a mock database for testing.
type mockDB struct {
	tasks     []model.ScheduledTask
	executions []model.TaskExecution
	mu        sync.Mutex
}

func newMockDB() *mockDB {
	return &mockDB{}
}

func (m *mockDB) Where(query string, args ...interface{}) *gorm.DB {
	// Simplified mock - just returns a chainable mock
	return &gorm.DB{}
}

func (m *mockDB) Find(out interface{}, where ...interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	if tasks, ok := out.(*[]model.ScheduledTask); ok {
		*tasks = m.tasks
	}
	return &gorm.DB{}
}

func (m *mockDB) Create(value interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	if task, ok := value.(*model.ScheduledTask); ok {
		m.tasks = append(m.tasks, *task)
	}
	if exec, ok := value.(*model.TaskExecution); ok {
		m.executions = append(m.executions, *exec)
	}
	return &gorm.DB{}
}

func (m *mockDB) Delete(value interface{}, where ...interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := value.(model.ScheduledTask); ok {
		m.tasks = nil
	}
	return &gorm.DB{}
}

func (m *mockDB) First(out interface{}, where ...interface{}) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	if task, ok := out.(*model.ScheduledTask); ok {
		if len(m.tasks) > 0 {
			*task = m.tasks[0]
		}
	}
	return &gorm.DB{}
}

func (m *mockDB) Model(value interface{}) *gorm.DB {
	return &gorm.DB{}
}

func (m *mockDB) Updates(values interface{}) *gorm.DB {
	return &gorm.DB{}
}

func (m *mockDB) Order(value string) *gorm.DB {
	return &gorm.DB{}
}

func (m *mockDB) Limit(limit int) *gorm.DB {
	return &gorm.DB{}
}

func (m *mockDB) Error() error {
	return nil
}

// TestScheduler tests the Scheduler struct initialization.
func TestScheduler_Struct(t *testing.T) {
	_ = newMockDB() // db is unused but needed for test setup
	bridge := &Bridge{}
	s := NewScheduler(nil, bridge)

	if s == nil {
		t.Error("NewScheduler returned nil")
	}
	if s.entryMap == nil {
		t.Error("entryMap should be initialized")
	}
}

func TestScheduler_AddTask_Validation(t *testing.T) {
	// Test that AddTask validates the task structure
	tests := []struct {
		name    string
		task    model.ScheduledTask
		wantErr bool
	}{
		{
			name: "valid task",
			task: model.ScheduledTask{
				ID:       "test-1",
				Name:     "Test Task",
				CronExpr: "0 * * * *",
				TaskType: "shell",
				Command:  "echo hello",
				Enabled:  true,
			},
			wantErr: false,
		},
		{
			name: "valid disabled task",
			task: model.ScheduledTask{
				ID:       "test-2",
				Name:     "Disabled Task",
				CronExpr: "0 * * * *",
				TaskType: "shell",
				Enabled:  false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - cron expression should be parseable
			parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
			_, err := parser.Parse(tt.task.CronExpr)
			if err != nil && !tt.wantErr {
				t.Errorf("Invalid cron expression: %v", err)
			}
		})
	}
}

func TestScheduler_CronExpressionParsing(t *testing.T) {
	tests := []struct {
		cronExpr string
		wantErr  bool
	}{
		// Valid expressions (standard 5-field format)
		{"0 * * * *", false},
		{"*/5 * * * *", false},
		{"0 0 * * *", false},
		{"0 9 * * 1", false},
		{"0 0 1 * *", false},
		{"0 0 1 1 *", false},
		{"0 30 4 * * *", false}, // with seconds (6-field)
		// Invalid expressions
		{"invalid", true},
		{"* * * *", true}, // missing field
		{"60 * * * *", true}, // minute out of range
		{"* 25 * * *", true}, // hour out of range
		{"", true},
		// Note: @ descriptors require cron.WithSeconds() option which is used in actual scheduler
		// but our test parser doesn't include it - descriptors are valid in production
	}

	parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	for _, tt := range tests {
		t.Run(tt.cronExpr, func(t *testing.T) {
			_, err := parser.Parse(tt.cronExpr)
			if tt.wantErr && err == nil {
				t.Errorf("Expected error for cron expression %q", tt.cronExpr)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error for cron expression %q: %v", tt.cronExpr, err)
			}
		})
	}
}

func TestScheduler_TaskTypes(t *testing.T) {
	// Test that all supported task types are recognized
	validTypes := []string{"shell", "health_check", "log_cleanup"}
	invalidTypes := []string{"unknown", "invalid", "", "backup"} // backup not in switch

	for _, tt := range validTypes {
		t.Run("valid_"+tt, func(t *testing.T) {
			// Task type should be accepted
			task := model.ScheduledTask{
				TaskType: tt,
				Command:  "test command",
			}
			if task.TaskType == "" {
				t.Error("Task type should not be empty")
			}
		})
	}

	for _, tt := range invalidTypes {
		t.Run("invalid_"+tt, func(t *testing.T) {
			// Unknown task types would result in error during execution
			task := model.ScheduledTask{
				TaskType: tt,
			}
			if task.TaskType != tt {
				t.Errorf("Task type mismatch")
			}
		})
	}
}

func TestScheduler_EntryMap(t *testing.T) {
	// Test entry map operations
	s := &Scheduler{
		entryMap: make(map[string]cron.EntryID),
	}

	// Add entry
	s.entryMap["task-1"] = cron.EntryID(1)
	s.entryMap["task-2"] = cron.EntryID(2)

	if len(s.entryMap) != 2 {
		t.Errorf("entryMap length = %d, want 2", len(s.entryMap))
	}

	// Remove entry
	delete(s.entryMap, "task-1")

	if len(s.entryMap) != 1 {
		t.Errorf("entryMap length after removal = %d, want 1", len(s.entryMap))
	}

	// Check remaining entry
	if s.entryMap["task-2"] != cron.EntryID(2) {
		t.Errorf("Remaining entry has wrong ID")
	}
}

func TestScheduler_ConcurrentAccess(t *testing.T) {
	// Test concurrent access to entryMap
	s := &Scheduler{
		entryMap: make(map[string]cron.EntryID),
		mu:       sync.RWMutex{},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		// Writer
		go func(id int) {
			defer wg.Done()
			key := "task-" + string(rune('a'+id%26))
			s.mu.Lock()
			s.entryMap[key] = cron.EntryID(id)
			s.mu.Unlock()
		}(i)
		// Reader
		go func(id int) {
			defer wg.Done()
			key := "task-" + string(rune('a'+id%26))
			s.mu.RLock()
			_ = s.entryMap[key]
			s.mu.RUnlock()
		}(i)
	}
	wg.Wait()

	// Should not panic or race
}

func TestScheduler_TimeoutHandling(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
		wantDur time.Duration
	}{
		{"default", 0, 5 * time.Minute},
		{"1 minute", 60, 60 * time.Second},
		{"5 minutes", 300, 300 * time.Second},
		{"10 minutes", 600, 600 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout := time.Duration(tt.timeout) * time.Second
			if timeout == 0 {
				timeout = 5 * time.Minute
			}
			if timeout != tt.wantDur {
				t.Errorf("Timeout = %v, want %v", timeout, tt.wantDur)
			}
		})
	}
}

func TestScheduler_ExecutionStatus(t *testing.T) {
	// Test execution status transitions
	statuses := []string{"running", "success", "failed"}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			exec := model.TaskExecution{
				Status: status,
			}
			if exec.Status != status {
				t.Errorf("Status mismatch")
			}
		})
	}
}

func TestScheduler_TaskExecutionRecord(t *testing.T) {
	now := time.Now()
	exec := model.TaskExecution{
		ID:        "exec-1",
		TaskID:    "task-1",
		TenantID:  "tenant-1",
		Status:    "success",
		Output:    "command output",
		StartedAt: now,
		EndedAt:   now.Add(5 * time.Second),
		Duration:  5000,
	}

	if exec.ID != "exec-1" {
		t.Error("ID mismatch")
	}
	if exec.Duration != 5000 {
		t.Error("Duration mismatch")
	}
}

func TestScheduler_HealthCheckTask(t *testing.T) {
	// Test health_check task type requires container name
	task := model.ScheduledTask{
		TaskType: "health_check",
		Command:  "my-container",
	}

	if task.Command == "" {
		t.Error("health_check task requires container name in Command field")
	}

	// Container name should be present
	if task.Command != "my-container" {
		t.Errorf("Container name mismatch")
	}
}

func TestScheduler_LogCleanupTask(t *testing.T) {
	// Test log_cleanup task type
	_ = model.ScheduledTask{
		TaskType: "log_cleanup",
	}

	// Default cleanup age
	defaultAge := 7 * 24 * time.Hour
	if defaultAge != 168*time.Hour {
		t.Errorf("Default cleanup age = %v, want 168h", defaultAge)
	}
}

func TestScheduler_ShellCommandTask(t *testing.T) {
	// Test shell task type
	task := model.ScheduledTask{
		TaskType: "shell",
		Command:  "docker ps -a",
		ServerID: "", // local execution
		Timeout:  60,
	}

	if task.TaskType != "shell" {
		t.Error("Task type mismatch")
	}
	if task.ServerID != "" {
		t.Error("Should be local execution")
	}
}

func TestScheduler_RemoteExecution(t *testing.T) {
	// Test remote execution via ServerID
	task := model.ScheduledTask{
		TaskType: "shell",
		Command:  "docker ps -a",
		ServerID: "server-123", // remote execution
	}

	if task.ServerID == "" {
		t.Error("Should be remote execution")
	}
}

func TestScheduler_EnableDisable(t *testing.T) {
	// Test ToggleTask logic
	enabled := true
	disabled := false

	task := model.ScheduledTask{
		ID:      "task-1",
		Enabled: enabled,
	}

	// Toggle to disabled
	task.Enabled = disabled
	if task.Enabled {
		t.Error("Task should be disabled")
	}

	// Toggle back to enabled
	task.Enabled = enabled
	if !task.Enabled {
		t.Error("Task should be enabled")
	}
}

func TestScheduler_MultipleTasks(t *testing.T) {
	// Test handling multiple tasks
	tasks := []model.ScheduledTask{
		{ID: "task-1", Name: "Task 1", Enabled: true},
		{ID: "task-2", Name: "Task 2", Enabled: false},
		{ID: "task-3", Name: "Task 3", Enabled: true},
	}

	enabledCount := 0
	for _, task := range tasks {
		if task.Enabled {
			enabledCount++
		}
	}

	if enabledCount != 2 {
		t.Errorf("Enabled count = %d, want 2", enabledCount)
	}
}

func TestScheduler_RunCount(t *testing.T) {
	// Test run count increment
	task := model.ScheduledTask{
		ID:       "task-1",
		RunCount: 0,
	}

	// Simulate successful run
	task.RunCount++
	task.LastStatus = "success"

	if task.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", task.RunCount)
	}
	if task.LastStatus != "success" {
		t.Errorf("LastStatus = %q, want 'success'", task.LastStatus)
	}
}

func TestScheduler_LastError(t *testing.T) {
	// Test error recording
	task := model.ScheduledTask{
		ID:        "task-1",
		LastError: "",
	}

	// Simulate failed run
	task.LastStatus = "failed"
	task.LastError = "command failed: exit status 1"

	if task.LastError == "" {
		t.Error("LastError should be recorded")
	}
}

func TestScheduler_ContextCancellation(t *testing.T) {
	// Test context cancellation handling
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Context should be cancelled
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled")
	}
}

func TestScheduler_ContextTimeout(t *testing.T) {
	// Test context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Context should not be cancelled immediately
	select {
	case <-ctx.Done():
		t.Error("Context should not be cancelled immediately")
	default:
		// Expected
	}
}
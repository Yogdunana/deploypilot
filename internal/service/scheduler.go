package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/Yogdunana/deploypilot/internal/sandbox"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/util/timeutil"
)

// Scheduler manages cron-based scheduled tasks.
type Scheduler struct {
	db       *gorm.DB
	cron     *cron.Cron
	bridge   *Bridge
	entryMap map[string]cron.EntryID // taskID -> entryID
	mu       sync.RWMutex
}

// NewScheduler creates a new Scheduler.
func NewScheduler(db *gorm.DB, bridge *Bridge) *Scheduler {
	return &Scheduler{
		db:       db,
		cron:     cron.New(cron.WithSeconds()),
		bridge:   bridge,
		entryMap: make(map[string]cron.EntryID),
	}
}

// Start loads enabled tasks from DB and starts the cron scheduler.
func (s *Scheduler) Start(ctx context.Context) {
	// Load tasks from DB
	var tasks []model.ScheduledTask
	if err := s.db.Where("enabled = ?", true).Find(&tasks).Error; err != nil {
		slog.Error("failed to load scheduled tasks", "error", err)
		return
	}

	for _, task := range tasks {
		s.addTask(ctx, task)
	}

	s.cron.Start()
	slog.Info("scheduler started", "tasks_loaded", len(tasks))
}

// Stop stops the cron scheduler.
func (s *Scheduler) Stop() {
	s.cron.Stop()
	slog.Info("scheduler stopped")
}

// AddTask registers a new scheduled task and starts it.
func (s *Scheduler) AddTask(ctx context.Context, task model.ScheduledTask) error {
	if err := s.db.Create(&task).Error; err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	if task.Enabled {
		s.addTask(ctx, task)
	}
	return nil
}

// RemoveTask removes a scheduled task.
func (s *Scheduler) RemoveTask(taskID string) error {
	s.mu.Lock()
	if entryID, ok := s.entryMap[taskID]; ok {
		s.cron.Remove(entryID)
		delete(s.entryMap, taskID)
	}
	s.mu.Unlock()

	return s.db.Delete(&model.ScheduledTask{}, "id = ?", taskID).Error
}

// ToggleTask enables or disables a task.
func (s *Scheduler) ToggleTask(ctx context.Context, taskID string, enabled bool) error {
	if err := s.db.Model(&model.ScheduledTask{}).Where("id = ?", taskID).Update("enabled", enabled).Error; err != nil {
		return err
	}

	s.mu.Lock()
	if !enabled {
		if entryID, ok := s.entryMap[taskID]; ok {
			s.cron.Remove(entryID)
			delete(s.entryMap, taskID)
		}
	} else {
		var task model.ScheduledTask
		if err := s.db.First(&task, "id = ?", taskID).Error; err == nil {
			s.addTask(ctx, task)
		}
	}
	s.mu.Unlock()
	return nil
}

// ListTasks returns all scheduled tasks.
func (s *Scheduler) ListTasks() ([]model.ScheduledTask, error) {
	var tasks []model.ScheduledTask
	err := s.db.Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

// GetTaskExecutions returns execution history for a task.
func (s *Scheduler) GetTaskExecutions(taskID string, limit int) ([]model.TaskExecution, error) {
	var executions []model.TaskExecution
	q := s.db.Where("task_id = ?", taskID).Order("started_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&executions).Error
	return executions, err
}

// addTask adds a task to the cron scheduler.
func (s *Scheduler) addTask(ctx context.Context, task model.ScheduledTask) {
	entryID, err := s.cron.AddFunc(task.CronExpr, func() {
		s.executeTask(ctx, task)
	})
	if err != nil {
		slog.Error("failed to schedule task", "task_id", task.ID, "cron", task.CronExpr, "error", err)
		return
	}

	s.mu.Lock()
	s.entryMap[task.ID] = entryID
	s.mu.Unlock()
}

// executeTask runs a scheduled task and records the result.
func (s *Scheduler) executeTask(ctx context.Context, task model.ScheduledTask) {
	startTime := timeutil.Now()
	execution := model.TaskExecution{
		ID:        fmt.Sprintf("%s-%d", task.ID, timeutil.UnixNano()),
		TaskID:    task.ID,
		TenantID:  task.TenantID,
		Status:    "running",
		StartedAt: startTime,
	}

	// Update task status
	s.db.Model(&model.ScheduledTask{}).Where("id = ?", task.ID).
		Updates(map[string]interface{}{"last_run_at": startTime, "last_status": "running"})

	// Execute based on task type
	var output string
	var execErr error

	switch task.TaskType {
	case "shell":
		output, execErr = s.executeShellCommand(ctx, task)
	case "health_check":
		output, execErr = s.executeHealthCheck(ctx, task)
	case "log_cleanup":
		output, execErr = s.executeLogCleanup(ctx, task)
	default:
		execErr = fmt.Errorf("unknown task type: %s", task.TaskType)
	}

	endTime := timeutil.Now()
	duration := endTime.Sub(startTime).Milliseconds()

	// Update execution record
	execution.EndedAt = endTime
	execution.Duration = duration
	if execErr != nil {
		execution.Status = "failed"
		execution.Error = execErr.Error()
		s.db.Model(&model.ScheduledTask{}).Where("id = ?", task.ID).
			Updates(map[string]interface{}{"last_status": "failed", "last_error": execErr.Error()})
	} else {
		execution.Status = "success"
		execution.Output = output
		s.db.Model(&model.ScheduledTask{}).Where("id = ?", task.ID).
			Updates(map[string]interface{}{"last_status": "success", "last_error": "", "run_count": gorm.Expr("run_count + 1")})
	}

	if err := s.db.Create(&execution).Error; err != nil {
		slog.Warn("failed to save task execution", "error", err)
	}
}

// executeShellCommand runs a shell command on local or remote server.
// Local commands are wrapped with SandboxExecutor to enforce sandbox rules.
func (s *Scheduler) executeShellCommand(ctx context.Context, task model.ScheduledTask) (string, error) {
	var executor CommandExecutor
	if task.ServerID != "" {
		remoteExec, err := s.bridge.getRemoteExecutor(ctx, task.ServerID)
		if err != nil {
			return "", fmt.Errorf("failed to get remote executor: %w", err)
		}
		defer func() {
			if cerr := remoteExec.Close(); cerr != nil {
				slog.Warn("failed to close remote executor", "error", cerr)
			}
		}()
		executor = remoteExec
	} else {
		// Wrap local executor with sandbox to enforce command validation
		if sb := s.bridge.GetSandbox(); sb != nil {
			sandboxInst, ok := sb.(interface{ GetConfig() sandbox.Config; Validate(cmd string) error })
			if ok {
				if err := sandboxInst.Validate(task.Command); err != nil {
					return "", fmt.Errorf("sandbox validation failed: %w", err)
				}
			}
		}
		executor = s.bridge.Executor
	}

	timeout := time.Duration(task.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return executor.RunCommand(ctx, task.Command)
}

// executeHealthCheck runs a container health check via docker inspect.
func (s *Scheduler) executeHealthCheck(ctx context.Context, task model.ScheduledTask) (string, error) {
	// Use docker inspect to check container health status
	// task.Command contains the container name
	containerName := task.Command
	if containerName == "" {
		return "", fmt.Errorf("container name is required for health_check task type")
	}

	checkCmd := fmt.Sprintf("docker inspect --format='{{.State.Health.Status}}' %s 2>/dev/null || docker inspect --format='{{.State.Status}}' %s", containerName, containerName)

	var executor CommandExecutor
	if task.ServerID != "" {
		remoteExec, err := s.bridge.getRemoteExecutor(ctx, task.ServerID)
		if err != nil {
			return "", fmt.Errorf("failed to get remote executor: %w", err)
		}
		defer func() {
			if cerr := remoteExec.Close(); cerr != nil {
				slog.Warn("failed to close remote executor", "error", cerr)
			}
		}()
		executor = remoteExec
	} else {
		executor = s.bridge.Executor
	}

	timeout := time.Duration(task.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	output, err := executor.RunCommand(ctx, checkCmd)
	if err != nil {
		return "", fmt.Errorf("health check failed for container %s: %w", containerName, err)
	}

	return fmt.Sprintf("container %s health status: %s", containerName, output), nil
}

// executeLogCleanup cleans up old metric records.
func (s *Scheduler) executeLogCleanup(ctx context.Context, task model.ScheduledTask) (string, error) {
	// Default: clean metrics older than 7 days
	olderThan := 7 * 24 * time.Hour
	mon := s.bridge.Monitor
	if mon != nil && mon.GetStore() != nil {
		if err := mon.GetStore().CleanupOldMetrics(ctx, olderThan); err != nil {
			return "", fmt.Errorf("failed to cleanup old metrics: %w", err)
		}
		return fmt.Sprintf("cleaned up metrics older than %v", olderThan), nil
	}
	return "no monitor store configured, skipping cleanup", nil
}

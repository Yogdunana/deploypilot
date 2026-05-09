package service

import (
	"context"
	"fmt"
)

// ========== SchedulerService interface (stubs) ==========

func (b *Bridge) CreateScheduledTask(ctx context.Context, name, cronExpr, taskType, command string, serverID string) (interface{}, error) {
	return nil, fmt.Errorf("scheduled tasks: use Scheduler directly")
}

func (b *Bridge) ListScheduledTasks(ctx context.Context) (interface{}, error) {
	return nil, fmt.Errorf("scheduled tasks: use Scheduler directly")
}

func (b *Bridge) GetTaskExecutions(ctx context.Context, taskID string, limit int) (interface{}, error) {
	return nil, fmt.Errorf("scheduled tasks: use Scheduler directly")
}

func (b *Bridge) ToggleScheduledTask(ctx context.Context, taskID string, enabled bool) (interface{}, error) {
	return nil, fmt.Errorf("scheduled tasks: use Scheduler directly")
}

func (b *Bridge) DeleteScheduledTask(ctx context.Context, taskID string) (interface{}, error) {
	return nil, fmt.Errorf("scheduled tasks: use Scheduler directly")
}

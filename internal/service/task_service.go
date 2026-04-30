package service

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// Simple in-memory task tracker.
type taskInfo struct {
	ID        string      `json:"task_id"`
	Type      string      `json:"type"`
	Status    string      `json:"status"`   // pending, running, success, failed
	Progress  int         `json:"progress"` // 0-100
	Message   string      `json:"message"`
	Result    interface{} `json:"result,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func (b *Bridge) createTask(taskType string) string {
	b.taskMu.Lock()
	defer b.taskMu.Unlock()
	b.taskCounter++
	id := fmt.Sprintf("task-%d", b.taskCounter)
	b.tasks[id] = &taskInfo{
		ID:        id,
		Type:      taskType,
		Status:    "pending",
		Progress:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return id
}

// updateTask updates the status of an existing task.
func (b *Bridge) updateTask(id, status string, progress int, message string) {
	b.taskMu.Lock()
	defer b.taskMu.Unlock()
	if t, ok := b.tasks[id]; ok {
		t.Status = status
		t.Progress = progress
		t.Message = message
		t.UpdatedAt = time.Now()
	}
}

// getTask returns a copy of the task info for the given task ID.
func (b *Bridge) getTask(id string) *taskInfo {
	b.taskMu.RLock()
	defer b.taskMu.RUnlock()
	if t, ok := b.tasks[id]; ok {
		cp := *t
		return &cp
	}
	return nil
}

// ---------- 27. GetTaskStatus ----------

func (b *Bridge) GetTaskStatus(ctx context.Context, taskID string) (interface{}, error) {
	b.taskMu.RLock()
	defer b.taskMu.RUnlock()
	t, ok := b.tasks[taskID]
	if !ok {
		return map[string]interface{}{
			"task_id": taskID,
			"status":  "not_found",
			"message": "task not found",
		}, nil
	}
	return t, nil
}

// ---------- 28. ListTasks ----------

func (b *Bridge) ListTasks(ctx context.Context, limit int, statusFilter string) (interface{}, error) {
	b.taskMu.RLock()
	defer b.taskMu.RUnlock()

	result := make([]*taskInfo, 0)
	for _, t := range b.tasks {
		if statusFilter != "" && t.Status != statusFilter {
			continue
		}
		result = append(result, t)
	}

	// Sort by created_at descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	if result == nil {
		result = []*taskInfo{}
	}

	return map[string]interface{}{
		"status": "success",
		"tasks":  result,
		"total":  len(result),
		"limit":  limit,
		"filter": statusFilter,
	}, nil
}

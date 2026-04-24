package service

import (
	"sync"
	"testing"
	"time"
)

func TestNewDeployEventBus(t *testing.T) {
	bus := NewDeployEventBus()
	if bus == nil {
		t.Fatal("NewDeployEventBus() returned nil")
	}
	if bus.subscribers == nil {
		t.Fatal("expected non-nil subscribers map")
	}
}

func TestDeployEventBus_SubscribeUnsubscribe(t *testing.T) {
	bus := NewDeployEventBus()

	ch1 := bus.Subscribe("app-1")
	if ch1 == nil {
		t.Fatal("Subscribe returned nil channel")
	}

	// Verify subscriber count
	bus.mu.RLock()
	count := len(bus.subscribers["app-1"])
	bus.mu.RUnlock()
	if count != 1 {
		t.Errorf("expected 1 subscriber, got %d", count)
	}

	// Subscribe a second channel
	ch2 := bus.Subscribe("app-1")
	bus.mu.RLock()
	count = len(bus.subscribers["app-1"])
	bus.mu.RUnlock()
	if count != 2 {
		t.Errorf("expected 2 subscribers, got %d", count)
	}

	// Unsubscribe first channel
	bus.Unsubscribe("app-1", ch1)
	bus.mu.RLock()
	count = len(bus.subscribers["app-1"])
	bus.mu.RUnlock()
	if count != 1 {
		t.Errorf("expected 1 subscriber after unsubscribe, got %d", count)
	}

	// Unsubscribe second channel - should clean up the appID key
	bus.Unsubscribe("app-1", ch2)
	bus.mu.RLock()
	_, exists := bus.subscribers["app-1"]
	bus.mu.RUnlock()
	if exists {
		t.Error("expected app-1 key to be removed after last subscriber unsubscribed")
	}

	// Verify channel was closed
	_, ok := <-ch1
	if ok {
		t.Error("expected ch1 to be closed after unsubscribe")
	}
}

func TestDeployEventBus_Publish(t *testing.T) {
	bus := NewDeployEventBus()
	ch := bus.Subscribe("app-pub")

	event := DeployEvent{
		TaskID:    "task-1",
		AppID:     "app-pub",
		Step:      "pull",
		Status:    "running",
		Progress:  30,
		Message:   "pulling image",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	bus.Publish(event)

	select {
	case received := <-ch:
		if received.TaskID != event.TaskID {
			t.Errorf("TaskID = %q, want %q", received.TaskID, event.TaskID)
		}
		if received.Step != event.Step {
			t.Errorf("Step = %q, want %q", received.Step, event.Step)
		}
		if received.Progress != event.Progress {
			t.Errorf("Progress = %d, want %d", received.Progress, event.Progress)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for published event")
	}
}

func TestDeployEventBus_UnsubscribeCleansUp(t *testing.T) {
	bus := NewDeployEventBus()

	ch := bus.Subscribe("app-cleanup")
	bus.Unsubscribe("app-cleanup", ch)

	// Publishing to an app with no subscribers should not panic
	event := DeployEvent{AppID: "app-cleanup", Step: "done", Status: "success", Progress: 100}
	bus.Publish(event)
	// If we get here without panic, the test passes
}

func TestDeployEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewDeployEventBus()

	ch1 := bus.Subscribe("app-multi")
	ch2 := bus.Subscribe("app-multi")
	ch3 := bus.Subscribe("app-multi")

	event := DeployEvent{
		TaskID: "task-multi",
		AppID:  "app-multi",
		Step:   "done",
		Status: "success",
	}

	bus.Publish(event)

	// All three channels should receive the event
	for i, ch := range []chan DeployEvent{ch1, ch2, ch3} {
		select {
		case received := <-ch:
			if received.TaskID != "task-multi" {
				t.Errorf("subscriber %d: TaskID = %q, want %q", i, received.TaskID, "task-multi")
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("subscriber %d: timed out waiting for event", i)
		}
	}
}

func TestDeployEventBus_PublishNoSubscribers(t *testing.T) {
	bus := NewDeployEventBus()

	// Publishing to an app with no subscribers should not panic
	event := DeployEvent{AppID: "nonexistent", Step: "done", Status: "success", Progress: 100}
	bus.Publish(event)
}

func TestDeployEventBus_ConcurrentPublish(t *testing.T) {
	bus := NewDeployEventBus()
	ch := bus.Subscribe("app-concurrent")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			bus.Publish(DeployEvent{
				TaskID:   "task-concurrent",
				AppID:    "app-concurrent",
				Step:     "pull",
				Status:   "running",
				Progress: n,
			})
		}(i)
	}
	wg.Wait()

	// Drain the channel and count received events
	received := 0
	timeout := time.After(2 * time.Second)
	for {
		select {
		case <-ch:
			received++
		case <-timeout:
			goto done
		}
	}
done:
	if received == 0 {
		t.Error("expected to receive at least some events from concurrent publish")
	}
}

func TestUpdateTask(t *testing.T) {
	// Reset tasks for test
	taskMu.Lock()
	oldTasks := tasks
	oldCounter := taskCounter
	tasks = make(map[string]*taskInfo)
	taskCounter = 0
	taskMu.Unlock()

	defer func() {
		taskMu.Lock()
		tasks = oldTasks
		taskCounter = oldCounter
		taskMu.Unlock()
	}()

	id := createTask("test")
	updateTask(id, "running", 50, "halfway done")

	task := getTask(id)
	if task == nil {
		t.Fatal("getTask returned nil")
	}
	if task.Status != "running" {
		t.Errorf("Status = %q, want %q", task.Status, "running")
	}
	if task.Progress != 50 {
		t.Errorf("Progress = %d, want %d", task.Progress, 50)
	}
	if task.Message != "halfway done" {
		t.Errorf("Message = %q, want %q", task.Message, "halfway done")
	}

	// Update non-existent task should not panic
	updateTask("nonexistent", "failed", 100, "error")
}

func TestGetTask(t *testing.T) {
	taskMu.Lock()
	oldTasks := tasks
	oldCounter := taskCounter
	tasks = make(map[string]*taskInfo)
	taskCounter = 0
	taskMu.Unlock()

	defer func() {
		taskMu.Lock()
		tasks = oldTasks
		taskCounter = oldCounter
		taskMu.Unlock()
	}()

	// Non-existent task
	task := getTask("nonexistent")
	if task != nil {
		t.Error("expected nil for non-existent task")
	}

	id := createTask("test")
	task = getTask(id)
	if task == nil {
		t.Fatal("expected non-nil task")
	}
	if task.ID != id {
		t.Errorf("ID = %q, want %q", task.ID, id)
	}
}

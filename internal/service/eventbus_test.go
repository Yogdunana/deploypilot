package service

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewInMemoryEventBus(t *testing.T) {
	bus := NewInMemoryEventBus()
	if bus == nil {
		t.Fatal("NewInMemoryEventBus() returned nil")
	}
	if bus.subscribers == nil {
		t.Fatal("expected non-nil subscribers map")
	}
}

func TestInMemoryEventBus_ImplementsInterface(t *testing.T) {
	var _ EventBus = NewInMemoryEventBus()
}

func TestInMemoryEventBus_Subscribe(t *testing.T) {
	bus := NewInMemoryEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.Subscribe(ctx, "app-1")
	if ch == nil {
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
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	_ = bus.Subscribe(ctx2, "app-1")
	bus.mu.RLock()
	count = len(bus.subscribers["app-1"])
	bus.mu.RUnlock()
	if count != 2 {
		t.Errorf("expected 2 subscribers, got %d", count)
	}

	// Cancel first subscription
	cancel()
	time.Sleep(50 * time.Millisecond)
	bus.mu.RLock()
	count = len(bus.subscribers["app-1"])
	bus.mu.RUnlock()
	if count != 1 {
		t.Errorf("expected 1 subscriber after cancel, got %d", count)
	}

	// Cancel second subscription - should clean up the appID key
	cancel2()
	time.Sleep(50 * time.Millisecond)
	bus.mu.RLock()
	_, exists := bus.subscribers["app-1"]
	bus.mu.RUnlock()
	if exists {
		t.Error("expected app-1 key to be removed after last subscriber cancelled")
	}

	// Verify channel was closed
	_, ok := <-ch
	if ok {
		t.Error("expected ch to be closed after cancel")
	}
}

func TestInMemoryEventBus_Publish(t *testing.T) {
	bus := NewInMemoryEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.Subscribe(ctx, "app-pub")

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

func TestInMemoryEventBus_UnsubscribeCleansUp(t *testing.T) {
	bus := NewInMemoryEventBus()
	ctx, cancel := context.WithCancel(context.Background())

	ch := bus.Subscribe(ctx, "app-cleanup")
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Publishing to an app with no subscribers should not panic
	event := DeployEvent{AppID: "app-cleanup", Step: "done", Status: "success", Progress: 100}
	bus.Publish(event)
	// If we get here without panic, the test passes
	// Also verify the channel was closed
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after context cancel")
	}
}

func TestInMemoryEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewInMemoryEventBus()

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	ctx3, cancel3 := context.WithCancel(context.Background())
	defer cancel3()

	ch1 := bus.Subscribe(ctx1, "app-multi")
	ch2 := bus.Subscribe(ctx2, "app-multi")
	ch3 := bus.Subscribe(ctx3, "app-multi")

	event := DeployEvent{
		TaskID: "task-multi",
		AppID:  "app-multi",
		Step:   "done",
		Status: "success",
	}

	bus.Publish(event)

	// All three channels should receive the event
	for i, ch := range []<-chan DeployEvent{ch1, ch2, ch3} {
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

func TestInMemoryEventBus_PublishNoSubscribers(t *testing.T) {
	bus := NewInMemoryEventBus()

	// Publishing to an app with no subscribers should not panic
	event := DeployEvent{AppID: "nonexistent", Step: "done", Status: "success", Progress: 100}
	bus.Publish(event)
}

func TestInMemoryEventBus_ConcurrentPublish(t *testing.T) {
	bus := NewInMemoryEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.Subscribe(ctx, "app-concurrent")

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

func TestInMemoryEventBus_Close(t *testing.T) {
	bus := NewInMemoryEventBus()
	err := bus.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestUpdateTask(t *testing.T) {
	bridge := NewBridge(nil, nil, nil, nil)

	// Reset tasks for test
	bridge.taskMu.Lock()
	oldTasks := bridge.tasks
	oldCounter := bridge.taskCounter
	bridge.tasks = make(map[string]*taskInfo)
	bridge.taskCounter = 0
	bridge.taskMu.Unlock()

	defer func() {
		bridge.taskMu.Lock()
		bridge.tasks = oldTasks
		bridge.taskCounter = oldCounter
		bridge.taskMu.Unlock()
	}()

	id := bridge.createTask("test")
	bridge.updateTask(id, "running", 50, "halfway done")

	task := bridge.getTask(id)
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
	bridge.updateTask("nonexistent", "failed", 100, "error")
}

func TestGetTask(t *testing.T) {
	bridge := NewBridge(nil, nil, nil, nil)

	bridge.taskMu.Lock()
	oldTasks := bridge.tasks
	oldCounter := bridge.taskCounter
	bridge.tasks = make(map[string]*taskInfo)
	bridge.taskCounter = 0
	bridge.taskMu.Unlock()

	defer func() {
		bridge.taskMu.Lock()
		bridge.tasks = oldTasks
		bridge.taskCounter = oldCounter
		bridge.taskMu.Unlock()
	}()

	// Non-existent task
	task := bridge.getTask("nonexistent")
	if task != nil {
		t.Error("expected nil for non-existent task")
	}

	id := bridge.createTask("test")
	task = bridge.getTask(id)
	if task == nil {
		t.Fatal("expected non-nil task")
	}
	if task.ID != id {
		t.Errorf("ID = %q, want %q", task.ID, id)
	}
}

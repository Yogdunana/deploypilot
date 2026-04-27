package confirm

import (
	"testing"
	"time"
)

func TestCreateAndGet(t *testing.T) {
	store := NewStore()
	req := store.Create("delete_app", "Delete app myapp", map[string]string{"app_id": "123"})

	if req.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if req.Status != StatusPending {
		t.Errorf("expected pending, got %s", req.Status)
	}
	if req.Tool != "delete_app" {
		t.Errorf("expected delete_app, got %s", req.Tool)
	}

	got, ok := store.Get(req.ID)
	if !ok {
		t.Fatal("expected to find request")
	}
	if got.ID != req.ID {
		t.Errorf("expected %s, got %s", req.ID, got.ID)
	}
}

func TestConfirm(t *testing.T) {
	store := NewStore()
	req := store.Create("delete_app", "Delete app", nil)

	confirmed, err := store.Confirm(req.ID, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if confirmed.Status != StatusApproved {
		t.Errorf("expected approved, got %s", confirmed.Status)
	}
	if confirmed.ConfirmedBy != "admin" {
		t.Errorf("expected admin, got %s", confirmed.ConfirmedBy)
	}
}

func TestReject(t *testing.T) {
	store := NewStore()
	req := store.Create("delete_app", "Delete app", nil)

	rejected, err := store.Reject(req.ID, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rejected.Status != StatusRejected {
		t.Errorf("expected rejected, got %s", rejected.Status)
	}
}

func TestConfirmNotFound(t *testing.T) {
	store := NewStore()
	_, err := store.Confirm("nonexistent", "admin")
	if err == nil {
		t.Error("expected error for nonexistent ID")
	}
}

func TestConfirmAlreadyApproved(t *testing.T) {
	store := NewStore()
	req := store.Create("delete_app", "Delete app", nil)
	store.Confirm(req.ID, "admin")

	_, err := store.Confirm(req.ID, "admin")
	if err == nil {
		t.Error("expected error for already approved confirmation")
	}
}

func TestList(t *testing.T) {
	store := NewStore()
	store.Create("delete_app", "Delete app 1", nil)
	store.Create("delete_app", "Delete app 2", nil)
	store.Create("deploy_app", "Deploy app", nil)

	all := store.List("")
	if len(all) != 3 {
		t.Errorf("expected 3, got %d", len(all))
	}

	pending := store.List(StatusPending)
	if len(pending) != 3 {
		t.Errorf("expected 3 pending, got %d", len(pending))
	}
}

func TestSetResult(t *testing.T) {
	store := NewStore()
	req := store.Create("delete_app", "Delete app", nil)
	store.Confirm(req.ID, "admin")

	store.SetResult(req.ID, "app deleted", "")
	got, _ := store.Get(req.ID)
	if got.Status != StatusExecuted {
		t.Errorf("expected executed, got %s", got.Status)
	}
	if got.Result != "app deleted" {
		t.Errorf("expected 'app deleted', got %s", got.Result)
	}
}

func TestSetResultError(t *testing.T) {
	store := NewStore()
	req := store.Create("delete_app", "Delete app", nil)
	store.Confirm(req.ID, "admin")

	store.SetResult(req.ID, "", "deployment failed")
	got, _ := store.Get(req.ID)
	if got.Error != "deployment failed" {
		t.Errorf("expected error message, got %s", got.Error)
	}
}

func TestCleanup(t *testing.T) {
	store := NewStore()
	store.defaultTTL = 100 * time.Millisecond

	req := store.Create("delete_app", "Delete app", nil)
	store.Confirm(req.ID, "admin")
	store.SetResult(req.ID, "done", "")

	time.Sleep(200 * time.Millisecond)

	count := store.Cleanup(time.Millisecond)
	if count != 1 {
		t.Errorf("expected 1 cleaned, got %d", count)
	}

	_, ok := store.Get(req.ID)
	if ok {
		t.Error("expected request to be cleaned up")
	}
}

func TestExpiration(t *testing.T) {
	store := NewStore()
	store.defaultTTL = 50 * time.Millisecond

	req := store.Create("delete_app", "Delete app", nil)
	time.Sleep(100 * time.Millisecond)

	_, err := store.Confirm(req.ID, "admin")
	if err == nil {
		t.Error("expected error for expired confirmation")
	}
}

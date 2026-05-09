package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := New(DefaultConfig())
	if cb.State() != Closed {
		t.Errorf("expected closed, got %s", cb.State())
	}
}

func TestExecuteSuccess(t *testing.T) {
	cb := New(DefaultConfig())
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cb.State() != Closed {
		t.Errorf("expected closed, got %s", cb.State())
	}
}

func TestExecuteFailure(t *testing.T) {
	cb := New(Config{FailureThreshold: 3, SuccessThreshold: 1, Timeout: 100 * time.Millisecond})
	for i := 0; i < 3; i++ {
		cb.Execute(func() error { return errors.New("fail") })
	}
	if cb.State() != Open {
		t.Errorf("expected open, got %s", cb.State())
	}
}

func TestCircuitOpen(t *testing.T) {
	cb := New(Config{FailureThreshold: 1, SuccessThreshold: 1, Timeout: 100 * time.Millisecond})
	cb.Execute(func() error { return errors.New("fail") })
	if cb.State() != Open {
		t.Fatalf("expected open, got %s", cb.State())
	}
	err := cb.Execute(func() error { return nil })
	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestHalfOpenToClosed(t *testing.T) {
	cb := New(Config{FailureThreshold: 1, SuccessThreshold: 2, Timeout: 50 * time.Millisecond})
	cb.Execute(func() error { return errors.New("fail") })
	if cb.State() != Open {
		t.Fatalf("expected open, got %s", cb.State())
	}
	time.Sleep(60 * time.Millisecond)
	if cb.State() != HalfOpen {
		t.Fatalf("expected half-open, got %s", cb.State())
	}
	cb.Execute(func() error { return nil })
	cb.Execute(func() error { return nil })
	if cb.State() != Closed {
		t.Errorf("expected closed, got %s", cb.State())
	}
}

func TestHalfOpenFailureReopens(t *testing.T) {
	cb := New(Config{FailureThreshold: 1, SuccessThreshold: 2, Timeout: 50 * time.Millisecond})
	cb.Execute(func() error { return errors.New("fail") })
	time.Sleep(60 * time.Millisecond)
	cb.Execute(func() error { return errors.New("fail") })
	if cb.State() != Open {
		t.Errorf("expected open after half-open failure, got %s", cb.State())
	}
}

func TestReset(t *testing.T) {
	cb := New(Config{FailureThreshold: 1, SuccessThreshold: 1, Timeout: 100 * time.Millisecond})
	cb.Execute(func() error { return errors.New("fail") })
	cb.Reset()
	if cb.State() != Closed {
		t.Errorf("expected closed after reset, got %s", cb.State())
	}
}

func TestOnStateChange(t *testing.T) {
	var transitions []string
	cb := New(Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          50 * time.Millisecond,
		OnStateChange: func(from, to State) {
			transitions = append(transitions, from.String()+"->"+to.String())
		},
	})
	cb.Execute(func() error { return errors.New("fail") })
	if len(transitions) != 1 || transitions[0] != "closed->open" {
		t.Errorf("expected closed->open, got %v", transitions)
	}
	time.Sleep(60 * time.Millisecond)
	cb.Execute(func() error { return nil })
	if len(transitions) != 2 || transitions[1] != "half-open->closed" {
		t.Errorf("expected half-open->closed, got %v", transitions)
	}
}

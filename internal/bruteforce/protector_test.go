package bruteforce

import (
	"testing"
	"time"
)

func TestCheckAllowsNormalLogin(t *testing.T) {
	p := New(DefaultConfig())
	result := p.Check("admin", "1.2.3.4")
	if !result.Allowed {
		t.Error("expected normal login to be allowed")
	}
	if result.Delay > 0 {
		t.Errorf("expected no delay, got %v", result.Delay)
	}
}

func TestProgressiveDelay(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProgressiveDelay = true
	cfg.BaseDelay = 100 * time.Millisecond
	cfg.MaxDelay = 2 * time.Second
	p := New(cfg)

	// Record failures
	for i := 0; i < 3; i++ {
		p.RecordFailure("user1", "1.2.3.4", "invalid password")
	}

	result := p.Check("user1", "1.2.3.4")
	if !result.Allowed {
		t.Error("expected login to be allowed (not locked yet)")
	}
	if result.Delay == 0 {
		t.Error("expected progressive delay after 3 failures")
	}
}

func TestAccountLockout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxAttempts = 3
	cfg.LockoutDuration = 1 * time.Minute
	p := New(cfg)

	for i := 0; i < 3; i++ {
		p.RecordFailure("user2", "1.2.3.4", "invalid password")
	}

	result := p.Check("user2", "1.2.3.4")
	if result.Allowed {
		t.Error("expected account to be locked after 3 failures")
	}
	if result.Reason != "account_locked" {
		t.Errorf("expected reason 'account_locked', got '%s'", result.Reason)
	}
}

func TestIPLockout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.IPMaxAttempts = 3
	cfg.IPLockoutDuration = 1 * time.Minute
	p := New(cfg)

	// Fail from same IP with different usernames
	p.RecordFailure("user_a", "10.0.0.1", "invalid")
	p.RecordFailure("user_b", "10.0.0.1", "invalid")
	p.RecordFailure("user_c", "10.0.0.1", "invalid")

	result := p.Check("user_d", "10.0.0.1")
	if result.Allowed {
		t.Error("expected IP to be blocked after 3 failures")
	}
	if result.Reason != "ip_blocked" {
		t.Errorf("expected reason 'ip_blocked', got '%s'", result.Reason)
	}
}

func TestSuccessClearsFailures(t *testing.T) {
	p := New(DefaultConfig())
	p.RecordFailure("user3", "1.2.3.4", "invalid")
	p.RecordFailure("user3", "1.2.3.4", "invalid")

	p.RecordSuccess("user3", "1.2.3.4")

	attempts, lockedUntil := p.GetAccountStatus("user3")
	if attempts != 0 {
		t.Errorf("expected 0 attempts after success, got %d", attempts)
	}
	if !lockedUntil.IsZero() {
		t.Error("expected no lock after success")
	}
}

func TestUnlockAccount(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxAttempts = 2
	p := New(cfg)

	p.RecordFailure("user4", "1.2.3.4", "invalid")
	p.RecordFailure("user4", "1.2.3.4", "invalid")

	// Should be locked
	result := p.Check("user4", "1.2.3.4")
	if result.Allowed {
		t.Error("expected locked")
	}

	// Unlock
	unlocked := p.UnlockAccount("user4")
	if !unlocked {
		t.Error("expected unlock to succeed")
	}

	// Should be allowed now
	result = p.Check("user4", "1.2.3.4")
	if !result.Allowed {
		t.Error("expected allowed after unlock")
	}
}

func TestUnlockIP(t *testing.T) {
	cfg := DefaultConfig()
	cfg.IPMaxAttempts = 2
	p := New(cfg)

	p.RecordFailure("user_a", "10.0.0.2", "invalid")
	p.RecordFailure("user_b", "10.0.0.2", "invalid")

	unlocked := p.UnlockIP("10.0.0.2")
	if !unlocked {
		t.Error("expected IP unlock to succeed")
	}
}

func TestListLockedAccounts(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxAttempts = 2
	p := New(cfg)

	p.RecordFailure("locked1", "1.1.1.1", "invalid")
	p.RecordFailure("locked1", "1.1.1.1", "invalid")

	locked := p.ListLockedAccounts()
	if len(locked) != 1 {
		t.Fatalf("expected 1 locked account, got %d", len(locked))
	}
	if locked[0]["username"] != "locked1" {
		t.Errorf("expected 'locked1', got %v", locked[0]["username"])
	}
}

func TestListBlockedIPs(t *testing.T) {
	cfg := DefaultConfig()
	cfg.IPMaxAttempts = 2
	p := New(cfg)

	p.RecordFailure("user_a", "10.0.0.5", "invalid")
	p.RecordFailure("user_b", "10.0.0.5", "invalid")

	blocked := p.ListBlockedIPs()
	if len(blocked) != 1 {
		t.Fatalf("expected 1 blocked IP, got %d", len(blocked))
	}
	if blocked[0]["ip"] != "10.0.0.5" {
		t.Errorf("expected '10.0.0.5', got %v", blocked[0]["ip"])
	}
}

func TestIsAccountLockedError(t *testing.T) {
	tests := []struct {
		result   CheckResult
		contains string
	}{
		{CheckResult{Reason: "account_locked", LockedUntil: time.Now().Add(time.Minute)}, "account locked"},
		{CheckResult{Reason: "ip_blocked", LockedUntil: time.Now().Add(time.Minute)}, "too many failed attempts"},
		{CheckResult{Allowed: true}, ""},
	}
	for _, tc := range tests {
		msg := IsAccountLockedError(tc.result)
		if tc.contains != "" && len(msg) == 0 {
			t.Errorf("expected error message containing %q, got empty", tc.contains)
		}
	}
}

func TestWindowExpiration(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxAttempts = 3
	cfg.WindowDuration = 50 * time.Millisecond
	p := New(cfg)

	p.RecordFailure("user5", "1.2.3.4", "invalid")
	p.RecordFailure("user5", "1.2.3.4", "invalid")

	time.Sleep(100 * time.Millisecond)

	// Old failures should be pruned, account should not be locked
	p.RecordFailure("user5", "1.2.3.4", "invalid")
	result := p.Check("user5", "1.2.3.4")
	if !result.Allowed {
		t.Error("expected allowed after window expiration (only 1 recent failure)")
	}
}

func TestGetAccountStatusNonExistent(t *testing.T) {
	p := New(DefaultConfig())
	attempts, lockedUntil := p.GetAccountStatus("nonexistent")
	if attempts != 0 {
		t.Errorf("expected 0 attempts, got %d", attempts)
	}
	if !lockedUntil.IsZero() {
		t.Error("expected zero lock time")
	}
}

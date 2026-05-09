package auth

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestMemoryTokenBlacklist_RevokeAndCheck(t *testing.T) {
	bl := NewMemoryTokenBlacklist()

	if revoked, _ := bl.IsRevoked("jti-1"); revoked {
		t.Error("new token should not be revoked")
	}

	if err := bl.Revoke("jti-1", 5*time.Minute); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	if revoked, _ := bl.IsRevoked("jti-1"); !revoked {
		t.Error("token should be revoked after Revoke()")
	}

	if revoked, _ := bl.IsRevoked("jti-2"); revoked {
		t.Error("different token should not be revoked")
	}
}

func TestMemoryTokenBlacklist_Expired(t *testing.T) {
	bl := NewMemoryTokenBlacklist()

	bl.Revoke("jti-expired", 1*time.Nanosecond)
	time.Sleep(10 * time.Millisecond)

	if revoked, _ := bl.IsRevoked("jti-expired"); revoked {
		t.Error("expired token should not be considered revoked")
	}
}

func TestMemoryTokenBlacklist_Cleanup(t *testing.T) {
	bl := NewMemoryTokenBlacklist()

	bl.Revoke("jti-keep", 1*time.Hour)
	bl.Revoke("jti-expire", 1*time.Nanosecond)
	time.Sleep(10 * time.Millisecond)

	bl.cleanup()

	if revoked, _ := bl.IsRevoked("jti-keep"); !revoked {
		t.Error("non-expired token should still be revoked after cleanup")
	}
	if revoked, _ := bl.IsRevoked("jti-expire"); revoked {
		t.Error("expired token should be removed after cleanup")
	}
}

func TestRedisTokenBlacklist_RevokeAndCheck(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	bl := NewRedisTokenBlacklist(client)

	if revoked, _ := bl.IsRevoked("jti-redis-1"); revoked {
		t.Error("new token should not be revoked")
	}

	if err := bl.Revoke("jti-redis-1", 5*time.Minute); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	if revoked, _ := bl.IsRevoked("jti-redis-1"); !revoked {
		t.Error("token should be revoked after Revoke()")
	}
}

func TestRedisTokenBlacklist_Expired(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	bl := NewRedisTokenBlacklist(client)
	bl.Revoke("jti-redis-exp", 1*time.Second)

	mr.FastForward(2 * time.Second)

	if revoked, _ := bl.IsRevoked("jti-redis-exp"); revoked {
		t.Error("expired token should not be considered revoked in Redis")
	}
}

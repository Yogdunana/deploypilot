package service

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemoryCache_SetAndGet(t *testing.T) {
	cache := NewMemoryCache("test:")
	defer cache.Close()
	ctx := context.Background()

	if err := cache.Set(ctx, "foo", "bar", 10*time.Second); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := cache.Get(ctx, "foo")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != "bar" {
		t.Errorf("Get(foo) = %q, want %q", got, "bar")
	}
}

func TestMemoryCache_GetMissing(t *testing.T) {
	cache := NewMemoryCache("test:")
	defer cache.Close()
	ctx := context.Background()

	_, err := cache.Get(ctx, "missing")
	if err != ErrCacheMiss {
		t.Errorf("Get(missing) err = %v, want %v", err, ErrCacheMiss)
	}
}

func TestMemoryCache_KeyPrefixing(t *testing.T) {
	cache := NewMemoryCache("prefix:")
	defer cache.Close()
	ctx := context.Background()

	if err := cache.Set(ctx, "k", "v", 10*time.Second); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// "k" stored with prefix, so "other" should not hit it
	_, err := cache.Get(ctx, "other")
	if err != ErrCacheMiss {
		t.Errorf("Get(other) err = %v, want %v", err, ErrCacheMiss)
	}

	got, err := cache.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get(k) failed: %v", err)
	}
	if got != "v" {
		t.Errorf("Get(k) = %q, want %q", got, "v")
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache("test:")
	defer cache.Close()
	ctx := context.Background()

	if err := cache.Set(ctx, "del", "me", 10*time.Second); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := cache.Delete(ctx, "del"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err := cache.Get(ctx, "del")
	if err != ErrCacheMiss {
		t.Errorf("Get after Delete = %v, want %v", err, ErrCacheMiss)
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	cache := NewMemoryCache("test:")
	defer cache.Close()
	ctx := context.Background()

	err := cache.Set(ctx, "expiring", "value", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := cache.Get(ctx, "expiring")
	if err != nil {
		t.Fatalf("Get immediately after Set failed: %v", err)
	}
	if got != "value" {
		t.Errorf("Get = %q, want %q", got, "value")
	}

	time.Sleep(100 * time.Millisecond)

	_, err = cache.Get(ctx, "expiring")
	if err != ErrCacheMiss {
		t.Errorf("Get after expiration err = %v, want %v", err, ErrCacheMiss)
	}
}

func TestMemoryCache_Overwrite(t *testing.T) {
	cache := NewMemoryCache("test:")
	defer cache.Close()
	ctx := context.Background()

	if err := cache.Set(ctx, "k", "v1", 10*time.Second); err != nil {
		t.Fatalf("Set v1 failed: %v", err)
	}
	if err := cache.Set(ctx, "k", "v2", 10*time.Second); err != nil {
		t.Fatalf("Set v2 failed: %v", err)
	}
	got, err := cache.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != "v2" {
		t.Errorf("Get(k) = %q, want %q", got, "v2")
	}
}

func TestMemoryCache_JSON(t *testing.T) {
	cache := NewMemoryCache("test:")
	defer cache.Close()
	ctx := context.Background()

	type testStruct struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	in := testStruct{Name: "test", Count: 42}
	if err := cache.SetJSON(ctx, "json-key", in, 10*time.Second); err != nil {
		t.Fatalf("SetJSON failed: %v", err)
	}

	var out testStruct
	if err := cache.GetJSON(ctx, "json-key", &out); err != nil {
		t.Fatalf("GetJSON failed: %v", err)
	}
	if out.Name != "test" || out.Count != 42 {
		t.Errorf("GetJSON = %+v, want %+v", out, in)
	}
}

func TestMemoryCache_GetJSONMissing(t *testing.T) {
	cache := NewMemoryCache("test:")
	defer cache.Close()
	ctx := context.Background()

	var dest map[string]string
	err := cache.GetJSON(ctx, "nope", &dest)
	if err != ErrCacheMiss {
		t.Errorf("GetJSON(missing) err = %v, want %v", err, ErrCacheMiss)
	}
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewMemoryCache("concurrent:")
	defer cache.Close()
	ctx := context.Background()

	const n = 50
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%26))
			_ = cache.Set(ctx, key, "val", 5*time.Second)
			_, _ = cache.Get(ctx, key)
		}(i)
	}
	wg.Wait()
	// No race is the pass condition when running with -race.
}

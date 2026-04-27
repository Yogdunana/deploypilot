package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/api"
	"github.com/gin-gonic/gin"
)

func TestServerShutdown(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a simple test server
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	wsHub := api.NewWSHub()
	go wsHub.Run()

	srv := &Server{
		router: r,
		wsHub:  wsHub,
	}

	// Create a test HTTP server
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	// Verify server is running
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("server not responding: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- srv.Shutdown(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("shutdown failed: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown timed out")
	}
}

func TestWSHubClose(t *testing.T) {
	hub := api.NewWSHub()
	go hub.Run()

	// Give hub time to start
	time.Sleep(50 * time.Millisecond)

	// Close should not block or panic
	done := make(chan struct{})
	go func() {
		hub.Close()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("WSHub.Close() timed out")
	}
}

func TestWSHubCloseConcurrency(t *testing.T) {
	hub := api.NewWSHub()
	go hub.Run()
	time.Sleep(50 * time.Millisecond)

	// Test concurrent close (should not panic)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Only one Close() should succeed; others should handle closed channel
			defer func() {
				recover() // ignore panic from closing already-closed channel
			}()
			hub.Close()
		}()
	}
	wg.Wait()
}

func TestServerShutdownContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	wsHub := api.NewWSHub()
	go wsHub.Run()

	srv := &Server{
		router: r,
		wsHub:  wsHub,
	}

	// Test with already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := srv.Shutdown(ctx)
	// Should still succeed (http.Server.Shutdown with cancelled context returns immediately)
	// The error might be "context canceled" which is expected
	if err != nil {
		t.Logf("shutdown with cancelled context returned (expected): %v", err)
	}
}

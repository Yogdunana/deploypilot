package api

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupSSETestDB creates an in-memory SQLite database for SSE tests.
func setupSSETestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	db.Exec(`CREATE TABLE IF NOT EXISTS apps (
		id TEXT PRIMARY KEY, tenant_id TEXT, server_id TEXT,
		name TEXT NOT NULL, repo_url TEXT NOT NULL, branch TEXT DEFAULT 'main',
		domain TEXT, tech_stack TEXT DEFAULT 'docker', deploy_mode TEXT DEFAULT 'api',
		status TEXT DEFAULT 'pending', current_version TEXT, container_name TEXT,
		env_vars TEXT, resource_limits TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS servers (
		id TEXT PRIMARY KEY, tenant_id TEXT, credential_id TEXT, provider_id TEXT,
		name TEXT NOT NULL, host TEXT NOT NULL, port INTEGER DEFAULT 22,
		tags TEXT, status TEXT DEFAULT 'unknown', detected_info TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS credentials (
		id TEXT PRIMARY KEY, tenant_id TEXT, name TEXT NOT NULL,
		type TEXT NOT NULL, encrypted_value TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS tenants (
		id TEXT PRIMARY KEY, name TEXT NOT NULL, slug TEXT UNIQUE NOT NULL,
		plan TEXT DEFAULT 'free', max_servers INTEGER DEFAULT 5, max_apps INTEGER DEFAULT 20,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY, tenant_id TEXT, server_id TEXT,
		app_name TEXT, container_name TEXT, image TEXT,
		status TEXT DEFAULT 'deploying', preflight_code TEXT,
		preflight_message TEXT, preflight_checks TEXT, error_message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`INSERT INTO tenants (id, name, slug) VALUES ('tenant-default', 'Default', 'default')`)

	return db
}

// setupSSERouter creates a Gin engine with SSE routes for testing.
func setupSSERouter(db *gorm.DB, bridge *service.Bridge) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	api := r.Group("/api/v1")
	api.Use(auth.AuthMiddleware())
	{
		api.GET("/sse/deploy/:app_id", DeploySSE(bridge))
	}
	return r
}

// parseSSELines parses SSE data from an HTTP response body.
func parseSSELines(body string) []map[string]string {
	var events []map[string]string
	scanner := bufio.NewScanner(strings.NewReader(body))
	var currentEvent map[string]string
	var currentData strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event:") {
			if currentEvent != nil {
				currentEvent["data"] = strings.TrimSpace(currentData.String())
				events = append(events, currentEvent)
			}
			currentEvent = map[string]string{"event": strings.TrimSpace(strings.TrimPrefix(line, "event:"))}
			currentData.Reset()
		} else if strings.HasPrefix(line, "data:") {
			currentData.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		} else if line == "" && currentEvent != nil {
			currentEvent["data"] = strings.TrimSpace(currentData.String())
			events = append(events, currentEvent)
			currentEvent = nil
			currentData.Reset()
		}
	}
	// Handle last event if no trailing newline
	if currentEvent != nil {
		currentEvent["data"] = strings.TrimSpace(currentData.String())
		events = append(events, currentEvent)
	}
	return events
}

func getSSEToken(t *testing.T) string {
	t.Helper()
	token, err := auth.GenerateToken("user-sse-test", "owner")
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestDeploySSE_Connection(t *testing.T) {
	db := setupSSETestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)

	router := setupSSERouter(db, bridge)
	server := httptest.NewServer(router)
	defer server.Close()

	token := getSSEToken(t)
	req, _ := http.NewRequest("GET", server.URL+"/api/v1/sse/deploy/app-1", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Use a cancellable context to stop the SSE stream
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Check SSE headers
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	if resp.Header.Get("Cache-Control") != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", resp.Header.Get("Cache-Control"))
	}
	if resp.Header.Get("Connection") != "keep-alive" {
		t.Errorf("Connection = %q, want keep-alive", resp.Header.Get("Connection"))
	}
}

func TestDeploySSE_DeployEvent(t *testing.T) {
	db := setupSSETestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)

	router := setupSSERouter(db, bridge)
	server := httptest.NewServer(router)
	defer server.Close()

	token := getSSEToken(t)
	req, _ := http.NewRequest("GET", server.URL+"/api/v1/sse/deploy/app-sse-test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Wait for the connection event to be sent
	time.Sleep(300 * time.Millisecond)

	// Publish a deploy event
	bridge.EventBus.Publish(service.DeployEvent{
		TaskID:    "task-1",
		AppID:     "app-sse-test",
		Step:      "pull",
		Status:    "running",
		Progress:  30,
		Message:   "pulling image",
		Timestamp: time.Now().Format(time.RFC3339),
	})

	// Read the response body with a deadline
	var body strings.Builder
	buf := make([]byte, 4096)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				body.Write(buf[:n])
			}
			if readErr != nil {
				return
			}
		}
	}()

	// Wait for the event to arrive or timeout
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
	cancel()

	// Wait for reader goroutine to finish before accessing body
	<-done

	events := parseSSELines(body.String())

	// Should have at least a "connected" event
	foundConnected := false
	foundDeploy := false
	for _, ev := range events {
		if ev["event"] == "connected" {
			foundConnected = true
		}
		if ev["event"] == "deploy" {
			foundDeploy = true
			var event service.DeployEvent
			if err := json.Unmarshal([]byte(ev["data"]), &event); err != nil {
				t.Errorf("failed to unmarshal deploy event: %v", err)
			}
			if event.Step != "pull" {
				t.Errorf("Step = %q, want %q", event.Step, "pull")
			}
			if event.Progress != 30 {
				t.Errorf("Progress = %d, want %d", event.Progress, 30)
			}
		}
	}

	if !foundConnected {
		t.Error("expected connected event")
	}
	if !foundDeploy {
		t.Errorf("expected deploy event, got events: %v", events)
	}
}

func TestDeploySSE_DoneClosesConnection(t *testing.T) {
	db := setupSSETestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)

	router := setupSSERouter(db, bridge)
	server := httptest.NewServer(router)
	defer server.Close()

	token := getSSEToken(t)
	req, _ := http.NewRequest("GET", server.URL+"/api/v1/sse/deploy/app-done-test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Wait for connection, then publish a "done" event
	time.Sleep(300 * time.Millisecond)
	bridge.EventBus.Publish(service.DeployEvent{
		TaskID:    "task-done",
		AppID:     "app-done-test",
		Step:      "done",
		Status:    "success",
		Progress:  100,
		Message:   "deploy completed",
		Timestamp: time.Now().Format(time.RFC3339),
	})

	// Read response - connection should close after "done" event
	var body strings.Builder
	buf := make([]byte, 4096)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				body.Write(buf[:n])
			}
			if readErr != nil {
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for connection to close after done event")
	}

	bodyStr := body.String()
	if !strings.Contains(bodyStr, "deploy completed") {
		t.Errorf("response body should contain 'deploy completed', got: %s", bodyStr)
	}
}

func TestDeploySSE_Unauthorized(t *testing.T) {
	db := setupSSETestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)

	router := setupSSERouter(db, bridge)
	server := httptest.NewServer(router)
	defer server.Close()

	// Request without auth token
	resp, err := http.Get(server.URL + "/api/v1/sse/deploy/app-1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestDeploySSE_Heartbeat(t *testing.T) {
	db := setupSSETestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)

	router := setupSSERouter(db, bridge)
	server := httptest.NewServer(router)
	defer server.Close()

	token := getSSEToken(t)
	req, _ := http.NewRequest("GET", server.URL+"/api/v1/sse/deploy/app-heartbeat", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Wait for connected event
	time.Sleep(500 * time.Millisecond)

	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])

	if !strings.Contains(body, "connected") {
		t.Error("expected connected event in SSE stream")
	}

	cancel()
}

func TestDeployAsyncHandler_InvalidJSON(t *testing.T) {
	db := setupSSETestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/apps/:id/deploy", DeployAsyncHandler(bridge))

	server := httptest.NewServer(r)
	defer server.Close()

	token := getSSEToken(t)
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/apps/app-1/deploy?async=true", strings.NewReader("not json"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

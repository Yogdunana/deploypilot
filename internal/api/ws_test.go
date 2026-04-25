package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupWSTestDB creates an in-memory SQLite database for WebSocket tests.
func setupWSTestDB(t *testing.T) *gorm.DB {
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
	db.Exec(`INSERT INTO tenants (id, name, slug) VALUES ('tenant-default', 'Default', 'default')`)

	return db
}

func getWSToken(t *testing.T) string {
	t.Helper()
	token, err := auth.GenerateToken("user-ws-test", "owner")
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// setupWSRouter creates a Gin engine with WebSocket routes for testing.
func setupWSRouter(db *gorm.DB, bridge *service.Bridge, hub *WSHub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})
	ticketStore := auth.NewWSTicketStore()
	wsGroup := r.Group("/ws")
	{
		wsGroup.GET("/logs/:app_id", LogStreamWS(bridge, hub, ticketStore))
		wsGroup.GET("/terminal/:server_id", TerminalWS(bridge, hub, ticketStore))
	}
	return r
}

func dialWS(t *testing.T, server *httptest.Server, path string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + path
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	return conn
}

// --- WSHub Tests ---

func TestNewWSHub(t *testing.T) {
	hub := NewWSHub()
	if hub == nil {
		t.Fatal("NewWSHub() returned nil")
	}
	if hub.clients == nil {
		t.Fatal("expected non-nil clients map")
	}
	if hub.register == nil {
		t.Fatal("expected non-nil register channel")
	}
	if hub.unregister == nil {
		t.Fatal("expected non-nil unregister channel")
	}
}

func TestWSHubRegisterUnregister(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()

	registered := make(chan struct{})
	unregistered := make(chan struct{})

	// Create a test server that registers/unregisters the server-side conn
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		hub.Register(conn, "app-1")
		time.Sleep(50 * time.Millisecond)
		close(registered)

		// Block until client disconnects (ReadMessage returns error)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
		hub.Unregister(conn, "app-1")
		time.Sleep(50 * time.Millisecond)
		close(unregistered)
	}))
	defer server.Close()

	wsConn := dialWS(t, server, "/")
	defer wsConn.Close()

	// Wait for registration
	select {
	case <-registered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for registration")
	}

	count := hub.ClientCount("app-1")
	if count != 1 {
		t.Errorf("ClientCount(app-1) = %d, want 1", count)
	}

	// Close the client connection to trigger server-side cleanup
	wsConn.Close()

	select {
	case <-unregistered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for unregistration")
	}

	count = hub.ClientCount("app-1")
	if count != 0 {
		t.Errorf("ClientCount(app-1) after unregister = %d, want 0", count)
	}
}

func TestWSHubBroadcast(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()

	// Create a server that registers the server-side conn with the hub
	// and then broadcasts a message after a short delay
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer serverConn.Close()

		hub.Register(serverConn, "app-broadcast")
		time.Sleep(100 * time.Millisecond) // ensure registration is processed

		// Broadcast a message - this writes to serverConn, sending it to the client
		msg := WSMessage{
			Type:      "log",
			Timestamp: time.Now().Format(time.RFC3339),
			Data:      "test log line",
			AppID:     "app-broadcast",
		}
		hub.Broadcast("app-broadcast", msg)

		// Wait a bit for the write to complete
		time.Sleep(100 * time.Millisecond)
		close(done)
	}))
	defer server.Close()

	clientConn := dialWS(t, server, "/")
	defer clientConn.Close()

	// Wait for the server to broadcast
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for broadcast")
	}

	// Read the broadcast message from the client side
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, p, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read broadcast: %v", err)
	}

	var received WSMessage
	if err := json.Unmarshal(p, &received); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}
	if received.Type != "log" {
		t.Errorf("received.Type = %q, want %q", received.Type, "log")
	}
	if received.Data != "test log line" {
		t.Errorf("received.Data = %v, want %q", received.Data, "test log line")
	}
}

func TestWSHubClientCount(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()

	// No clients for unknown app
	count := hub.ClientCount("nonexistent")
	if count != 0 {
		t.Errorf("ClientCount(nonexistent) = %d, want 0", count)
	}

	ready := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		hub.Register(conn, "app-count")
		time.Sleep(50 * time.Millisecond)
		close(ready)

		// Keep alive until client disconnects
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	conn1 := dialWS(t, server, "/")
	defer conn1.Close()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for registration")
	}

	count = hub.ClientCount("app-count")
	if count != 1 {
		t.Errorf("ClientCount(app-count) = %d, want 1", count)
	}

	// Dial a second connection
	ready2 := make(chan struct{})
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		hub.Register(conn, "app-count")
		time.Sleep(50 * time.Millisecond)
		close(ready2)

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server2.Close()

	conn2 := dialWS(t, server2, "/")
	defer conn2.Close()

	select {
	case <-ready2:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second registration")
	}

	count = hub.ClientCount("app-count")
	if count != 2 {
		t.Errorf("ClientCount(app-count) = %d, want 2", count)
	}
}

func TestWSMessageJSON(t *testing.T) {
	msg := WSMessage{
		Type:      "log",
		Timestamp: "2024-01-01T00:00:00Z",
		Data:      "hello world",
		AppID:     "app-123",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded WSMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Type != msg.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, msg.Type)
	}
	if decoded.Timestamp != msg.Timestamp {
		t.Errorf("Timestamp = %q, want %q", decoded.Timestamp, msg.Timestamp)
	}
	if decoded.Data != msg.Data {
		t.Errorf("Data = %v, want %v", decoded.Data, msg.Data)
	}
	if decoded.AppID != msg.AppID {
		t.Errorf("AppID = %q, want %q", decoded.AppID, msg.AppID)
	}

	// Test without AppID
	msgNoApp := WSMessage{Type: "ping", Timestamp: "2024-01-01T00:00:00Z", Data: nil}
	data2, _ := json.Marshal(msgNoApp)
	var decoded2 WSMessage
	json.Unmarshal(data2, &decoded2)
	if decoded2.AppID != "" {
		t.Errorf("expected empty AppID when omitted, got %q", decoded2.AppID)
	}
}

// --- LogStreamWS Handler Tests ---

func TestLogStreamWS_NoToken(t *testing.T) {
	db := setupWSTestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)
	hub := NewWSHub()
	go hub.Run()

	router := setupWSRouter(db, bridge, hub)
	server := httptest.NewServer(router)
	defer server.Close()

	// Make a regular HTTP request (no WebSocket upgrade) to check auth
	resp, err := http.Get(server.URL + "/ws/logs/app-1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestLogStreamWS_InvalidToken(t *testing.T) {
	db := setupWSTestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)
	hub := NewWSHub()
	go hub.Run()

	router := setupWSRouter(db, bridge, hub)
	server := httptest.NewServer(router)
	defer server.Close()

	// Try to connect with invalid token
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/logs/app-1?token=invalid-token"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

// --- TerminalWS Handler Tests ---

func TestTerminalWS_NoToken(t *testing.T) {
	db := setupWSTestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)
	hub := NewWSHub()
	go hub.Run()

	router := setupWSRouter(db, bridge, hub)
	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/ws/terminal/server-1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestTerminalWS_InvalidToken(t *testing.T) {
	db := setupWSTestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)
	hub := NewWSHub()
	go hub.Run()

	router := setupWSRouter(db, bridge, hub)
	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/terminal/server-1?token=bad-token"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestTerminalWS_ServerNotFound(t *testing.T) {
	db := setupWSTestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)
	hub := NewWSHub()
	go hub.Run()

	router := setupWSRouter(db, bridge, hub)
	server := httptest.NewServer(router)
	defer server.Close()

	token := getWSToken(t)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/terminal/nonexistent-server?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, p, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var msg WSMessage
	json.Unmarshal(p, &msg)
	if msg.Type != "error" {
		t.Errorf("expected error message type, got %q", msg.Type)
	}
}

func TestLogStreamWS_AppNotFound(t *testing.T) {
	db := setupWSTestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)
	hub := NewWSHub()
	go hub.Run()

	router := setupWSRouter(db, bridge, hub)
	server := httptest.NewServer(router)
	defer server.Close()

	token := getWSToken(t)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/logs/nonexistent-app?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, p, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var msg WSMessage
	json.Unmarshal(p, &msg)
	if msg.Type != "error" {
		t.Errorf("expected error message type, got %q", msg.Type)
	}
}

func TestWSHub_Send(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()

	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer serverConn.Close()

		// Send a message directly to the client via the server-side conn
		msg := WSMessage{Type: "status", Data: "connected"}
		hub.Send(serverConn, msg)

		time.Sleep(100 * time.Millisecond)
		close(done)
	}))
	defer server.Close()

	clientConn := dialWS(t, server, "/")
	defer clientConn.Close()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for send")
	}

	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, p, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read sent message: %v", err)
	}

	var received WSMessage
	json.Unmarshal(p, &received)
	if received.Type != "status" {
		t.Errorf("received.Type = %q, want %q", received.Type, "status")
	}
}

func TestWSHub_BroadcastNoClients(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()

	// Broadcasting to an app with no clients should not panic
	msg := WSMessage{Type: "log", Data: "no clients"}
	hub.Broadcast("nonexistent-app", msg)
	// If we get here without panic, the test passes
}

func TestWSHub_BroadcastMarshalError(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()

	// Broadcasting a message with unmarshalable data should not panic
	msg := WSMessage{Type: "log", Data: make(chan int)}
	hub.Broadcast("app-test", msg)
	// If we get here without panic, the test passes
}

func TestWSHub_SendMarshalError(t *testing.T) {
	hub := NewWSHub()

	// Create a server that upgrades and immediately closes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		conn.Close()
	}))
	defer server.Close()

	wsConn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/", nil)
	if err != nil {
		// Connection may fail since server closes immediately, that's fine
		return
	}
	defer wsConn.Close()

	// Send with unmarshalable data - should not panic
	msg := WSMessage{Type: "test", Data: make(chan int)}
	hub.Send(wsConn, msg)
}

func TestWsToString(t *testing.T) {
	// Test nil
	if wsToString(nil) != "" {
		t.Error("wsToString(nil) should return empty string")
	}

	// Test string
	if wsToString("hello") != "hello" {
		t.Error("wsToString(string) should return the string")
	}

	// Test []byte
	if wsToString([]byte("bytes")) != "bytes" {
		t.Error("wsToString([]byte) should return the string")
	}

	// Test int
	result := wsToString(42)
	if result != "42" {
		t.Errorf("wsToString(42) = %q, want %q", result, "42")
	}

	// Test empty string
	if wsToString("") != "" {
		t.Error("wsToString('') should return empty string")
	}
}

func TestAgentTunnelWS_NoToken(t *testing.T) {
	db := setupWSTestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	ticketStore := auth.NewWSTicketStore()
	r.GET("/ws/agent/:server_id", AgentTunnelWS(bridge, ticketStore))

	server := httptest.NewServer(r)
	defer server.Close()

	// Dial without token — should get HTTP 401, not a WebSocket upgrade
	resp, err := http.Get(server.URL + "/ws/agent/test-server")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAgentTunnelWS_InvalidToken(t *testing.T) {
	db := setupWSTestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	ticketStore := auth.NewWSTicketStore()
	r.GET("/ws/agent/:server_id", AgentTunnelWS(bridge, ticketStore))

	server := httptest.NewServer(r)
	defer server.Close()

	resp, err := http.Get(server.URL + "/ws/agent/test-server?token=invalid")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAgentTunnelWS_NoTunnelManager(t *testing.T) {
	db := setupWSTestDB(t)
	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)
	// TunnelManager is nil by default

	gin.SetMode(gin.TestMode)
	r := gin.New()
	ticketStore := auth.NewWSTicketStore()
	r.GET("/ws/agent/:server_id", AgentTunnelWS(bridge, ticketStore))

	server := httptest.NewServer(r)
	defer server.Close()

	token := getWSToken(t)
	resp, err := http.Get(server.URL + "/ws/agent/test-server?token=" + token)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}
}

func TestLogStreamWS_ValidApp(t *testing.T) {
	db := setupWSTestDB(t)
	// Insert a test app with container_name
	db.Exec(`INSERT INTO apps (id, tenant_id, name, repo_url, container_name) VALUES ('app-ws-1', 'tenant-default', 'ws-app', 'https://github.com/test/test', 'ws-app-container')`)

	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)
	hub := NewWSHub()
	go hub.Run()

	router := setupWSRouter(db, bridge, hub)
	server := httptest.NewServer(router)
	defer server.Close()

	token := getWSToken(t)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/logs/app-ws-1?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// Read at least one message (log or error) within a few seconds
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, p, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var msg WSMessage
	json.Unmarshal(p, &msg)
	// Should be either "log" or "error" type
	if msg.Type != "log" && msg.Type != "error" {
		t.Errorf("expected log or error message type, got %q", msg.Type)
	}
}

func TestLogStreamWS_AppWithFallbackName(t *testing.T) {
	db := setupWSTestDB(t)
	// Insert app with no container_name but with name
	db.Exec(`INSERT INTO apps (id, tenant_id, name, repo_url, container_name) VALUES ('app-ws-2', 'tenant-default', 'fallback-app', 'https://github.com/test/test', '')`)

	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)
	hub := NewWSHub()
	go hub.Run()

	router := setupWSRouter(db, bridge, hub)
	server := httptest.NewServer(router)
	defer server.Close()

	token := getWSToken(t)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/logs/app-ws-2?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// Should connect and start streaming (using fallback name)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
}

func TestTerminalWS_ValidServer(t *testing.T) {
	db := setupWSTestDB(t)
	// Insert a test server
	db.Exec(`INSERT INTO servers (id, tenant_id, name, host, port, status) VALUES ('srv-ws-1', 'tenant-default', 'ws-server', '192.168.1.1', 22, 'reachable')`)
	// Insert a credential for the server
	db.Exec(`INSERT INTO credentials (id, tenant_id, name, type, encrypted_value) VALUES ('cred-ws-1', 'tenant-default', 'ssh-key', 'ssh', 'encrypted-value')`)

	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)
	hub := NewWSHub()
	go hub.Run()

	router := setupWSRouter(db, bridge, hub)
	server := httptest.NewServer(router)
	defer server.Close()

	token := getWSToken(t)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/terminal/srv-ws-1?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// Send an input command
	cmdMsg := WSMessage{Type: "input", Data: "ls"}
	data, _ := json.Marshal(cmdMsg)
	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Read the output
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, p, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var msg WSMessage
	json.Unmarshal(p, &msg)
	// Terminal may return "output" or "error" depending on SSH availability
	if msg.Type != "output" && msg.Type != "error" {
		t.Errorf("expected output or error message type, got %q", msg.Type)
	}
}

func TestTerminalWS_InvalidCommand(t *testing.T) {
	db := setupWSTestDB(t)
	db.Exec(`INSERT INTO servers (id, tenant_id, name, host, port, status) VALUES ('srv-ws-3', 'tenant-default', 'ws-server-3', '192.168.1.3', 22, 'reachable')`)
	db.Exec(`INSERT INTO credentials (id, tenant_id, name, type, encrypted_value) VALUES ('cred-ws-3', 'tenant-default', 'ssh-key-3', 'ssh', 'encrypted-value')`)

	bridge := service.NewBridge(db, &localExecutor{}, []byte("test-key-1234567890abcdef"), nil)
	hub := NewWSHub()
	go hub.Run()

	router := setupWSRouter(db, bridge, hub)
	server := httptest.NewServer(router)
	defer server.Close()

	token := getWSToken(t)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/terminal/srv-ws-3?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// Send a non-input message type (should be ignored)
	cmdMsg := WSMessage{Type: "ping", Data: nil}
	data, _ := json.Marshal(cmdMsg)
	conn.WriteMessage(websocket.TextMessage, data)

	// Send invalid JSON
	conn.WriteMessage(websocket.TextMessage, []byte("not json"))

	// Close connection to exit read loop
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

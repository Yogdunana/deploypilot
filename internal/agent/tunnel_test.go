package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewTunnelManager(t *testing.T) {
	tm := NewTunnelManager()
	if tm == nil {
		t.Fatal("NewTunnelManager() returned nil")
	}
	if tm.agents == nil {
		t.Fatal("expected non-nil agents map")
	}
	if tm.commands == nil {
		t.Fatal("expected non-nil commands map")
	}
}

func TestTunnelManager_IsConnected(t *testing.T) {
	tm := NewTunnelManager()

	// Not connected initially
	if tm.IsConnected("server-1") {
		t.Error("expected IsConnected to return false for unknown server")
	}

	// After connecting
	tm.mu.Lock()
	tm.agents["server-1"] = &agentConn{
		conn:     nil, // nil conn is fine for this test
		serverID: "server-1",
		lastSeen: time.Now(),
	}
	tm.mu.Unlock()

	if !tm.IsConnected("server-1") {
		t.Error("expected IsConnected to return true for connected server")
	}
}

func TestTunnelManager_ListAgents(t *testing.T) {
	tm := NewTunnelManager()

	// Empty initially
	agents := tm.ListAgents()
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}

	// Add some agents
	tm.mu.Lock()
	tm.agents["server-1"] = &agentConn{serverID: "server-1", lastSeen: time.Now()}
	tm.agents["server-2"] = &agentConn{serverID: "server-2", lastSeen: time.Now()}
	tm.mu.Unlock()

	agents = tm.ListAgents()
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}

func TestTunnelManager_ExecuteCommand_Timeout(t *testing.T) {
	tm := NewTunnelManager()

	// No agent connected
	_, err := tm.ExecuteCommand(context.Background(), "nonexistent", "echo hello", 1*time.Second)
	if err == nil {
		t.Fatal("expected error when agent not connected")
	}
	if !strings.Contains(err.Error(), "agent not connected") {
		t.Errorf("expected 'agent not connected' error, got: %v", err)
	}
}

func TestTunnelManager_ExecuteCommand_ContextCancelled(t *testing.T) {
	tm := NewTunnelManager()

	// Create a mock server that accepts WebSocket but never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Keep connection open, read messages
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	// Connect agent
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial WebSocket: %v", err)
	}
	defer wsConn.Close()

	tm.mu.Lock()
	tm.agents["test-server"] = &agentConn{
		conn:     wsConn,
		serverID: "test-server",
		lastSeen: time.Now(),
	}
	tm.mu.Unlock()

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err = tm.ExecuteCommand(ctx, "test-server", "echo hello", 30*time.Second)
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
}

func TestTunnelManager_ExecuteCommand_TimeoutOnNoResponse(t *testing.T) {
	tm := NewTunnelManager()

	// Create a mock server that accepts WebSocket but never responds to commands
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
			// Don't send a response
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial WebSocket: %v", err)
	}
	defer wsConn.Close()

	tm.mu.Lock()
	tm.agents["test-server"] = &agentConn{
		conn:     wsConn,
		serverID: "test-server",
		lastSeen: time.Now(),
	}
	tm.mu.Unlock()

	// Should timeout
	_, err = tm.ExecuteCommand(context.Background(), "test-server", "echo hello", 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected 'timed out' error, got: %v", err)
	}
}

func TestTunnelManager_Cleanup(t *testing.T) {
	tm := NewTunnelManager()

	// Add a stale agent
	tm.mu.Lock()
	tm.agents["stale-server"] = &agentConn{
		conn:     nil,
		serverID: "stale-server",
		lastSeen: time.Now().Add(-10 * time.Minute), // 10 minutes ago
	}
	tm.agents["fresh-server"] = &agentConn{
		conn:     nil,
		serverID: "fresh-server",
		lastSeen: time.Now(), // now
	}
	tm.mu.Unlock()

	// Run cleanup
	tm.cleanup()

	// Stale should be removed, fresh should remain
	if tm.IsConnected("stale-server") {
		t.Error("expected stale-server to be cleaned up")
	}
	if !tm.IsConnected("fresh-server") {
		t.Error("expected fresh-server to remain")
	}
}

func TestTunnelManager_StartCleanup(t *testing.T) {
	tm := NewTunnelManager()
	tm.StartCleanup(50 * time.Millisecond)
	// Just verify it doesn't panic — the cleanup goroutine runs in background
	time.Sleep(100 * time.Millisecond)
}

func TestTunnelManager_HandleTunnel_RegisterAck(t *testing.T) {
	tm := NewTunnelManager()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tm.HandleTunnel(w, r, "test-agent")
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial WebSocket: %v", err)
	}
	defer conn.Close()

	// Read the registration acknowledgment
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	var tunnelMsg TunnelMessage
	if err := json.Unmarshal(msg, &tunnelMsg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if tunnelMsg.Type != "register" {
		t.Errorf("expected type 'register', got %q", tunnelMsg.Type)
	}
	if tunnelMsg.ServerID != "test-agent" {
		t.Errorf("expected server_id 'test-agent', got %q", tunnelMsg.ServerID)
	}
}

func TestTunnelManager_HandleTunnel_FullFlow(t *testing.T) {
	tm := NewTunnelManager()

	// Create a test HTTP server that handles the WebSocket upgrade
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tm.HandleTunnel(w, r, "flow-server")
	}))
	defer server.Close()

	// Connect as an agent
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial WebSocket: %v", err)
	}
	defer conn.Close()

	// Read register ack
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read register ack: %v", err)
	}
	var regMsg TunnelMessage
	if err := json.Unmarshal(msg, &regMsg); err != nil {
		t.Fatalf("failed to unmarshal register: %v", err)
	}
	if regMsg.Type != "register" {
		t.Fatalf("expected register, got %s", regMsg.Type)
	}

	// Verify agent is connected
	if !tm.IsConnected("flow-server") {
		t.Fatal("expected agent to be connected")
	}

	// Send a command to the agent via ExecuteCommand
	resultCh := make(chan *CommandResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := tm.ExecuteCommand(context.Background(), "flow-server", "echo hello", 5*time.Second)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	// Agent reads the command
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, cmdMsg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read command: %v", err)
	}
	var cmd TunnelMessage
	if err := json.Unmarshal(cmdMsg, &cmd); err != nil {
		t.Fatalf("failed to unmarshal command: %v", err)
	}
	if cmd.Type != "command" {
		t.Fatalf("expected command type, got %s", cmd.Type)
	}

	// Agent sends back a result
	resultPayload, _ := json.Marshal(CommandResult{Output: "hello"})
	resultMsg := TunnelMessage{
		Type:      "result",
		ServerID:  "flow-server",
		CommandID: cmd.CommandID,
		Payload:   resultPayload,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	if err := conn.WriteJSON(resultMsg); err != nil {
		t.Fatalf("failed to send result: %v", err)
	}

	// Verify the result was received
	select {
	case result := <-resultCh:
		if result.Output != "hello" {
			t.Errorf("expected output 'hello', got %q", result.Output)
		}
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for command result")
	}
}

func TestTunnelManager_HandleTunnel_Heartbeat(t *testing.T) {
	tm := NewTunnelManager()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tm.HandleTunnel(w, r, "hb-server")
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial WebSocket: %v", err)
	}
	defer conn.Close()

	// Read register ack
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read register ack: %v", err)
	}

	// Send a heartbeat
	hbMsg := TunnelMessage{
		Type:      "heartbeat",
		ServerID:  "hb-server",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	if err := conn.WriteJSON(hbMsg); err != nil {
		t.Fatalf("failed to send heartbeat: %v", err)
	}

	// Verify agent is still connected
	time.Sleep(50 * time.Millisecond)
	if !tm.IsConnected("hb-server") {
		t.Error("expected agent to still be connected after heartbeat")
	}
}

func TestTunnelManager_HandleTunnel_InvalidMessage(t *testing.T) {
	tm := NewTunnelManager()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tm.HandleTunnel(w, r, "invalid-msg-server")
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial WebSocket: %v", err)
	}
	defer conn.Close()

	// Read register ack
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read register ack: %v", err)
	}

	// Send an invalid message (not JSON)
	if err := conn.WriteMessage(websocket.TextMessage, []byte("not json")); err != nil {
		t.Fatalf("failed to send invalid message: %v", err)
	}

	// Send an unknown type message
	unknownMsg := TunnelMessage{
		Type:      "unknown_type",
		ServerID:  "invalid-msg-server",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	if err := conn.WriteJSON(unknownMsg); err != nil {
		t.Fatalf("failed to send unknown message: %v", err)
	}

	// Verify agent is still connected (shouldn't crash)
	time.Sleep(50 * time.Millisecond)
	if !tm.IsConnected("invalid-msg-server") {
		t.Error("expected agent to still be connected after invalid message")
	}
}

func TestTunnelManager_HandleTunnel_ResultForUnknownCommand(t *testing.T) {
	tm := NewTunnelManager()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tm.HandleTunnel(w, r, "result-server")
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial WebSocket: %v", err)
	}
	defer conn.Close()

	// Read register ack
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read register ack: %v", err)
	}

	// Send a result for a non-existent command
	resultPayload, _ := json.Marshal(CommandResult{Output: "test"})
	resultMsg := TunnelMessage{
		Type:      "result",
		ServerID:  "result-server",
		CommandID: "nonexistent-cmd-id",
		Payload:   resultPayload,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	if err := conn.WriteJSON(resultMsg); err != nil {
		t.Fatalf("failed to send result: %v", err)
	}

	// Should not panic, agent should still be connected
	time.Sleep(50 * time.Millisecond)
	if !tm.IsConnected("result-server") {
		t.Error("expected agent to still be connected")
	}
}

func TestTunnelManager_HandleTunnel_ReplaceExisting(t *testing.T) {
	tm := NewTunnelManager()

	var mu sync.Mutex
	connectCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		connectCount++
		mu.Unlock()
		tm.HandleTunnel(w, r, "replace-server")
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// First connection
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial first connection: %v", err)
	}

	// Read register ack
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn1.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read first register ack: %v", err)
	}

	// Second connection should replace the first
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial second connection: %v", err)
	}
	defer conn2.Close()

	// Read register ack
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn2.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read second register ack: %v", err)
	}

	// First connection should be closed
	time.Sleep(50 * time.Millisecond)
	_ = conn1.Close()
}

func TestTunnelMessage_JSON(t *testing.T) {
	msg := TunnelMessage{
		Type:      "command",
		ServerID:  "server-1",
		CommandID: "cmd-123",
		Payload:   json.RawMessage(`{"command":"echo hello"}`),
		Timestamp: "2024-01-01T00:00:00Z",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded TunnelMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Type != msg.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, msg.Type)
	}
	if decoded.ServerID != msg.ServerID {
		t.Errorf("ServerID = %q, want %q", decoded.ServerID, msg.ServerID)
	}
	if decoded.CommandID != msg.CommandID {
		t.Errorf("CommandID = %q, want %q", decoded.CommandID, msg.CommandID)
	}
}

func TestCommandResult_JSON(t *testing.T) {
	result := CommandResult{
		Output: "hello world",
		Error:  "",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded CommandResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Output != result.Output {
		t.Errorf("Output = %q, want %q", decoded.Output, result.Output)
	}
	if decoded.Error != result.Error {
		t.Errorf("Error = %q, want %q", decoded.Error, result.Error)
	}
}

func TestHandleResult_InvalidPayload(t *testing.T) {
	tm := NewTunnelManager()

	// Register a pending command
	cmdCh := make(chan CommandResult, 1)
	tm.mu.Lock()
	tm.commands["test-cmd"] = cmdCh
	tm.mu.Unlock()

	// Send result with invalid payload
	msg := TunnelMessage{
		Type:      "result",
		CommandID: "test-cmd",
		Payload:   json.RawMessage(`not valid json for CommandResult`),
	}

	tm.handleResult(msg)

	// Should still receive a result (with error message)
	select {
	case result := <-cmdCh:
		if result.Error == "" {
			t.Error("expected error in result for invalid payload")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for result")
	}
}

func TestHandleResult_NoPendingCommand(t *testing.T) {
	tm := NewTunnelManager()

	// Send result for non-existent command — should not panic
	msg := TunnelMessage{
		Type:      "result",
		CommandID: "nonexistent",
		Payload:   json.RawMessage(`{"output":"test"}`),
	}

	tm.handleResult(msg)
	// If we get here without panic, the test passes
}

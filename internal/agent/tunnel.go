package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// TunnelMessage represents a message sent through the agent tunnel.
type TunnelMessage struct {
	Type      string          `json:"type"`       // command, result, heartbeat, register
	ServerID  string          `json:"server_id"`
	CommandID string          `json:"command_id"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp string          `json:"timestamp"`
}

// CommandResult represents the result of a remote command.
type CommandResult struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

// TunnelManager manages agent WebSocket connections for reverse tunneling.
type TunnelManager struct {
	agents   map[string]*agentConn // serverID -> connection
	commands map[string]chan CommandResult // commandID -> result channel
	mu       sync.RWMutex
	upgrader websocket.Upgrader
}

type agentConn struct {
	conn     *websocket.Conn
	serverID string
	lastSeen time.Time
	writeMu  sync.Mutex // protects concurrent writes to conn
}

// NewTunnelManager creates a new TunnelManager.
func NewTunnelManager() *TunnelManager {
	return &TunnelManager{
		agents:   make(map[string]*agentConn),
		commands: make(map[string]chan CommandResult),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

// HandleTunnel handles WebSocket upgrade for agent connections.
// GET /ws/agent/:server_id
func (tm *TunnelManager) HandleTunnel(w http.ResponseWriter, r *http.Request, serverID string) {
	conn, err := tm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("agent tunnel upgrade failed", "server_id", serverID, "error", err)
		return
	}
	defer func() { _ = conn.Close() }()

	// Register the agent connection
	ac := &agentConn{
		conn:     conn,
		serverID: serverID,
		lastSeen: time.Now(),
	}
	tm.mu.Lock()
	// Close existing connection if any
	if existing, ok := tm.agents[serverID]; ok {
		_ = existing.conn.Close()
	}
	tm.agents[serverID] = ac
	tm.mu.Unlock()

	slog.Info("agent connected", "server_id", serverID)

	// Send registration acknowledgment
	ac.writeMu.Lock()
	_ = ac.conn.WriteJSON(TunnelMessage{
		Type:      "register",
		ServerID:  serverID,
		Timestamp: time.Now().Format(time.RFC3339),
	})
	ac.writeMu.Unlock()

	// Read loop
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var tunnelMsg TunnelMessage
		if err := json.Unmarshal(msg, &tunnelMsg); err != nil {
			slog.Warn("invalid message from agent", "server_id", serverID, "error", err)
			continue
		}

		// Update last seen
		tm.mu.Lock()
		if ac, ok := tm.agents[serverID]; ok {
			ac.lastSeen = time.Now()
		}
		tm.mu.Unlock()

		switch tunnelMsg.Type {
		case "result":
			tm.handleResult(tunnelMsg)
		case "heartbeat":
			// Agent heartbeat, nothing to do
		default:
			slog.Warn("unknown message type from agent", "server_id", serverID, "type", tunnelMsg.Type)
		}
	}

	// Clean up on disconnect
	tm.mu.Lock()
	if ac, ok := tm.agents[serverID]; ok && ac.conn == conn {
		delete(tm.agents, serverID)
	}
	tm.mu.Unlock()

	slog.Info("agent disconnected", "server_id", serverID)
}

// handleResult routes a command result to the waiting caller.
func (tm *TunnelManager) handleResult(msg TunnelMessage) {
	tm.mu.Lock()
	ch, ok := tm.commands[msg.CommandID]
	if ok {
		delete(tm.commands, msg.CommandID)
	}
	tm.mu.Unlock()

	if !ok {
		slog.Warn("no pending command for result", "command_id", msg.CommandID)
		return
	}

	var result CommandResult
	if err := json.Unmarshal(msg.Payload, &result); err != nil {
		result = CommandResult{Error: fmt.Sprintf("failed to unmarshal result: %v", err)}
	}

	select {
	case ch <- result:
	default:
		slog.Warn("result channel full, dropping command result", "command_id", msg.CommandID)
	}
}

// ExecuteCommand sends a command to an agent and waits for the result.
func (tm *TunnelManager) ExecuteCommand(ctx context.Context, serverID, cmd string, timeout time.Duration) (*CommandResult, error) {
	tm.mu.RLock()
	ac, ok := tm.agents[serverID]
	tm.mu.RUnlock()

	if !ok || ac == nil {
		return nil, fmt.Errorf("agent not connected: server_id=%s", serverID)
	}

	commandID := uuid.New().String()
	payload, _ := json.Marshal(map[string]string{"command": cmd})

	msg := TunnelMessage{
		Type:      "command",
		ServerID:  serverID,
		CommandID: commandID,
		Payload:   payload,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Register result channel
	resultCh := make(chan CommandResult, 1)
	tm.mu.Lock()
	tm.commands[commandID] = resultCh
	tm.mu.Unlock()

	defer func() {
		tm.mu.Lock()
		delete(tm.commands, commandID)
		tm.mu.Unlock()
	}()

	// Send command to agent
	tm.mu.RLock()
	ac.writeMu.Lock()
	writeErr := ac.conn.WriteJSON(msg)
	ac.writeMu.Unlock()
	tm.mu.RUnlock()

	if writeErr != nil {
		return nil, fmt.Errorf("failed to send command to agent: %w", writeErr)
	}

	// Wait for result with timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		return &result, nil
	case <-time.After(timeout):
		return nil, errors.New("command timed out")
	}
}

// IsConnected checks if an agent is connected.
func (tm *TunnelManager) IsConnected(serverID string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	_, ok := tm.agents[serverID]
	return ok
}

// ListAgents returns all connected agent server IDs.
func (tm *TunnelManager) ListAgents() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	ids := make([]string, 0, len(tm.agents))
	for id := range tm.agents {
		ids = append(ids, id)
	}
	return ids
}

// StartCleanup starts a background goroutine to clean up stale connections.
func (tm *TunnelManager) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			tm.cleanup()
		}
	}()
}

// cleanup removes stale agent connections.
func (tm *TunnelManager) cleanup() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	now := time.Now()
	for id, ac := range tm.agents {
		if now.Sub(ac.lastSeen) > 5*time.Minute {
			slog.Info("cleaning up stale agent connection", "server_id", id)
			if ac.conn != nil {
				_ = ac.conn.Close()
			}
			delete(tm.agents, id)
		}
	}
}

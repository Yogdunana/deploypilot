package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/metrics"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// allowedWSSOrigins contains origins permitted for WebSocket connections.
// Configured via DEPLOYPILOT_WS_ALLOWED_ORIGINS (comma-separated).
// If empty, defaults to same-origin only (request Host header).
var allowedWSSOrigins []string

func init() {
	if origins := os.Getenv("DEPLOYPILOT_WS_ALLOWED_ORIGINS"); origins != "" {
		allowedWSSOrigins = strings.Split(origins, ",")
		for i, o := range allowedWSSOrigins {
			allowedWSSOrigins[i] = strings.TrimSpace(o)
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // non-browser clients (e.g. CLI tools) don't send Origin
		}
		// If no explicit origins configured, allow same-origin requests
		if len(allowedWSSOrigins) == 0 {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			return origin == scheme+"://"+r.Host
		}
		// Check against whitelist
		for _, allowed := range allowedWSSOrigins {
			if origin == allowed {
				return true
			}
		}
		return false
	},
}

// WSMessage is the standard WebSocket message format.
type WSMessage struct {
	Type      string      `json:"type"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
	AppID     string      `json:"app_id,omitempty"`
}

// wsClient represents a single WebSocket connection tied to an app.
type wsClient struct {
	conn  *websocket.Conn
	appID string
}

// WSHub manages WebSocket connections for broadcasting.
type WSHub struct {
	clients    map[string]map[*websocket.Conn]bool
	mu         sync.RWMutex
	register   chan *wsClient
	unregister chan *wsClient
}

// NewWSHub creates a new WSHub.
func NewWSHub() *WSHub {
	return &WSHub{
		clients:    make(map[string]map[*websocket.Conn]bool),
		register:   make(chan *wsClient),
		unregister: make(chan *wsClient),
	}
}

// Run starts the background goroutine that handles register/unregister events.
func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.appID] == nil {
				h.clients[client.appID] = make(map[*websocket.Conn]bool)
			}
			h.clients[client.appID][client.conn] = true
			h.mu.Unlock()
			metrics.WSConnections.Inc()
		case client := <-h.unregister:
			h.mu.Lock()
			if conns, ok := h.clients[client.appID]; ok {
				delete(conns, client.conn)
				if len(conns) == 0 {
					delete(h.clients, client.appID)
				}
			}
			h.mu.Unlock()
			metrics.WSConnections.Dec()
		}
	}
}

// Register adds a WebSocket connection to the hub for the given appID.
func (h *WSHub) Register(conn *websocket.Conn, appID string) {
	h.register <- &wsClient{conn: conn, appID: appID}
}

// Unregister removes a WebSocket connection from the hub.
func (h *WSHub) Unregister(conn *websocket.Conn, appID string) {
	h.unregister <- &wsClient{conn: conn, appID: appID}
}

// Broadcast sends a message to all connections subscribed to the given appID.
func (h *WSHub) Broadcast(appID string, msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal broadcast message", "error", err)
		return
	}

	h.mu.RLock()
	conns := h.clients[appID]
	// Copy keys to avoid holding lock during writes
	keys := make([]*websocket.Conn, 0, len(conns))
	for c := range conns {
		keys = append(keys, c)
	}
	h.mu.RUnlock()

	for _, c := range keys {
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			slog.Warn("broadcast write error", "error", err)
		}
	}
}

// Send writes a single message to a specific connection.
func (h *WSHub) Send(conn *websocket.Conn, msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal ws message", "error", err)
		return
	}
	_ = conn.WriteMessage(websocket.TextMessage, data)
}

// ClientCount returns the number of active connections for an appID.
func (h *WSHub) ClientCount(appID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[appID])
}

// LogStreamWS handles WebSocket connections for real-time container logs.
// GET /ws/logs/:app_id?ticket=xxx
func LogStreamWS(bridge *service.Bridge, hub *WSHub, ticketStore *auth.WSTicketStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Authenticate via ticket (preferred) or JWT token (backward compat)
		userID, role, err := authenticateWS(c, ticketStore)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		_ = userID
		_ = role

		// 2. Upgrade to WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("ws upgrade failed", "error", err)
			return
		}
		defer func() { _ = conn.Close() }()

		appID := c.Param("app_id")
		hub.Register(conn, appID)
		defer hub.Unregister(conn, appID)

		// 3. Get app's container name
		var appRow map[string]interface{}
		if err := bridge.DB.Table("apps").Where("id = ?", appID).Take(&appRow).Error; err != nil {
			hub.Send(conn, WSMessage{
				Type: "error",
				Data: "app not found",
			})
			return
		}

		containerName := wsToString(appRow["container_name"])
		if containerName == "" {
			containerName = wsToString(appRow["name"])
		}

		// 4. Stream logs using periodic polling (since CommandExecutor returns string)
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		go streamContainerLogs(ctx, bridge, containerName, hub, appID)

		// 5. Read loop (handle ping/pong, close)
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		// Send periodic pings for heartbeat (30s interval)
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						return
					}
				}
			}
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}
}

// authenticateWS authenticates a WebSocket connection using a one-time ticket (preferred)
// or falls back to JWT token query param for backward compatibility.
// Returns userID, role, and an error if authentication fails.
func authenticateWS(c *gin.Context, ticketStore *auth.WSTicketStore) (string, string, error) {
	// Prefer ticket-based authentication
	ticket := c.Query("ticket")
	if ticket != "" {
		userID, role, err := ticketStore.ValidateTicket(ticket)
		if err != nil {
			return "", "", fmt.Errorf("invalid or expired websocket ticket")
		}
		return userID, role, nil
	}

	// Fallback: JWT token in query param (backward compatibility)
	token := c.Query("token")
	if token == "" {
		return "", "", fmt.Errorf("websocket ticket is required")
	}

	claims, err := auth.ParseToken(token)
	if err != nil {
		return "", "", fmt.Errorf("invalid or expired websocket ticket")
	}

	return claims.UserID, claims.Role, nil
}

// AgentTunnelWS handles WebSocket connections for agent reverse tunnels.
// GET /ws/agent/:server_id?ticket=xxx
func AgentTunnelWS(bridge *service.Bridge, ticketStore *auth.WSTicketStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Authenticate via ticket (preferred) or JWT token (backward compat)
		_, _, err := authenticateWS(c, ticketStore)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		serverID := c.Param("server_id")

		// Delegate to tunnel manager
		if bridge.TunnelManager != nil {
			bridge.TunnelManager.HandleTunnel(c.Writer, c.Request, serverID)
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "agent tunnel not configured"})
		}
	}
}

// wsToString converts an interface{} to string (local helper for ws.go).
func wsToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// streamContainerLogs periodically fetches new container logs and broadcasts them.
func streamContainerLogs(ctx context.Context, bridge *service.Bridge, containerName string, hub *WSHub, appID string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logs, err := bridge.GetContainerLogs(ctx, containerName, 50)
			if err != nil {
				hub.Broadcast(appID, WSMessage{
					Type:      "error",
					Timestamp: time.Now().Format(time.RFC3339),
					Data:      err.Error(),
					AppID:     appID,
				})
				continue
			}
			hub.Broadcast(appID, WSMessage{
				Type:      "log",
				Timestamp: time.Now().Format(time.RFC3339),
				Data:      logs,
				AppID:     appID,
			})
		}
	}
}

// TerminalWS handles WebSocket connections for SSH terminal.
// GET /ws/terminal/:server_id?ticket=xxx
func TerminalWS(bridge *service.Bridge, hub *WSHub, ticketStore *auth.WSTicketStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Authenticate via ticket (preferred) or JWT token (backward compat)
		userID, role, err := authenticateWS(c, ticketStore)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		_ = userID
		_ = role

		// 2. Upgrade to WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("terminal ws upgrade failed", "error", err)
			return
		}
		defer func() { _ = conn.Close() }()

		// 3. Get server info
		serverID := c.Param("server_id")
		var serverRow map[string]interface{}
		if err := bridge.DB.Table("servers").Where("id = ?", serverID).Take(&serverRow).Error; err != nil {
			hub.Send(conn, WSMessage{
				Type: "error",
				Data: "server not found",
			})
			return
		}

		// 4. Connect SSH using bridge's remote executor
		remoteExec, err := bridge.GetRemoteExecutorForTerminal(c.Request.Context(), serverID)
		if err != nil {
			hub.Send(conn, WSMessage{
				Type: "error",
				Data: "SSH connection failed: " + err.Error(),
			})
			return
		}
		defer func() { _ = remoteExec.Close() }()

		// 5. Read loop: receive commands from client
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			var cmdMsg WSMessage
			if err := json.Unmarshal(msg, &cmdMsg); err != nil {
				continue
			}
			if cmdMsg.Type == "input" {
				cmd, ok := cmdMsg.Data.(string)
				if !ok {
					continue
				}
				output, err := remoteExec.RunCommand(c.Request.Context(), cmd)
				if err != nil {
					hub.Send(conn, WSMessage{
						Type: "output",
						Data: output + "\n" + err.Error(),
					})
				} else {
					hub.Send(conn, WSMessage{
						Type: "output",
						Data: output,
					})
				}
			}
		}
	}
}

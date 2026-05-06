package api

import (
	"context"
	"encoding/base64"
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
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
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
	Type           string      `json:"type"`
	Timestamp      string      `json:"timestamp"`
	Data           interface{} `json:"data"`
	AppID          string      `json:"app_id,omitempty"`
	SourceInstance string      `json:"source_instance,omitempty"`
}

// wsClient represents a single WebSocket connection tied to an app.
type wsClient struct {
	conn  *websocket.Conn
	appID string
}

// WSHub manages WebSocket connections for broadcasting log/deployment events.
type WSHub struct {
	clients        map[string]map[*websocket.Conn]bool
	mu             sync.RWMutex
	register       chan *wsClient
	unregister     chan *wsClient
	done           chan struct{} // signals the Run loop to stop
	closeOnce      sync.Once     // ensures Close() is safe to call multiple times
	runDone        chan struct{} // closed when Run() goroutine exits
	rdb            *redis.Client
	instanceID     string
	redisSub       *redis.PubSub
	redisDone      chan struct{}
}

// NewWSHub creates a new WSHub. If rdb is non-nil, Redis Pub/Sub is enabled
// for multi-instance broadcast.
func NewWSHub(rdb *redis.Client) *WSHub {
	instanceID := uuid.New().String()[:8]
	hub := &WSHub{
		clients:    make(map[string]map[*websocket.Conn]bool),
		register:   make(chan *wsClient),
		unregister: make(chan *wsClient),
		done:       make(chan struct{}),
		runDone:    make(chan struct{}),
		rdb:        rdb,
		instanceID: instanceID,
	}
	if rdb != nil {
		hub.redisDone = make(chan struct{})
	}
	return hub
}

// Run starts the background goroutine that handles register/unregister events.
// If Redis is configured, it also starts a Redis Pub/Sub subscriber for
// cross-instance message relay.
func (h *WSHub) Run() {
	defer close(h.runDone)

	// Start Redis subscriber if Redis is available
	if h.rdb != nil {
		go h.runRedisSubscriber()
	}

	for {
		select {
		case <-h.done:
			return
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

// Close gracefully shuts down the WebSocket hub.
// It stops the Run loop, closes the Redis subscription, and closes all active WebSocket connections.
// Safe to call multiple times.
func (h *WSHub) Close() {
	h.closeOnce.Do(func() {
		close(h.done) // signal Run() to exit
		// Signal Redis subscriber to stop
		if h.redisDone != nil {
			close(h.redisDone)
		}
	})

	<-h.runDone // wait for Run() goroutine to exit

	// Close Redis Pub/Sub subscription
	if h.redisSub != nil {
		_ = h.redisSub.Close()
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Close all active WebSocket connections
	for appID, conns := range h.clients {
		for conn := range conns {
			// Use WriteControl to send a close frame
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server shutting down"),
				time.Now().Add(time.Second))
			_ = conn.Close()
		}
		delete(h.clients, appID)
	}
	slog.Info("WebSocket hub closed")
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
// If Redis is configured, the message is also published to the Redis Pub/Sub
// channel so other instances can relay it to their local clients.
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

	// Redis broadcast for cross-instance delivery
	if h.rdb != nil {
		msg.SourceInstance = h.instanceID
		redisData, err := json.Marshal(msg)
		if err != nil {
			slog.Error("failed to marshal redis broadcast message", "error", err)
			return
		}
		h.rdb.Publish(context.Background(), "deploypilot:ws:broadcast", redisData)
	}
}

// runRedisSubscriber subscribes to the Redis Pub/Sub channel and relays
// messages from other instances to local WebSocket clients.
// Messages originating from this instance (same SourceInstance) are skipped
// to prevent echo.
func (h *WSHub) runRedisSubscriber() {
	const channel = "deploypilot:ws:broadcast"
	ctx := context.Background()
	sub := h.rdb.Subscribe(ctx, channel)
	h.redisSub = sub

	slog.Info("WebSocket Redis subscriber started", "channel", channel, "instance", h.instanceID)

	defer func() {
		_ = sub.Close()
		slog.Info("WebSocket Redis subscriber stopped", "instance", h.instanceID)
	}()

	for {
		select {
		case <-h.redisDone:
			return
		case msg, ok := <-sub.Channel():
			if !ok {
				return
			}
			var wsMsg WSMessage
			if err := json.Unmarshal([]byte(msg.Payload), &wsMsg); err != nil {
				slog.Warn("failed to unmarshal redis ws message", "error", err)
				continue
			}
			// Skip messages from this instance to avoid echo
			if wsMsg.SourceInstance == h.instanceID {
				continue
			}
			// Relay to local clients
			appID := wsMsg.AppID
			data, err := json.Marshal(wsMsg)
			if err != nil {
				slog.Warn("failed to marshal relay ws message", "error", err)
				continue
			}
			h.mu.RLock()
			conns := h.clients[appID]
			keys := make([]*websocket.Conn, 0, len(conns))
			for c := range conns {
				keys = append(keys, c)
			}
			h.mu.RUnlock()
			for _, c := range keys {
				if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
					slog.Warn("redis relay write error", "error", err)
				}
			}
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

// authorizeAppAccessCached checks if the user has permission to access the given app using cache.
func authorizeAppAccessCached(bridge *service.Bridge, ctx context.Context, appID, role, userID string) bool {
	return bridge.CheckResourceAccessCached(ctx, "app", appID, role, userID)
}

// authorizeServerAccessCached checks if the user has permission to access the given server using cache.
func authorizeServerAccessCached(bridge *service.Bridge, ctx context.Context, serverID, role, userID string) bool {
	return bridge.CheckResourceAccessCached(ctx, "server", serverID, role, userID)
}

// LogStreamWS handles WebSocket connections for real-time container logs.
// GET /ws/logs/:app_id?ticket=xxx
func LogStreamWS(bridge *service.Bridge, hub *WSHub, ticketStore *auth.WSTicketStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Authenticate via ticket (preferred) or JWT token (backward compat)
		userID, role, err := authenticateWS(c, ticketStore)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": err.Error()})
			return
		}
		// 2. Check resource-level authorization
		appID := c.Param("app_id")
		if !authorizeAppAccessCached(bridge, c.Request.Context(), appID, role, userID) {
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "no permission to access this app"})
			return
		}

		// 3. Upgrade to WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("ws upgrade failed", "error", err)
			return
		}
		defer func() { _ = conn.Close() }()
		conn.SetReadLimit(64 * 1024)

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
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": err.Error()})
			return
		}

		serverID := c.Param("server_id")

		// Delegate to tunnel manager
		if bridge.TunnelManager != nil {
			bridge.TunnelManager.HandleTunnel(c.Writer, c.Request, serverID)
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "agent tunnel not configured"})
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

// TerminalWS handles WebSocket connections for interactive SSH terminal with PTY.
// GET /ws/terminal/:server_id?ticket=xxx
//
// Protocol:
//   Client → Server:
//     {"type":"input","data":"<raw keystrokes>"}   — keystrokes to stdin
//     {"type":"resize","data":{"rows":N,"cols":M}}  — terminal resize
//   Server → Client:
//     {"type":"output","data":"<base64>"}           — stdout/stderr (base64 encoded)
//     {"type":"connected"}                           — session established
//     {"type":"disconnected","data":"<reason>"}      — session ended
//     {"type":"error","data":"<message>"}            — error occurred
func TerminalWS(bridge *service.Bridge, hub *WSHub, ticketStore *auth.WSTicketStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Authenticate via ticket (preferred) or JWT token (backward compat)
		userID, role, err := authenticateWS(c, ticketStore)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": err.Error()})
			return
		}
		// 2. Check resource-level authorization
		serverID := c.Param("server_id")
		if !authorizeServerAccessCached(bridge, c.Request.Context(), serverID, role, userID) {
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "no permission to access this server"})
			return
		}

		// 3. Upgrade to WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("terminal ws upgrade failed", "error", err)
			return
		}
		defer func() { _ = conn.Close() }()
		conn.SetReadLimit(64 * 1024)

		// 4. Get remote executor
		remoteExec, err := bridge.GetRemoteExecutorForTerminal(c.Request.Context(), serverID)
		if err != nil {
			_ = conn.WriteJSON(WSMessage{Type: "error", Data: "SSH connection failed: " + err.Error()})
			return
		}
		defer func() { _ = remoteExec.Close() }()

		// 5. Create interactive PTY session (default 24x80)
		session, err := remoteExec.CreateInteractiveSession(c.Request.Context(), "xterm-256color", 24, 80)
		if err != nil {
			_ = conn.WriteJSON(WSMessage{Type: "error", Data: "Failed to create session: " + err.Error()})
			return
		}
		defer func() { _ = session.Close() }()

		// 6. Notify client that session is ready
		_ = conn.WriteJSON(WSMessage{Type: "connected"})

		// 7. Read goroutine: WebSocket → SSH stdin
		go func() {
			defer func() { _ = session.Close() }()
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return // WebSocket closed
				}
				var wsMsg WSMessage
				if err := json.Unmarshal(msg, &wsMsg); err != nil {
					continue
				}
				switch wsMsg.Type {
				case "input":
					if data, ok := wsMsg.Data.(string); ok {
						_, _ = session.StdinPipe().Write([]byte(data))
					}
				case "resize":
					if m, ok := wsMsg.Data.(map[string]interface{}); ok {
						rows := int(m["rows"].(float64))
						cols := int(m["cols"].(float64))
						if rows < 1 || rows > 500 || cols < 1 || cols > 500 {
							continue
						}
						_ = session.SetWindowSize(rows, cols)
					}
				}
			}
		}()

		// 8. Write goroutine: SSH stdout → WebSocket (base64 encoded)
		outputCh := session.Output()
		doneCh := session.Done()
		for {
			select {
			case data, ok := <-outputCh:
				if !ok {
					_ = conn.WriteJSON(WSMessage{Type: "disconnected", Data: "session closed"})
					return
				}
				// Base64 encode binary data for safe JSON transport
				encoded := base64.StdEncoding.EncodeToString(data)
				if err := conn.WriteJSON(WSMessage{Type: "output", Data: encoded}); err != nil {
					return
				}
			case <-doneCh:
				_ = conn.WriteJSON(WSMessage{Type: "disconnected", Data: "session exited"})
				return
			}
		}
	}
}

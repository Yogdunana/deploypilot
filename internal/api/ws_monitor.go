package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// WSMonitorHub manages WebSocket connections for real-time monitoring dashboard updates.
type WSMonitorHub struct {
	clients   map[*websocket.Conn]bool
	mu        sync.RWMutex
	broadcast chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	done       chan struct{}
	closeOnce  sync.Once
}

// NewWSMonitorHub creates a new WSMonitorHub.
func NewWSMonitorHub() *WSMonitorHub {
	return &WSMonitorHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:   make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		done:       make(chan struct{}),
	}
}

// Run starts the hub's event loop.
func (h *WSMonitorHub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
		slog.Debug("monitor WS client connected", "total", h.ClientCount())

	case conn := <-h.unregister:
			h.mu.Lock()
		if _, ok := h.clients[conn]; ok {
			delete(h.clients, conn)
			_ = conn.Close()
		}
			h.mu.Unlock()
		slog.Debug("monitor WS client disconnected", "total", h.ClientCount())

	case message := <-h.broadcast:
		h.mu.RLock()
		for conn := range h.clients {
			_ = conn.WriteMessage(websocket.TextMessage, message)
		}
		h.mu.RUnlock()

	case <-h.done:
		return
		}
	}
}

// Close gracefully shuts down the hub.
func (h *WSMonitorHub) Close() {
	h.closeOnce.Do(func() {
		close(h.done)
		h.mu.Lock()
		for conn := range h.clients {
			_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
			_ = conn.Close()
		}
		h.clients = make(map[*websocket.Conn]bool)
		h.mu.Unlock()
	})
}

// Register adds a new WebSocket connection.
func (h *WSMonitorHub) Register(conn *websocket.Conn) {
	h.register <- conn
}

// Unregister removes a WebSocket connection.
func (h *WSMonitorHub) Unregister(conn *websocket.Conn) {
	h.unregister <- conn
}

// Broadcast sends a message to all connected clients.
func (h *WSMonitorHub) Broadcast(data interface{}) {
	msg, err := json.Marshal(data)
	if err != nil {
		slog.Warn("failed to marshal monitor broadcast", "error", err)
		return
	}
	select {
	case h.broadcast <- msg:
	default:
		slog.Warn("monitor broadcast channel full, dropping message")
	}
}

// ClientCount returns the number of connected clients.
func (h *WSMonitorHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// MonitorWS handles WebSocket connections for the monitoring dashboard.
func (m *MonitorAPI) MonitorWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("monitor WS upgrade failed", "error", err)
		return
	}

	if m.monitorHub != nil {
		m.monitorHub.Register(conn)

		defer func() {
			m.monitorHub.Unregister(conn)
		}()

		// Read loop to detect disconnect
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	} else {
		_ = conn.Close()
	}
}

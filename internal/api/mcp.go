package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/server"
	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/service"
)

// MCPHTTPHandler handles MCP requests over HTTP/SSE.
// It allows cloud-based AI IDEs (TRAE Solo, Coze, Cursor Cloud) to connect
// to DeployPilot via the network instead of stdio.
type MCPHTTPHandler struct {
	mu          sync.RWMutex
	sessions    map[string]*mcpSession
	mcpServer   *server.MCPServer
	bridge      *service.Bridge
}

type mcpSession struct {
	id       string
	role     string
	userID   string
	username string
}

// NewMCPHTTPHandler creates a new MCP HTTP handler.
func NewMCPHTTPHandler(bridge *service.Bridge) *MCPHTTPHandler {
	mcpServer := mcp.NewServer(bridge)
	return &MCPHTTPHandler{
		sessions:  make(map[string]*mcpSession),
		mcpServer: mcpServer,
		bridge:    bridge,
	}
}

// HandleSSE handles SSE connections for MCP clients.
// GET /api/v1/mcp/sse
func (h *MCPHTTPHandler) HandleSSE(c *gin.Context) {
	// Get user info from context (set by auth middleware)
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	// Generate session ID
	sessionID := uuid.New().String()

	// Store session
	h.mu.Lock()
	h.sessions[sessionID] = &mcpSession{
		id:       sessionID,
		role:     role.(string),
		userID:   userID.(string),
		username: username.(string),
	}
	h.mu.Unlock()

	// Cleanup on disconnect
	defer func() {
		h.mu.Lock()
		delete(h.sessions, sessionID)
		h.mu.Unlock()
	}()

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Mcp-Session-Id", sessionID)

	// Send endpoint event
	c.SSEvent("endpoint", gin.H{
		"message_endpoint": "/api/v1/mcp/message?session=" + sessionID,
	})
	c.Writer.Flush()

	// Keep connection alive
	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			slog.Debug("MCP SSE connection closed", "session", sessionID)
			return
		case <-c.Done():
			return
		}
	}
}

// HandleMessage handles MCP JSON-RPC messages from clients.
// POST /api/v1/mcp/message
func (h *MCPHTTPHandler) HandleMessage(c *gin.Context) {
	sessionID := c.Query("session")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing session parameter"})
		return
	}

	// Get session
	h.mu.RLock()
	session, ok := h.sessions[sessionID]
	h.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Read request body
	var rawMessage json.RawMessage
	if err := c.BindJSON(&rawMessage); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// Create context with role for permission checking
	ctx := mcp.ContextWithRole(c.Request.Context(), session.role)

	// Handle message through MCP server
	response := h.mcpServer.HandleMessage(ctx, rawMessage)

	// Return response
	if response != nil {
		c.Data(http.StatusOK, "application/json", mustMarshal(response))
	} else {
		c.Status(http.StatusNoContent)
	}
}

// HandleMessageDirect handles direct JSON-RPC messages (non-SSE mode).
// POST /api/v1/mcp
// This is for clients that prefer stateless HTTP instead of SSE.
func (h *MCPHTTPHandler) HandleMessageDirect(c *gin.Context) {
	// Get user info from context (set by auth middleware)
	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Read request body
	var rawMessage json.RawMessage
	if err := c.BindJSON(&rawMessage); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// Create context with role for permission checking
	ctx := mcp.ContextWithRole(c.Request.Context(), role.(string))

	// Handle message through MCP server
	response := h.mcpServer.HandleMessage(ctx, rawMessage)

	// Return response
	if response != nil {
		c.Data(http.StatusOK, "application/json", mustMarshal(response))
	} else {
		c.Status(http.StatusNoContent)
	}
}

// HandleListTools returns the list of available MCP tools.
// GET /api/v1/mcp/tools
func (h *MCPHTTPHandler) HandleListTools(c *gin.Context) {
	// Send a tools/list request to get available tools
	listRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}
	rawMessage, _ := json.Marshal(listRequest)

	ctx := mcp.ContextWithRole(c.Request.Context(), "owner")
	response := h.mcpServer.HandleMessage(ctx, rawMessage)

	if response != nil {
		c.Data(http.StatusOK, "application/json", mustMarshal(response))
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tools"})
	}
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		slog.Error("failed to marshal MCP response", "error", err)
		return []byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"Internal error"}}`)
	}
	return data
}

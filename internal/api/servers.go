package api

import (
	"net/http"
	"strconv"

	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// AddServer registers a new server.
func AddServer(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Name string `json:"name" binding:"required"`
			Host string `json:"host" binding:"required"`
			Port int    `json:"port"`
			User string `json:"user"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}
		if input.Port == 0 {
			input.Port = 22
		}
		if input.User == "" {
			input.User = "root"
		}

		srv, err := bridge.AddServer(c.Request.Context(), input.Name, input.Host, input.Port, input.User)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, srv)
	}
}

// ListServers lists all registered servers.
func ListServers(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		servers, err := bridge.ListServers(c.Request.Context())
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if servers == nil {
			servers = []mcp.ServerInfo{}
		}
		respondSuccess(c, servers)
	}
}

// UpdateServer updates a server's configuration.
func UpdateServer(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var config map[string]interface{}
		if err := c.ShouldBindJSON(&config); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		result, err := bridge.UpdateServer(c.Request.Context(), id, config)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// DeleteServer removes a server.
func DeleteServer(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := bridge.RemoveServer(c.Request.Context(), id); err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, gin.H{"message": "server deleted", "id": id})
	}
}

// DetectEnvironment detects the server environment.
func DetectEnvironment(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		level := 2
		if l := c.Query("level"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil {
				level = parsed
			}
		}

		var ports []int
		if p := c.Query("ports"); p != "" {
			for _, ps := range splitAndTrim(p) {
				if port, err := strconv.Atoi(ps); err == nil {
					ports = append(ports, port)
				}
			}
		}

		var services []string
		if s := c.Query("services"); s != "" {
			services = splitAndTrim(s)
		}

		env, err := bridge.DetectEnv(c.Request.Context(), level, ports, services)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, env)
	}
}

// GetServerEnvironment returns the detected environment info for a server.
func GetServerEnvironment(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		// Look up the server's detected_info field
		var row map[string]interface{}
		if err := bridge.DB.Table("servers").Where("id = ?", id).Take(&row).Error; err != nil {
			respondError(c, http.StatusNotFound, "server not found")
			return
		}
		respondSuccess(c, row)
	}
}

// TestServer tests connectivity to a server.
func TestServer(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result, err := bridge.TestServer(c.Request.Context(), id)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

package api

import (
	"net/http"
	"strconv"

	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// AddServer registers a new server.
// @Summary      Add a server
// @Description  Register a new deployment server with SSH connection details
// @Tags         Servers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{name=string,host=string,port=int,user=string} true "Server registration request"
// @Success      200 {object} map[string]interface{} "status, data (ServerInfo)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /servers [post]
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
// @Summary      List all servers
// @Description  Retrieve a list of all registered deployment servers
// @Tags         Servers
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data (array of ServerInfo)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /servers [get]
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
// @Summary      Update a server
// @Description  Update configuration fields of a registered server
// @Tags         Servers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Server ID"
// @Param        request body map[string]interface{} true "Fields to update"
// @Success      200 {object} map[string]interface{} "status, data (updated ServerInfo)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /servers/{id} [put]
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
// @Summary      Delete a server
// @Description  Remove a registered server by ID
// @Tags         Servers
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Server ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /servers/{id} [delete]
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
// @Summary      Detect server environment
// @Description  Run environment detection on a server to identify installed services, open ports, and system info
// @Tags         Servers
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Server ID"
// @Param        level query int false "Detection depth level" default(2)
// @Param        ports query string false "Comma-separated list of ports to check"
// @Param        services query string false "Comma-separated list of services to check"
// @Success      200 {object} map[string]interface{} "status, data (EnvironmentInfo)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /servers/{id}/detect [post]
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
// @Summary      Get server environment info
// @Description  Retrieve previously detected environment information for a server
// @Tags         Servers
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Server ID"
// @Success      200 {object} map[string]interface{} "status, data (server row with detected_info)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "server not found"
// @Router       /servers/{id}/environment [get]
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
// @Summary      Test server connectivity
// @Description  Test SSH connectivity to a registered server
// @Tags         Servers
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Server ID"
// @Success      200 {object} map[string]interface{} "status, data (test result)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /servers/{id}/test [post]
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

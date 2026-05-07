package api

import (
	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/sandbox"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// registerServerRoutes registers all server-related routes.
func registerServerRoutes(protected *gin.RouterGroup, db *gorm.DB, bridge *service.Bridge) {
	servers := protected.Group("/servers")
	{
		servers.POST("", AddServer(bridge))
		servers.GET("", ListServers(bridge))
		servers.PUT("/:id", auth.RequireResourceAccessCached(bridge, "server", "id"), UpdateServer(bridge))
		servers.DELETE("/:id", auth.RequireResourceAccessCached(bridge, "server", "id"), DeleteServer(bridge))
		servers.POST("/:id/detect", auth.RequireResourceAccessCached(bridge, "server", "id"), DetectEnvironment(bridge))
		servers.GET("/:id/environment", auth.RequireResourceAccessCached(bridge, "server", "id"), GetServerEnvironment(bridge))
		servers.POST("/:id/test", auth.RequireResourceAccessCached(bridge, "server", "id"), TestServer(bridge))

		// File management — requires server resource access
		fileAPI := NewFileManagerAPI(db, sandbox.New(sandbox.DefaultConfig()))
		servers.GET("/:id/files", auth.RequireResourceAccessCached(bridge, "server", "id"), fileAPI.ListFiles)
		servers.GET("/:id/files/read", auth.RequireResourceAccessCached(bridge, "server", "id"), fileAPI.ReadFile)
		servers.PUT("/:id/files/write", auth.RequireResourceAccessCached(bridge, "server", "id"), fileAPI.WriteFile)
		servers.DELETE("/:id/files", auth.RequireResourceAccessCached(bridge, "server", "id"), fileAPI.DeleteFile)
		servers.POST("/:id/files/mkdir", auth.RequireResourceAccessCached(bridge, "server", "id"), fileAPI.CreateDirectory)
		servers.POST("/:id/files/move", auth.RequireResourceAccessCached(bridge, "server", "id"), fileAPI.MoveFile)
		servers.GET("/:id/files/disk-usage", auth.RequireResourceAccessCached(bridge, "server", "id"), fileAPI.GetDiskUsage)
		servers.GET("/:id/files/info", auth.RequireResourceAccessCached(bridge, "server", "id"), fileAPI.GetFileInfo)
		servers.GET("/:id/files/search", auth.RequireResourceAccessCached(bridge, "server", "id"), fileAPI.SearchFiles)

		// Firewall management — requires server resource access
		fwAPI := NewFirewallAPI(db, sandbox.New(sandbox.DefaultConfig()))
		servers.GET("/:id/firewall", auth.RequireResourceAccessCached(bridge, "server", "id"), fwAPI.GetFirewallStatus)
		servers.GET("/:id/firewall/detect", auth.RequireResourceAccessCached(bridge, "server", "id"), fwAPI.DetectFirewall)
		servers.POST("/:id/firewall/ports/open", auth.RequireResourceAccessCached(bridge, "server", "id"), fwAPI.OpenPort)
		servers.POST("/:id/firewall/ports/close", auth.RequireResourceAccessCached(bridge, "server", "id"), fwAPI.ClosePort)
		servers.POST("/:id/firewall/blocks", auth.RequireResourceAccessCached(bridge, "server", "id"), fwAPI.BlockIP)
		servers.DELETE("/:id/firewall/blocks/:ip", auth.RequireResourceAccessCached(bridge, "server", "id"), fwAPI.UnblockIP)
		servers.POST("/:id/firewall/common-ports", auth.RequireResourceAccessCached(bridge, "server", "id"), fwAPI.AllowCommonPorts)

		// SSH management — requires server resource access
		servers.GET("/:id/ssh/authorizations", auth.RequireResourceAccessCached(bridge, "server", "id"), NewSSHAPI(db).ListServerAuthorizations)
	}

	// SSH key management (top-level)
	sshAPI := NewSSHAPI(db)
	sshGroup := protected.Group("/ssh")
	{
		sshGroup.POST("/keys/generate", sshAPI.GenerateKeyPair)
		sshGroup.POST("/keys/import", sshAPI.ImportPublicKey)
		sshGroup.GET("/keys", sshAPI.ListKeyPairs)
		sshGroup.GET("/keys/:id", sshAPI.GetKeyPair)
		sshGroup.DELETE("/keys/:id", sshAPI.DeleteKeyPair)
		sshGroup.GET("/keys/:id/authorizations", sshAPI.ListKeyAuthorizations)
		sshGroup.POST("/authorize", sshAPI.AuthorizeKey)
		sshGroup.POST("/revoke", sshAPI.RevokeKey)
	}

	// Process daemon management — requires server resource access
	procAPI := NewProcessAPI(db)
	servers.GET("/:id/processes", auth.RequireResourceAccessCached(bridge, "server", "id"), procAPI.ListProcesses)
	servers.GET("/:id/processes/tree", auth.RequireResourceAccessCached(bridge, "server", "id"), procAPI.GetProcessTree)
	servers.GET("/:id/processes/search", auth.RequireResourceAccessCached(bridge, "server", "id"), procAPI.SearchProcesses)
	servers.GET("/:id/processes/:pid", auth.RequireResourceAccessCached(bridge, "server", "id"), procAPI.GetProcess)
	servers.POST("/:id/processes/:pid/kill", auth.RequireResourceAccessCached(bridge, "server", "id"), procAPI.KillProcess)
	servers.GET("/:id/resources", auth.RequireResourceAccessCached(bridge, "server", "id"), procAPI.GetSystemResources)

	procGroup := protected.Group("/processes")
	{
		procGroup.GET("/rules", procAPI.ListRules)
		procGroup.GET("/rules/:id", procAPI.GetRule)
		procGroup.POST("/rules", procAPI.CreateRule)
		procGroup.PUT("/rules/:id", procAPI.UpdateRule)
		procGroup.DELETE("/rules/:id", procAPI.DeleteRule)
	}

	// System snapshot management — requires server resource access
	snapAPI := NewSnapshotAPI(db)
	servers.GET("/:id/snapshots", auth.RequireResourceAccessCached(bridge, "server", "id"), snapAPI.ListSnapshots)
	servers.GET("/:id/snapshots/files", auth.RequireResourceAccessCached(bridge, "server", "id"), snapAPI.GetSnapshotFiles)
	servers.GET("/:id/snapshots/diff", auth.RequireResourceAccessCached(bridge, "server", "id"), snapAPI.DiffSnapshots)
	servers.GET("/:id/snapshots/:snap_id", auth.RequireResourceAccessCached(bridge, "server", "id"), snapAPI.GetSnapshot)
	servers.POST("/:id/snapshots", auth.RequireResourceAccessCached(bridge, "server", "id"), snapAPI.CreateSnapshot)
	servers.POST("/:id/snapshots/:snap_id/restore", auth.RequireResourceAccessCached(bridge, "server", "id"), snapAPI.RestoreSnapshot)
	servers.DELETE("/:id/snapshots/:snap_id", auth.RequireResourceAccessCached(bridge, "server", "id"), snapAPI.DeleteSnapshot)

	// Toolbox management — requires server resource access
	tbAPI := NewToolboxAPI(db)
	servers.GET("/:id/toolbox/detect", auth.RequireResourceAccessCached(bridge, "server", "id"), tbAPI.DetectEnvironment)
	servers.POST("/:id/toolbox/run", auth.RequireResourceAccessCached(bridge, "server", "id"), tbAPI.RunScript)
	servers.POST("/:id/toolbox/builtin", auth.RequireResourceAccessCached(bridge, "server", "id"), tbAPI.RunBuiltInScript)
	servers.GET("/:id/toolbox/builtin-scripts", auth.RequireResourceAccessCached(bridge, "server", "id"), tbAPI.ListBuiltInScripts)
	servers.GET("/:id/toolbox/scripts", auth.RequireResourceAccessCached(bridge, "server", "id"), tbAPI.ListScripts)
	servers.POST("/:id/toolbox/scripts", auth.RequireResourceAccessCached(bridge, "server", "id"), tbAPI.CreateScript)
	servers.GET("/:id/toolbox/scripts/:script_id", auth.RequireResourceAccessCached(bridge, "server", "id"), tbAPI.GetScript)
	servers.PUT("/:id/toolbox/scripts/:script_id", auth.RequireResourceAccessCached(bridge, "server", "id"), tbAPI.UpdateScript)
	servers.DELETE("/:id/toolbox/scripts/:script_id", auth.RequireResourceAccessCached(bridge, "server", "id"), tbAPI.DeleteScript)

}

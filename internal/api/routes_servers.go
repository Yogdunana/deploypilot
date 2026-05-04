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

			// File management
			fileAPI := NewFileManagerAPI(db, sandbox.New(sandbox.DefaultConfig()))
			servers.GET("/:id/files", fileAPI.ListFiles)
			servers.GET("/:id/files/read", fileAPI.ReadFile)
			servers.PUT("/:id/files/write", fileAPI.WriteFile)
			servers.DELETE("/:id/files", fileAPI.DeleteFile)
			servers.POST("/:id/files/mkdir", fileAPI.CreateDirectory)
			servers.POST("/:id/files/move", fileAPI.MoveFile)
			servers.GET("/:id/files/disk-usage", fileAPI.GetDiskUsage)
			servers.GET("/:id/files/info", fileAPI.GetFileInfo)
			servers.GET("/:id/files/search", fileAPI.SearchFiles)

			// Firewall management
			fwAPI := NewFirewallAPI(db, sandbox.New(sandbox.DefaultConfig()))
			servers.GET("/:id/firewall", fwAPI.GetFirewallStatus)
			servers.GET("/:id/firewall/detect", fwAPI.DetectFirewall)
			servers.POST("/:id/firewall/ports/open", fwAPI.OpenPort)
			servers.POST("/:id/firewall/ports/close", fwAPI.ClosePort)
			servers.POST("/:id/firewall/blocks", fwAPI.BlockIP)
			servers.DELETE("/:id/firewall/blocks/:ip", fwAPI.UnblockIP)
			servers.POST("/:id/firewall/common-ports", fwAPI.AllowCommonPorts)

			// SSH management
			servers.GET("/:id/ssh/authorizations", NewSSHAPI(db).ListServerAuthorizations)
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

		// Process daemon management
		procAPI := NewProcessAPI(db)
		servers.GET("/:id/processes", procAPI.ListProcesses)
		servers.GET("/:id/processes/tree", procAPI.GetProcessTree)
		servers.GET("/:id/processes/search", procAPI.SearchProcesses)
		servers.GET("/:id/processes/:pid", procAPI.GetProcess)
		servers.POST("/:id/processes/:pid/kill", procAPI.KillProcess)
		servers.GET("/:id/resources", procAPI.GetSystemResources)

		procGroup := protected.Group("/processes")
		{
			procGroup.GET("/rules", procAPI.ListRules)
			procGroup.GET("/rules/:id", procAPI.GetRule)
			procGroup.POST("/rules", procAPI.CreateRule)
			procGroup.PUT("/rules/:id", procAPI.UpdateRule)
			procGroup.DELETE("/rules/:id", procAPI.DeleteRule)
		}

		// System snapshot management
		snapAPI := NewSnapshotAPI(db)
		servers.GET("/:id/snapshots", snapAPI.ListSnapshots)
		servers.GET("/:id/snapshots/files", snapAPI.GetSnapshotFiles)
		servers.GET("/:id/snapshots/diff", snapAPI.DiffSnapshots)
		servers.GET("/:id/snapshots/:snap_id", snapAPI.GetSnapshot)
		servers.POST("/:id/snapshots", snapAPI.CreateSnapshot)
		servers.POST("/:id/snapshots/:snap_id/restore", snapAPI.RestoreSnapshot)
		servers.DELETE("/:id/snapshots/:snap_id", snapAPI.DeleteSnapshot)

		// Toolbox management
		tbAPI := NewToolboxAPI(db)
		servers.GET("/:id/toolbox/detect", tbAPI.DetectEnvironment)
		servers.POST("/:id/toolbox/run", tbAPI.RunScript)
		servers.POST("/:id/toolbox/builtin", tbAPI.RunBuiltInScript)

}

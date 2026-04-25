package api

import (
	"net/http"
	"runtime"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetVersion returns the system version information.
// @Summary      Get system version
// @Description  Retrieve the current system version, Go runtime info, and CPU count
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data.version, data.go, data.goos, data.goarch, data.num_cpu"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Router       /system/version [get]
func GetVersion(c *gin.Context) {
	respondSuccess(c, gin.H{
		"version":  "0.6.0",
		"go":       runtime.Version(),
		"goos":     runtime.GOOS,
		"goarch":   runtime.GOARCH,
		"num_cpu":  runtime.NumCPU(),
	})
}

// CheckUpdate checks for system updates.
// @Summary      Check for system updates
// @Description  Check if a newer version of Deploypilot is available
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data (update check result)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /system/update/check [get]
func CheckUpdate(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := bridge.CheckSystemUpdate(c.Request.Context())
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// SystemHealth checks the health of system components.
// @Summary      System health check
// @Description  Check the health status of system components including database connectivity
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data.status, data.database.status"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      503 {object} map[string]interface{} "service unavailable"
// @Router       /system/health [get]
func SystemHealth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		health := gin.H{"status": "healthy"}

		// Check database
		sqlDB, err := db.DB()
		if err != nil {
			health["status"] = "unhealthy"
			health["database"] = gin.H{"status": "error", "message": err.Error()}
		} else if err := sqlDB.Ping(); err != nil {
			health["status"] = "unhealthy"
			health["database"] = gin.H{"status": "error", "message": err.Error()}
		} else {
			health["database"] = gin.H{"status": "ok"}
		}

		code := http.StatusOK
		if health["status"] != "healthy" {
			code = http.StatusServiceUnavailable
		}
		c.JSON(code, gin.H{"status": "success", "data": health})
	}
}

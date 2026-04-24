package api

import (
	"net/http"
	"runtime"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetVersion returns the system version information.
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

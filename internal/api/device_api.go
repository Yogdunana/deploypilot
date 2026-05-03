package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// globalDeviceAPI is the package-level DeviceAPI instance.
var globalDeviceAPI *DeviceAPI

// DeviceAPI handles device binding HTTP endpoints.
type DeviceAPI struct {
	svc *service.DeviceService
}

// NewDeviceAPI creates a new DeviceAPI.
func NewDeviceAPI(svc *service.DeviceService) *DeviceAPI {
	return &DeviceAPI{svc: svc}
}

// SetDeviceAPI sets the global DeviceAPI instance.
func SetDeviceAPI(api *DeviceAPI) {
	globalDeviceAPI = api
}

// GetGlobalDeviceAPI returns the global DeviceAPI instance.
func GetGlobalDeviceAPI() *DeviceAPI {
	return globalDeviceAPI
}

// ListDevices returns all devices for the authenticated user.
// GET /api/v1/devices
func ListDevices(c *gin.Context) {
	if globalDeviceAPI == nil {
		respondError(c, http.StatusInternalServerError, "device service not initialized")
		return
	}

	userID, exists := c.Get(string(auth.UserIDKey))
	if !exists {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	devices, err := globalDeviceAPI.svc.ListDevices(userID.(string))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to list devices")
		return
	}

	respondSuccess(c, devices)
}

// RevokeDevice removes trust from a device owned by the authenticated user.
// DELETE /api/v1/devices/:id
func RevokeDevice(c *gin.Context) {
	if globalDeviceAPI == nil {
		respondError(c, http.StatusInternalServerError, "device service not initialized")
		return
	}

	userID, exists := c.Get(string(auth.UserIDKey))
	if !exists {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	id := c.Param("id")
	if id == "" {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", "id is required")
		return
	}

	if err := globalDeviceAPI.svc.RevokeDevice(id, userID.(string)); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	respondSuccess(c, gin.H{"revoked": true})
}

// TrustDevice marks a device as trusted for N days.
// POST /api/v1/devices/:id/trust
func TrustDevice(c *gin.Context) {
	if globalDeviceAPI == nil {
		respondError(c, http.StatusInternalServerError, "device service not initialized")
		return
	}

	userID, exists := c.Get(string(auth.UserIDKey))
	if !exists {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	id := c.Param("id")
	if id == "" {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", "id is required")
		return
	}

	var input struct {
		Days int `json:"days"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		// Default to 30 days if body is empty or invalid
		input.Days = 30
	}
	if input.Days <= 0 {
		input.Days = 30
	}

	if err := globalDeviceAPI.svc.TrustDevice(id, userID.(string), input.Days); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	respondSuccess(c, gin.H{"trusted": true, "days": input.Days})
}

// CurrentDevice returns the current device info based on User-Agent and IP.
// GET /api/v1/devices/current
func CurrentDevice(c *gin.Context) {
	if globalDeviceAPI == nil {
		respondError(c, http.StatusInternalServerError, "device service not initialized")
		return
	}

	userID, exists := c.Get(string(auth.UserIDKey))
	if !exists {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	userAgent := c.GetHeader("User-Agent")
	clientIP := c.ClientIP()
	deviceID := model.GenerateDeviceID(userAgent, clientIP)

	isNew, device, err := globalDeviceAPI.svc.IsNewDevice(userID.(string), deviceID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to check device")
		return
	}

	if isNew {
		respondSuccess(c, gin.H{
			"device_id":   deviceID,
			"is_new":      true,
			"trusted":     false,
			"user_agent":  userAgent,
			"ip":          clientIP,
		})
		return
	}

	respondSuccess(c, gin.H{
		"id":              device.ID,
		"device_id":       device.DeviceID,
		"device_name":     device.DeviceName,
		"is_new":          false,
		"trusted":         device.Trusted,
		"trust_expires_at": device.TrustExpiresAt,
		"last_seen_at":    device.LastSeenAt,
		"user_agent":      device.UserAgent,
		"ip":              device.IP,
		"last_ip":         device.LastIP,
		"created_at":      device.CreatedAt,
	})
}

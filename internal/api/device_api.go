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

// getUserIDStrFromCtx safely extracts user ID string from context
func getUserIDStrFromCtx(c *gin.Context) (string, bool) {
	userID, exists := c.Get(string(auth.UserIDKey))
	if !exists {
		return "", false
	}
	userIDStr, ok := userID.(string)
	return userIDStr, ok
}

// ListDevices godoc
// @Summary      List devices
// @Description  Get all devices for the authenticated user
// @Tags         Devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "list of devices"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /devices [get]
func ListDevices(c *gin.Context) {
	if globalDeviceAPI == nil {
		respondError(c, http.StatusInternalServerError, "device service not initialized")
		return
	}

	userIDStr, ok := getUserIDStrFromCtx(c)
	if !ok {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	devices, err := globalDeviceAPI.svc.ListDevices(userIDStr)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to list devices")
		return
	}

	respondSuccess(c, devices)
}

// RevokeDevice godoc
// @Summary      Revoke device
// @Description  Remove trust from a device owned by the authenticated user
// @Tags         Devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Device ID"
// @Success      200 {object} map[string]interface{} "revocation confirmation"
// @Failure      400 {object} map[string]interface{} "invalid request or revoke failed"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /devices/{id} [delete]
func RevokeDevice(c *gin.Context) {
	if globalDeviceAPI == nil {
		respondError(c, http.StatusInternalServerError, "device service not initialized")
		return
	}

	userIDStr, ok := getUserIDStrFromCtx(c)
	if !ok {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	id := c.Param("id")
	if id == "" {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", "id is required")
		return
	}

	if err := globalDeviceAPI.svc.RevokeDevice(id, userIDStr); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	respondSuccess(c, gin.H{"revoked": true})
}

// TrustDevice godoc
// @Summary      Trust device
// @Description  Mark a device as trusted for a specified number of days (default 30)
// @Tags         Devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Device ID"
// @Param        request body object{days=int} false "Trust duration in days (default 30)"
// @Success      200 {object} map[string]interface{} "trust confirmation"
// @Failure      400 {object} map[string]interface{} "invalid request or trust failed"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /devices/{id}/trust [post]
func TrustDevice(c *gin.Context) {
	if globalDeviceAPI == nil {
		respondError(c, http.StatusInternalServerError, "device service not initialized")
		return
	}

	userIDStr, ok := getUserIDStrFromCtx(c)
	if !ok {
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

	if err := globalDeviceAPI.svc.TrustDevice(id, userIDStr, input.Days); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	respondSuccess(c, gin.H{"trusted": true, "days": input.Days})
}

// CurrentDevice godoc
// @Summary      Get current device
// @Description  Get the current device info based on User-Agent and IP
// @Tags         Devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "current device info"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /devices/current [get]
func CurrentDevice(c *gin.Context) {
	if globalDeviceAPI == nil {
		respondError(c, http.StatusInternalServerError, "device service not initialized")
		return
	}

	userIDStr, ok := getUserIDStrFromCtx(c)
	if !ok {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	userAgent := c.GetHeader("User-Agent")
	clientIP := c.ClientIP()
	deviceID := model.GenerateDeviceID(userAgent, clientIP)

	isNew, device, err := globalDeviceAPI.svc.IsNewDevice(userIDStr, deviceID)
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

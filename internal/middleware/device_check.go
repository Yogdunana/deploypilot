package middleware

import (
	"log/slog"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// DeviceCheckMiddleware checks the current device against the user's trusted devices.
// It sets the "X-New-Device" header if the device is unrecognized.
// API Key authenticated requests are exempt from device checks.
// It does not block requests — it only flags new devices and updates LastSeenAt.
func DeviceCheckMiddleware(svc *service.DeviceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if no user is authenticated
		userIDVal, exists := c.Get(string(auth.UserIDKey))
		if !exists {
			c.Next()
			return
		}
		userID, ok := userIDVal.(string)
		if !ok || userID == "" {
			c.Next()
			return
		}

		// Exempt API Key authenticated requests
		if authMethod, exists := c.Get("auth_method"); exists && authMethod == "api_key" {
			c.Next()
			return
		}

		userAgent := c.GetHeader("User-Agent")
		clientIP := c.ClientIP()
		deviceID := model.GenerateDeviceID(userAgent, clientIP)

		// Check if this is a new device
		isNew, device, err := svc.IsNewDevice(userID, deviceID)
		if err != nil {
			slog.Warn("failed to check device", "error", err)
			c.Next()
			return
		}

		if isNew {
			// Flag as new device but do not block
			c.Header("X-New-Device", "true")

			// Auto-register the device (untrusted)
			_, regErr := svc.RegisterDevice(userID, userAgent, clientIP, "", 0)
			if regErr != nil {
				slog.Warn("failed to register new device", "error", regErr)
			}
			c.Next()
			return
		}

		// Update LastSeenAt for existing device
		if device != nil {
			updateErr := svc.UpdateLastSeen(device.ID)
			if updateErr != nil {
				slog.Warn("failed to update device last seen", "error", updateErr)
			}
		}

		c.Next()
	}
}

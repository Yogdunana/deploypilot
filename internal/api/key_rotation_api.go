package api

import (
	"fmt"
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// RotateLicenseKeysHandler rotates the license signing keys (owner only).
func RotateLicenseKeysHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			N int `json:"n"` // total shares for Shamir backup (optional)
			M int `json:"m"` // threshold shares for Shamir backup (optional)
		}
		_ = c.ShouldBindJSON(&input)

		userID, _ := c.Get("user_id")
		uidStr := fmt.Sprintf("%v", userID)
		result, err := b.RotateLicenseKeys(c.Request.Context(), uidStr)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// ListLicenseKeysHandler returns all license signing keys (owner only).
func ListLicenseKeysHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := b.ListLicenseKeys(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// GetKeyVersionHandler returns the current active key version.
func GetKeyVersionHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := b.GetCurrentKeyVersion(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// BackupKeyShamirHandler creates Shamir's Secret Sharing backup of the private key.
func BackupKeyShamirHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			N int `json:"n" binding:"required,min=2,max=10"` // total shares
			M int `json:"m" binding:"required,min=2,max=10"` // threshold
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "invalid parameters: n and m required (2-10)")
			return
		}

		result, err := b.BackupKeyWithShamir(c.Request.Context(), input.N, input.M)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

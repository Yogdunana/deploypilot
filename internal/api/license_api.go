package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// GetLicenseStatusHandler returns the current license status.
// GET /api/v1/license/status
func GetLicenseStatusHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := b.GetLicenseStatus(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// ActivateLicenseHandler activates a license key or creates a community license.
// POST /api/v1/license/activate
// Body: {"license_key": "xxx"} OR {"use_type": "non_commercial", "agree_terms": true}
func ActivateLicenseHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			LicenseKey string `json:"license_key"`
			UseType    string `json:"use_type"`
			AgreeTerms bool   `json:"agree_terms"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "invalid request body")
			return
		}

		result, err := b.ActivateLicense(c.Request.Context(), input.LicenseKey, input.UseType, input.AgreeTerms)
		if err != nil {
			respondErrori18n(c, http.StatusBadRequest, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// DeactivateLicenseHandler deactivates the current license.
// POST /api/v1/license/deactivate
func DeactivateLicenseHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := b.DeactivateLicense(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// IssueLicenseHandler creates a new license key (developer only).
// POST /api/v1/license/issue
// Body: {"tenant_id": "...", "tier": "pro", "use_type": "commercial", "expires_at": "...", ...}
func IssueLicenseHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params map[string]interface{}
		if err := c.ShouldBindJSON(&params); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "invalid request body")
			return
		}

		result, err := b.IssueLicense(c.Request.Context(), params)
		if err != nil {
			respondErrori18n(c, http.StatusBadRequest, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// ListLicensesHandler lists all issued licenses (developer only).
// GET /api/v1/license/list
func ListLicensesHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := b.ListIssuedLicenses(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// RevokeLicenseHandler revokes a license (developer only).
// POST /api/v1/license/:id/revoke
// Body: {"reason": "..."}
func RevokeLicenseHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		licenseID := c.Param("id")
		if licenseID == "" {
			respondErrori18n(c, http.StatusBadRequest, "license id is required")
			return
		}

		var input struct {
			Reason string `json:"reason"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "invalid request body")
			return
		}

		result, err := b.RevokeLicense(c.Request.Context(), licenseID, input.Reason)
		if err != nil {
			respondErrori18n(c, http.StatusBadRequest, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// PurchaseAddonHandler purchases an addon for the current license.
// POST /api/v1/license/addon
// Body: {"addon_key": "feature:dashboard_tv", "amount": 0, "duration_days": 365}
func PurchaseAddonHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			AddonKey     string `json:"addon_key"`
			Amount       int    `json:"amount"`
			DurationDays int    `json:"duration_days"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "invalid request body")
			return
		}

		result, err := b.PurchaseAddon(c.Request.Context(), input.AddonKey, input.Amount, input.DurationDays)
		if err != nil {
			respondErrori18n(c, http.StatusBadRequest, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

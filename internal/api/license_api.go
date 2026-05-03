package api

import (
	"github.com/gin-gonic/gin"
)

// LicenseAPI handles license-related API requests.
type LicenseAPI struct{}

var globalLicenseAPI *LicenseAPI

// SetLicenseAPI sets the global license API instance.
func SetLicenseAPI(api *LicenseAPI) {
	globalLicenseAPI = api
}

// GetLicenseAPI returns the global license API instance.
func GetLicenseAPI() *LicenseAPI {
	return globalLicenseAPI
}

// GetLicenseStatus returns the current license status.
// GET /api/v1/license/status
func (api *LicenseAPI) GetLicenseStatus(c *gin.Context) {
	// Will be wired up with the license engine
	respondSuccess(c, gin.H{
		"status":    "active",
		"tier":      "community",
		"use_type":  "non_commercial",
		"features":  []string{},
		"addons":    []interface{}{},
		"limits": gin.H{
			"servers": 3,
			"apps":    10,
			"users":   5,
		},
	})
}

// ActivateLicense activates a license key.
// POST /api/v1/license/activate
func (api *LicenseAPI) ActivateLicense(c *gin.Context) {
	// Will be wired up with the license engine
	respondSuccess(c, gin.H{"message": "license activated"})
}

// DeactivateLicense deactivates the current license.
// POST /api/v1/license/deactivate
func (api *LicenseAPI) DeactivateLicense(c *gin.Context) {
	respondSuccess(c, gin.H{"message": "license deactivated"})
}

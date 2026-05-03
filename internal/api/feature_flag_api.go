package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// ListFeatureFlagsHandler returns all feature flags with their evaluation status.
func ListFeatureFlagsHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := b.ListFeatureFlags(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// GetFeatureFlagHandler returns a single feature flag by key.
func GetFeatureFlagHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		result, err := b.GetFeatureFlag(c.Request.Context(), key)
		if err != nil {
			respondErrori18n(c, http.StatusNotFound, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// UpdateFeatureFlagHandler updates a feature flag's properties (admin only).
func UpdateFeatureFlagHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		var input struct {
			Name            string `json:"name"`
			Description     string `json:"description"`
			Status          string `json:"status"`
			DefaultEnabled  *bool  `json:"default_enabled"`
			RequiredTier    string `json:"required_tier"`
			RequiredUseType string `json:"required_use_type"`
			Category        string `json:"category"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "invalid request body")
			return
		}

		params := make(map[string]interface{})
		if input.Name != "" {
			params["name"] = input.Name
		}
		if input.Description != "" {
			params["description"] = input.Description
		}
		if input.Status != "" {
			params["status"] = input.Status
		}
		if input.DefaultEnabled != nil {
			params["default_enabled"] = *input.DefaultEnabled
		}
		if input.RequiredTier != "" {
			params["required_tier"] = input.RequiredTier
		}
		if input.RequiredUseType != "" {
			params["required_use_type"] = input.RequiredUseType
		}
		if input.Category != "" {
			params["category"] = input.Category
		}

		result, err := b.UpdateFeatureFlag(c.Request.Context(), key, params)
		if err != nil {
			respondErrori18n(c, http.StatusBadRequest, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// SetFeatureFlagOverrideHandler creates or updates an admin override for a feature flag.
func SetFeatureFlagOverrideHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		var input struct {
			TenantID     string `json:"tenant_id" binding:"required"`
			Enabled      bool   `json:"enabled"`
			Reason       string `json:"reason"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "invalid request body")
			return
		}

		// Get current user from context for overridden_by
		overriddenBy, _ := c.Get("username")
		overriddenByStr, _ := overriddenBy.(string)
		if overriddenByStr == "" {
			overriddenByStr = "admin"
		}

		result, err := b.SetFeatureFlagOverride(c.Request.Context(), key, input.TenantID, input.Enabled, input.Reason, overriddenByStr)
		if err != nil {
			respondErrori18n(c, http.StatusBadRequest, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// DeleteFeatureFlagOverrideHandler removes an admin override.
func DeleteFeatureFlagOverrideHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		tenantID := c.Query("tenant_id")
		if tenantID == "" {
			respondErrori18n(c, http.StatusBadRequest, "tenant_id query parameter is required")
			return
		}

		result, err := b.DeleteFeatureFlagOverride(c.Request.Context(), key, tenantID)
		if err != nil {
			respondErrori18n(c, http.StatusBadRequest, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// ListFeatureFlagOverridesHandler returns all overrides for a given flag.
func ListFeatureFlagOverridesHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		result, err := b.ListFeatureFlagOverrides(c.Request.Context(), key)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// GetFeatureFlagsForTenantHandler returns all flags evaluated for a specific tenant.
func GetFeatureFlagsForTenantHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Param("tenant_id")
		if tenantID == "" {
			respondErrori18n(c, http.StatusBadRequest, "tenant_id is required")
			return
		}
		result, err := b.GetFeatureFlagsForTenant(c.Request.Context(), tenantID)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

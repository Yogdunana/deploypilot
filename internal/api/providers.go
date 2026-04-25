package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListProviders lists all providers.
// @Summary      List providers
// @Description  Retrieve all providers with optional type filtering
// @Tags         Providers
// @Produce      json
// @Security     BearerAuth
// @Param        type query string false "Filter by provider type"
// @Success      200 {object} map[string]interface{} "status, data (array of Provider)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /providers [get]
func ListProviders(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var providers []model.Provider
		query := db.Model(&model.Provider{})
		if pType := c.Query("type"); pType != "" {
			query = query.Where("type = ?", pType)
		}
		if err := query.Find(&providers).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if providers == nil {
			providers = []model.Provider{}
		}
		respondSuccess(c, providers)
	}
}

// CreateProvider creates a new provider.
// @Summary      Create a provider
// @Description  Create a new provider configuration
// @Tags         Providers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{name=string,type=string,config=map[string]interface{},tenant_id=string} true "Provider creation request"
// @Success      200 {object} map[string]interface{} "status, data (Provider)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /providers [post]
func CreateProvider(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input model.Provider
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}
		if input.Name == "" || input.Type == "" {
			respondError(c, http.StatusBadRequest, "name and type are required")
			return
		}

		input.ID = uuid.New().String()
		if input.TenantID == "" {
			input.TenantID = "tenant-default"
		}
		input.Enabled = true

		if err := db.Create(&input).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, input)
	}
}

// UpdateProvider updates a provider.
// @Summary      Update a provider
// @Description  Update an existing provider configuration
// @Tags         Providers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Provider ID"
// @Param        request body object true "Fields to update"
// @Success      200 {object} map[string]interface{} "status, data (updated Provider)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /providers/{id} [put]
func UpdateProvider(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input model.Provider
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		if err := db.Model(&model.Provider{}).Where("id = ?", id).Updates(input).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}

		var provider model.Provider
		if err := db.Where("id = ?", id).First(&provider).Error; err != nil {
			respondSuccess(c, gin.H{"status": "updated", "id": id})
			return
		}
		respondSuccess(c, provider)
	}
}

// DeleteProvider deletes a provider.
// @Summary      Delete a provider
// @Description  Delete a provider by ID
// @Tags         Providers
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Provider ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "provider not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /providers/{id} [delete]
func DeleteProvider(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result := db.Where("id = ?", id).Delete(&model.Provider{})
		if result.Error != nil {
			respondError(c, http.StatusInternalServerError, result.Error.Error())
			return
		}
		if result.RowsAffected == 0 {
			respondError(c, http.StatusNotFound, "provider not found")
			return
		}
		respondSuccess(c, gin.H{"message": "provider deleted", "id": id})
	}
}

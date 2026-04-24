package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListProviders lists all providers.
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

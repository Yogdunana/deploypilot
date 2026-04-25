package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListNotifications lists notification providers (type=notify).
// @Summary      List notification providers
// @Description  Retrieve all notification provider configurations
// @Tags         Notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data (array of Provider)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /notifications [get]
func ListNotifications(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var providers []model.Provider
		if err := db.Where("type = ?", "notify").Find(&providers).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if providers == nil {
			providers = []model.Provider{}
		}
		respondSuccess(c, providers)
	}
}

// CreateNotification creates a new notification provider.
// @Summary      Create a notification provider
// @Description  Create a new notification provider configuration
// @Tags         Notifications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{name=string,type=string,config=map[string]interface{},tenant_id=string} true "Notification provider creation request"
// @Success      200 {object} map[string]interface{} "status, data (Provider)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /notifications [post]
func CreateNotification(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input model.Provider
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}
		if input.Name == "" {
			respondError(c, http.StatusBadRequest, "name is required")
			return
		}

		input.ID = uuid.New().String()
		input.Type = "notify"
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

// UpdateNotification updates a notification provider.
// @Summary      Update a notification provider
// @Description  Update an existing notification provider configuration
// @Tags         Notifications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Notification provider ID"
// @Param        request body object true "Fields to update"
// @Success      200 {object} map[string]interface{} "status, data (updated Provider)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /notifications/{id} [put]
func UpdateNotification(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input model.Provider
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		if err := db.Model(&model.Provider{}).Where("id = ? AND type = ?", id, "notify").Updates(input).Error; err != nil {
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

// DeleteNotification deletes a notification provider.
// @Summary      Delete a notification provider
// @Description  Delete a notification provider by ID
// @Tags         Notifications
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Notification provider ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "notification provider not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /notifications/{id} [delete]
func DeleteNotification(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result := db.Where("id = ? AND type = ?", id, "notify").Delete(&model.Provider{})
		if result.Error != nil {
			respondError(c, http.StatusInternalServerError, result.Error.Error())
			return
		}
		if result.RowsAffected == 0 {
			respondError(c, http.StatusNotFound, "notification provider not found")
			return
		}
		respondSuccess(c, gin.H{"message": "notification provider deleted", "id": id})
	}
}

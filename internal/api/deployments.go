package api

import (
	"net/http"
	"strconv"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListDeployments lists deployment records.
// @Summary      List deployments
// @Description  Retrieve deployment records with optional filtering by app_id and status
// @Tags         Deployments
// @Produce      json
// @Security     BearerAuth
// @Param        app_id query string false "Filter by application ID"
// @Param        status query string false "Filter by deployment status"
// @Success      200 {object} map[string]interface{} "status, data (array of DeploymentRecord)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /deployments [get]
func ListDeployments(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}

		var records []model.DeploymentRecord
		var total int64

		query := db.Model(&model.DeploymentRecord{})

		if appID := c.Query("app_id"); appID != "" {
			query = query.Where("app_name = ?", appID)
		}
		if status := c.Query("status"); status != "" {
			query = query.Where("status = ?", status)
		}

		if err := query.Count(&total).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		if records == nil {
			records = []model.DeploymentRecord{}
		}
		// Return paginated response with data key for backward compatibility
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data":   records,
			"pagination": gin.H{
				"page":      page,
				"page_size": pageSize,
				"total":     total,
			},
		})
	}
}

// GetDeployment returns a single deployment record.
// @Summary      Get a deployment
// @Description  Retrieve a single deployment record by ID
// @Tags         Deployments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Deployment ID"
// @Success      200 {object} map[string]interface{} "status, data (DeploymentRecord)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "deployment not found"
// @Router       /deployments/{id} [get]
func GetDeployment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var record model.DeploymentRecord
		if err := db.Where("id = ?", id).First(&record).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.deployment.not_found")
			return
		}
		respondSuccess(c, record)
	}
}

package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListDeployments lists deployment records.
func ListDeployments(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var records []model.DeploymentRecord
		query := db.Model(&model.DeploymentRecord{})

		if appID := c.Query("app_id"); appID != "" {
			query = query.Where("app_name = ?", appID)
		}
		if status := c.Query("status"); status != "" {
			query = query.Where("status = ?", status)
		}

		if err := query.Order("created_at DESC").Find(&records).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if records == nil {
			records = []model.DeploymentRecord{}
		}
		respondSuccess(c, records)
	}
}

// GetDeployment returns a single deployment record.
func GetDeployment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var record model.DeploymentRecord
		if err := db.Where("id = ?", id).First(&record).Error; err != nil {
			respondError(c, http.StatusNotFound, "deployment not found")
			return
		}
		respondSuccess(c, record)
	}
}

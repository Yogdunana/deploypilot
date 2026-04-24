package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListTemplates lists all available templates (built-in + custom).
func ListTemplates(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		templates, err := bridge.ListTemplates(c.Request.Context())
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, templates)
	}
}

// CreateTemplate creates a custom template.
func CreateTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Name        string `json:"name" binding:"required"`
			Type        string `json:"type" binding:"required"`
			Description string `json:"description"`
			BuildCmd    string `json:"build_cmd"`
			RunCmd      string `json:"run_cmd"`
			Port        int    `json:"port"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		id := uuid.New().String()
		// Store custom templates in the providers table with type "template"
		if err := db.Exec(
			`INSERT INTO providers (id, tenant_id, type, name, config, enabled) VALUES (?, ?, 'template', ?, ?, 1)`,
			id, "tenant-default", input.Name,
			map[string]interface{}{
				"type":        input.Type,
				"description": input.Description,
				"build_cmd":   input.BuildCmd,
				"run_cmd":     input.RunCmd,
				"port":        input.Port,
			},
		).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, gin.H{"id": id, "name": input.Name, "type": input.Type})
	}
}

// UpdateTemplate updates a custom template.
func UpdateTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			BuildCmd    string `json:"build_cmd"`
			RunCmd      string `json:"run_cmd"`
			Port        int    `json:"port"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		if err := db.Exec(
			`UPDATE providers SET config = ?, name = ? WHERE id = ? AND type = 'template'`,
			map[string]interface{}{
				"description": input.Description,
				"build_cmd":   input.BuildCmd,
				"run_cmd":     input.RunCmd,
				"port":        input.Port,
			},
			input.Name, id,
		).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, gin.H{"id": id, "message": "template updated"})
	}
}

// DeleteTemplate deletes a custom template.
func DeleteTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result := db.Exec(`DELETE FROM providers WHERE id = ? AND type = 'template'`, id)
		if result.Error != nil {
			respondError(c, http.StatusInternalServerError, result.Error.Error())
			return
		}
		if result.RowsAffected == 0 {
			respondError(c, http.StatusNotFound, "template not found")
			return
		}
		respondSuccess(c, gin.H{"message": "template deleted", "id": id})
	}
}

package api

import (
	"net/http"
	"strconv"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// globalWebhookAPI is the package-level OutboundWebhookAPI instance, accessible via GetGlobalWebhookAPI().
var globalWebhookAPI *OutboundWebhookAPI

// GetGlobalWebhookAPI returns the global OutboundWebhookAPI instance.
func GetGlobalWebhookAPI() *OutboundWebhookAPI { return globalWebhookAPI }

// OutboundWebhookAPI provides HTTP handlers for outbound webhook management.
type OutboundWebhookAPI struct {
	db *gorm.DB
}

// NewOutboundWebhookAPI creates a new OutboundWebhookAPI.
func NewOutboundWebhookAPI(db *gorm.DB) *OutboundWebhookAPI {
	return &OutboundWebhookAPI{db: db}
}

// CreateWebhook creates a new outbound webhook.
// POST /api/v1/webhooks
func (a *OutboundWebhookAPI) CreateWebhook(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var webhook model.OutboundWebhook
	if err := c.ShouldBindJSON(&webhook); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	svc := service.NewOutboundWebhookService(db, nil)
	if err := svc.Create(c.Request.Context(), &webhook); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data":   webhook,
	})
}

// ListWebhooks lists all outbound webhooks with pagination.
// GET /api/v1/webhooks?page=1&page_size=20
func (a *OutboundWebhookAPI) ListWebhooks(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	svc := service.NewOutboundWebhookService(db, nil)
	webhooks, total, err := svc.List(c.Request.Context(), c.GetString("tenant_id"), page, pageSize)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondPaginated(c, webhooks, int(total), page, pageSize)
}

// GetWebhook gets a webhook by ID.
// GET /api/v1/webhooks/:id
func (a *OutboundWebhookAPI) GetWebhook(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id := c.Param("id")

	svc := service.NewOutboundWebhookService(db, nil)
	webhook, err := svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
		return
	}

	respondSuccess(c, webhook)
}

// UpdateWebhook updates an existing webhook.
// PUT /api/v1/webhooks/:id
func (a *OutboundWebhookAPI) UpdateWebhook(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id := c.Param("id")

	var webhook model.OutboundWebhook
	if err := c.ShouldBindJSON(&webhook); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	webhook.ID = id
	svc := service.NewOutboundWebhookService(db, nil)
	if err := svc.Update(c.Request.Context(), &webhook); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, webhook)
}

// DeleteWebhook deletes a webhook by ID.
// DELETE /api/v1/webhooks/:id
func (a *OutboundWebhookAPI) DeleteWebhook(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id := c.Param("id")

	svc := service.NewOutboundWebhookService(db, nil)
	if err := svc.Delete(c.Request.Context(), id); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"id": id, "status": "deleted"})
}

// TestWebhook sends a test delivery to a webhook.
// POST /api/v1/webhooks/:id/test
func (a *OutboundWebhookAPI) TestWebhook(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id := c.Param("id")

	svc := service.NewOutboundWebhookService(db, nil)
	delivery, err := svc.TestDelivery(c.Request.Context(), id)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, delivery)
}

// ListDeliveries lists delivery records for a webhook with pagination.
// GET /api/v1/webhooks/:id/deliveries?page=1&page_size=20
func (a *OutboundWebhookAPI) ListDeliveries(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	webhookID := c.Param("id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var deliveries []model.WebhookDelivery
	var total int64

	query := db.WithContext(c.Request.Context()).Model(&model.WebhookDelivery{}).Where("webhook_id = ?", webhookID)
	if err := query.Count(&total).Error; err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&deliveries).Error; err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondPaginated(c, deliveries, int(total), page, pageSize)
}

// GetDelivery gets a single delivery record.
// GET /api/v1/webhooks/:id/deliveries/:did
func (a *OutboundWebhookAPI) GetDelivery(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	deliveryID := c.Param("did")
	webhookID := c.Param("id")

	var delivery model.WebhookDelivery
	if err := db.WithContext(c.Request.Context()).Where("id = ? AND webhook_id = ?", deliveryID, webhookID).First(&delivery).Error; err != nil {
		respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
		return
	}

	respondSuccess(c, delivery)
}

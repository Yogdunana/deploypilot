package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// ListDNSRecords lists DNS records for a domain.
// @Summary      List DNS records
// @Description  Retrieve all DNS records for a given domain
// @Tags         DNS
// @Produce      json
// @Security     BearerAuth
// @Param        domain query string true "Domain name"
// @Success      200 {object} map[string]interface{} "status, data (array of DNS records)"
// @Failure      400 {object} map[string]interface{} "domain query parameter is required"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /dns/records [get]
func ListDNSRecords(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		domain := c.Query("domain")
		if domain == "" {
			respondErrori18n(c, http.StatusBadRequest, "error.dns.domain_required")
			return
		}

		records, err := bridge.DNSListRecords(c.Request.Context(), domain)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, records)
	}
}

// CreateDNSRecord creates a new DNS record.
// @Summary      Create a DNS record
// @Description  Create a new DNS record for a domain
// @Tags         DNS
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{domain=string,type=string,name=string,value=string} true "DNS record creation request"
// @Success      200 {object} map[string]interface{} "status, data (DNS record)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /dns/records [post]
func CreateDNSRecord(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Domain string `json:"domain" binding:"required"`
			Type   string `json:"type" binding:"required"`
			Name   string `json:"name" binding:"required"`
			Value  string `json:"value" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		record, err := bridge.DNSCreateRecord(c.Request.Context(), input.Domain, input.Type, input.Name, input.Value)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, record)
	}
}

// UpdateDNSRecord updates a DNS record.
// @Summary      Update a DNS record
// @Description  Update an existing DNS record with a new value
// @Tags         DNS
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "DNS record ID"
// @Param        request body object{domain=string,subdomain=string,type=string,new_value=string} true "DNS record update request"
// @Success      200 {object} map[string]interface{} "status, data (updated DNS record)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /dns/records/{id} [put]
func UpdateDNSRecord(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Domain    string `json:"domain" binding:"required"`
			Subdomain string `json:"subdomain" binding:"required"`
			Type      string `json:"type" binding:"required"`
			NewValue  string `json:"new_value" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		record, err := bridge.UpdateDNSRecord(c.Request.Context(), input.Domain, input.Subdomain, input.Type, input.NewValue)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, record)
	}
}

// DeleteDNSRecord deletes a DNS record.
// @Summary      Delete a DNS record
// @Description  Delete a DNS record by ID
// @Tags         DNS
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "DNS record ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /dns/records/{id} [delete]
func DeleteDNSRecord(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := bridge.DNSDeleteRecord(c.Request.Context(), id); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, gin.H{"message": "DNS record deleted", "id": id})
	}
}

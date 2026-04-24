package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// ListDNSRecords lists DNS records for a domain.
func ListDNSRecords(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		domain := c.Query("domain")
		if domain == "" {
			respondError(c, http.StatusBadRequest, "domain query parameter is required")
			return
		}

		records, err := bridge.DNSListRecords(c.Request.Context(), domain)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, records)
	}
}

// CreateDNSRecord creates a new DNS record.
func CreateDNSRecord(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Domain string `json:"domain" binding:"required"`
			Type   string `json:"type" binding:"required"`
			Name   string `json:"name" binding:"required"`
			Value  string `json:"value" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		record, err := bridge.DNSCreateRecord(c.Request.Context(), input.Domain, input.Type, input.Name, input.Value)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, record)
	}
}

// UpdateDNSRecord updates a DNS record.
func UpdateDNSRecord(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Domain    string `json:"domain" binding:"required"`
			Subdomain string `json:"subdomain" binding:"required"`
			Type      string `json:"type" binding:"required"`
			NewValue  string `json:"new_value" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		record, err := bridge.UpdateDNSRecord(c.Request.Context(), input.Domain, input.Subdomain, input.Type, input.NewValue)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, record)
	}
}

// DeleteDNSRecord deletes a DNS record.
func DeleteDNSRecord(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := bridge.DNSDeleteRecord(c.Request.Context(), id); err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, gin.H{"message": "DNS record deleted", "id": id})
	}
}

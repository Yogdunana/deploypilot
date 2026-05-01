package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityEntrance returns middleware that enforces a security entrance URL prefix.
// When configured (e.g. "/my-secret-panel"), all non-API requests must include this prefix.
// API and WebSocket paths are exempt to allow programmatic access.
func SecurityEntrance(entrance string) gin.HandlerFunc {
	if entrance == "" {
		return func(c *gin.Context) { c.Next() }
	}

	// Normalize: ensure leading slash, no trailing slash
	entrance = "/" + strings.Trim(entrance, "/")

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip health checks and API discovery
		if path == "/health" || path == "/api" || path == "/swagger" {
			c.Next()
			return
		}

		// Skip API paths (they use Bearer token auth)
		if strings.HasPrefix(path, "/api/") {
			c.Next()
			return
		}

		// Skip WebSocket paths
		if strings.HasPrefix(path, "/ws/") {
			c.Next()
			return
		}

		// Skip static assets (they are served under the SPA)
		if strings.HasPrefix(path, "/assets/") || path == "/icon.svg" {
			c.Next()
			return
		}

		// All other paths must start with the security entrance
		if !strings.HasPrefix(path, entrance) {
			// Return 404 to hide the panel existence
			c.Status(http.StatusNotFound)
			c.Abort()
			return
		}

		// Strip the entrance prefix for the SPA router
		c.Request.URL.Path = strings.TrimPrefix(path, entrance)
		if c.Request.URL.Path == "" {
			c.Request.URL.Path = "/"
		}

		c.Next()
	}
}

// DomainBinding returns middleware that restricts access to allowed domains.
// When configured, requests with Host header not matching any allowed domain are rejected.
func DomainBinding(allowedDomains []string) gin.HandlerFunc {
	if len(allowedDomains) == 0 {
		return func(c *gin.Context) { c.Next() }
	}

	// Normalize domains (lowercase, trim spaces)
	normalized := make(map[string]bool, len(allowedDomains))
	for _, d := range allowedDomains {
		d = strings.TrimSpace(strings.ToLower(d))
		if d != "" {
			normalized[d] = true
		}
	}

	return func(c *gin.Context) {
		host := strings.ToLower(c.Request.Host)
		// Strip port if present
		if idx := strings.LastIndex(host, ":"); idx > 0 {
			host = host[:idx]
		}

		if !normalized[host] {
			c.Status(http.StatusForbidden)
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "access denied: domain not allowed",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// IPWhitelist returns middleware that restricts access to allowed IP addresses.
// When configured, requests from IPs not in the whitelist are rejected.
// Supports CIDR notation (e.g. "192.168.1.0/24") and single IPs.
func IPWhitelist(allowedIPs []string) gin.HandlerFunc {
	if len(allowedIPs) == 0 {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !isIPAllowed(clientIP, allowedIPs) {
			c.Status(http.StatusForbidden)
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "access denied: IP not allowed",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// isIPAllowed checks if an IP is in the allowed list.
// Supports exact match and CIDR notation.
func isIPAllowed(ip string, allowed []string) bool {
	for _, allowedIP := range allowed {
		allowedIP = strings.TrimSpace(allowedIP)
		if allowedIP == "" {
			continue
		}
		if ip == allowedIP {
			return true
		}
		// CIDR check using simple prefix matching for /8, /16, /24
		if strings.Contains(allowedIP, "/") {
			if matchCIDR(ip, allowedIP) {
				return true
			}
		}
	}
	return false
}

// matchCIDR performs basic CIDR matching for common subnet masks.
func matchCIDR(ip, cidr string) bool {
	// Parse CIDR
	parts := strings.SplitN(cidr, "/", 2)
	if len(parts) != 2 {
		return false
	}
	network := parts[0]

	// Simple prefix match for the network part
	ipParts := strings.Split(ip, ".")
	netParts := strings.Split(network, "."")
	if len(ipParts) != 4 || len(netParts) != 4 {
		return false
	}

	switch parts[1] {
	case "8":
		return ipParts[0] == netParts[0]
	case "16":
		return ipParts[0] == netParts[0] && ipParts[1] == netParts[1]
	case "24":
		return ipParts[0] == netParts[0] && ipParts[1] == netParts[1] && ipParts[2] == netParts[2]
	case "32":
		return ip == network
	default:
		// For non-standard CIDR, do exact match
		return ip == network
	}
}

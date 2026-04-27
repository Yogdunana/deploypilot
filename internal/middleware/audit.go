package middleware

import (
	"fmt"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// AuditMiddleware logs all authenticated mutating requests.
func AuditMiddleware(auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		// After request, record audit log
		// Skip non-mutating methods (GET, HEAD, OPTIONS)
		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			return
		}

		userID, _ := c.Get("userID")
		username, _ := c.Get("username")
		traceID, _ := c.Get(TraceIDContextKey)

		action := mapMethodToAction(method, c.Request.URL.Path)
		_ = auditSvc.Record(c.Request.Context(), service.AuditEntry{
			UserID:       toUint(userID),
			Username:     toString(username),
			Action:       action,
			ResourceType: extractResourceType(c.Request.URL.Path),
			ResourceID:   extractResourceID(c.Request.URL.Path),
			IPAddress:    service.ClientIP(c.ClientIP(), c.GetHeader("X-Forwarded-For")),
			UserAgent:    c.Request.UserAgent(),
			TraceID:      toString(traceID),
		})
	}
}

// mapMethodToAction maps an HTTP method and path to a human-readable audit action.
func mapMethodToAction(method, path string) string {
	// Normalize path: /api/v1/apps/:id/deploy -> app.deploy
	trimmed := strings.TrimPrefix(path, "/api/v1")
	trimmed = strings.TrimPrefix(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 1 || parts[0] == "" {
		return fmt.Sprintf("unknown.%s", strings.ToLower(method))
	}

	resource := parts[0]
	// Check for sub-actions (e.g., /apps/:id/deploy, /apps/:id/build)
	if len(parts) >= 3 {
		subAction := parts[2]
		return fmt.Sprintf("%s.%s", resource, subAction)
	}

	action := methodToVerb(method)
	return fmt.Sprintf("%s.%s", resource, action)
}

// extractResourceType extracts the resource type from a URL path.
func extractResourceType(path string) string {
	trimmed := strings.TrimPrefix(path, "/api/v1")
	trimmed = strings.TrimPrefix(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 1 || parts[0] == "" {
		return "unknown"
	}
	return parts[0]
}

// extractResourceID extracts the resource ID from a URL path.
func extractResourceID(path string) string {
	trimmed := strings.TrimPrefix(path, "/api/v1")
	trimmed = strings.TrimPrefix(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

func methodToVerb(method string) string {
	switch method {
	case "POST":
		return "create"
	case "PUT":
		return "update"
	case "DELETE":
		return "delete"
	case "PATCH":
		return "patch"
	default:
		return strings.ToLower(method)
	}
}

func toUint(v interface{}) uint {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case uint:
		return val
	case uint64:
		return uint(val)
	case int:
		return uint(val)
	case int64:
		return uint(val)
	case float64:
		return uint(val)
	case string:
		return 0
	default:
		return 0
	}
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

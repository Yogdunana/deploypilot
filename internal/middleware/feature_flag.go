package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// FeatureFlagEvaluator defines the interface for checking feature flag status.
type FeatureFlagEvaluator interface {
	EvaluateFeature(ctx interface{}, featureKey string, tenantID string) (bool, error)
}

// FeatureFlagMiddleware creates a middleware that checks if a feature is enabled.
// If the feature is disabled, it returns HTTP 402 Payment Required.
func FeatureFlagMiddleware(evaluator FeatureFlagEvaluator, featureKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get tenant_id from context (set by auth middleware)
		tenantID, _ := c.Get("tenant_id")
		tenantIDStr, _ := tenantID.(string)

		enabled, err := evaluator.EvaluateFeature(c.Request.Context(), featureKey, tenantIDStr)
		if err != nil {
			slog.Warn("feature flag evaluation error", "feature", featureKey, "error", err)
			// On error, allow the request (fail-open for robustness)
			c.Next()
			return
		}

		if !enabled {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"status":  "error",
				"code":    "feature_disabled",
				"message": "This feature requires a higher license tier or is currently disabled",
				"feature": featureKey,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

package api

import (
	"math"
	"net/http"
	"strconv"

	appErrors "github.com/Yogdunana/deploypilot/pkg/errors"

	"github.com/Yogdunana/deploypilot/internal/i18n"
	"github.com/gin-gonic/gin"
)

// respondSuccess returns a standardized success response.
func respondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   data,
	})
}

// respondError returns a standardized error response.
func respondError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"status":  "error",
		"message": message,
	})
}

// respondErrori18n returns a standardized error response with i18n translation.
// It translates the given key using the locale from the gin.Context.
func respondErrori18n(c *gin.Context, code int, key string, args ...interface{}) {
	locale := i18n.GetLocaleFromContext(c)
	var message string
	if len(args) > 0 {
		message = i18n.Tf(locale, key, args...)
	} else {
		message = i18n.T(locale, key)
	}
	c.JSON(code, gin.H{
		"status":  "error",
		"message": message,
	})
}

// RespondAppError returns a standardized error response for an AppError.
// If the AppError has an I18nKey, it uses i18n.Tf to translate the message and suggestion.
// Otherwise, it falls back to the static Message and Suggestion fields.
func RespondAppError(c *gin.Context, appErr *appErrors.AppError) {
	locale := i18n.GetLocaleFromContext(c)

	var message, suggestion string
	if appErr.I18nKey != "" {
		message = i18n.Tf(locale, appErr.I18nKey+".message")
		suggestion = i18n.Tf(locale, appErr.I18nKey+".suggestion")
	} else {
		message = appErr.Message
		suggestion = appErr.Suggestion
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"status":     "error",
		"code":       appErr.Code,
		"message":    message,
		"suggestion": suggestion,
	})
}

// respondPaginated returns a paginated response with metadata.
func respondPaginated(c *gin.Context, data interface{}, total, page, pageSize int) {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   data,
		"pagination": gin.H{
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": totalPages,
		},
	})
}

// parsePaginationParams extracts and validates pagination parameters from query string.
// Returns page (1-based) and pageSize with sensible defaults.
func parsePaginationParams(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return
}

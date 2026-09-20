package api

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	appErrors "github.com/Yogdunana/deploypilot/pkg/errors"

	"github.com/Yogdunana/deploypilot/internal/i18n"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	if appErr == nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	locale := i18n.GetLocaleFromContext(c)

	var message, suggestion string
	if appErr.I18nKey != "" {
		message = i18n.Tf(locale, appErr.I18nKey+".message")
		suggestion = i18n.Tf(locale, appErr.I18nKey+".suggestion")
	} else {
		message = appErr.Message
		suggestion = appErr.Suggestion
	}

	status := http.StatusInternalServerError
	if appErr != nil {
		status = appErr.HTTPStatus()
	}

	c.JSON(status, gin.H{
		"status":     "error",
		"code":       appErr.Code,
		"message":    message,
		"suggestion": suggestion,
	})
}

// isRecordNotFound reports whether err is a GORM record-not-found error,
// including errors wrapped by model helpers.
func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
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
	page = 1
	pageSize = 20

	if raw := c.Query("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err == nil && parsed >= 1 {
			page = parsed
		}
	}
	if raw := c.Query("page_size"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err == nil && parsed >= 1 && parsed <= 100 {
			pageSize = parsed
		}
	}
	return
}

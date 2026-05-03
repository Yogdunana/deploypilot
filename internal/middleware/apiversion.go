package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// APIVersionHeader is the response header indicating the API version.
	APIVersionHeader = "X-API-Version"
	// APIDeprecationHeader indicates whether the API version is deprecated.
	APIDeprecationHeader = "Deprecation"
	// APISunsetHeader indicates the date after which the API version will be shut down.
	APISunsetHeader = "Sunset"
	// APILinkHeader is used for pagination and version links.
	APILinkHeader = "Link"

	// APICurrentVersion is the current stable API version.
	APICurrentVersion = "v1"
)

// APISupportedVersions lists all currently supported API versions.
var APISupportedVersions = []string{"v1"}

// APIVersionMiddleware validates the API version from the URL path and sets response headers.
// It does not block unsupported versions -- the URL routing already handles 404 for unknown paths.
func APIVersionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		version := extractAPIVersion(path)

		// Always set the version header so consumers know which version served the request.
		c.Header(APIVersionHeader, version)

		// Warn when an unsupported version is requested.
		if !isVersionSupported(version) {
			c.Header("Accept-Version", strings.Join(APISupportedVersions, ", "))
		}

		// Set deprecation headers for old versions (future: when v2 exists).
		if version != APICurrentVersion {
			// When v2 is released, v1 will get:
			// c.Header(APIDeprecationHeader, "true")
			// c.Header(APISunsetHeader, "2027-01-01")
			// c.Header(APILinkHeader, "</api/v2>; rel=\"successor-version\"")
		}

		c.Next()
	}
}

// extractAPIVersion extracts the API version from a URL path.
// Expected path format: /api/v1/...
func extractAPIVersion(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "api" && strings.HasPrefix(parts[1], "v") {
		return parts[1]
	}
	return APICurrentVersion
}

// isVersionSupported checks whether the given version string is in the supported list.
func isVersionSupported(version string) bool {
	for _, v := range APISupportedVersions {
		if v == version {
			return true
		}
	}
	return false
}

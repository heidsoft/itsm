package middleware

import (
	"encoding/json"
	"regexp"

	"github.com/gin-gonic/gin"
)

var (
	// simple patterns for masking; can be extended
	passwordPattern = regexp.MustCompile(`(?i)"password"\s*:\s*"[^"]*"`)
	apiKeyPattern   = regexp.MustCompile(`(?i)"api[_-]?key"\s*:\s*"[^"]*"`)
	tokenPattern    = regexp.MustCompile(`(?i)"token"\s*:\s*"[^"]*"`)
)

// MaskSensitiveFields masks sensitive fields in JSON request bodies for logging/audit
func MaskSensitiveFields(body string) string {
	masked := passwordPattern.ReplaceAllString(body, `"password":"***"`)
	masked = apiKeyPattern.ReplaceAllString(masked, `"api_key":"***"`)
	masked = tokenPattern.ReplaceAllString(masked, `"token":"***"`)
	return masked
}

// MaskResponseMiddleware is optional if we later log responses
func MaskResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		// placeholder for future response masking if needed
		_ = json.Marshal
	}
}

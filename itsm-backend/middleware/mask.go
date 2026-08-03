package middleware

import (
	"encoding/json"
	"regexp"

	"github.com/gin-gonic/gin"
)

// Ordered from most specific to most generic to avoid collisions on keys
// like "access_token" being clobbered by a broader "token" rule first.
var maskRules = []struct {
	re   *regexp.Regexp
	repl string
}{
	{regexp.MustCompile(`(?i)"(password|passwd|pwd|current_password|new_password|confirm_password)"\s*:\s*"[^"]*"`), `"$1":"***"`},
	{regexp.MustCompile(`(?i)"(api[_-]?key|apikey|x-api-key)"\s*:\s*"[^"]*"`), `"$1":"***"`},
	{regexp.MustCompile(`(?i)"(access[_-]?token|refresh[_-]?token|id[_-]?token|auth[_-]?token|bearer[_-]?token|jwt|session[_-]?token|csrf[_-]?token)"\s*:\s*"[^"]*"`), `"$1":"***"`},
	{regexp.MustCompile(`(?i)"(secret|client[_-]?secret|app[_-]?secret|private[_-]?key|signing[_-]?key|hmac[_-]?key)"\s*:\s*"[^"]*"`), `"$1":"***"`},
	{regexp.MustCompile(`(?i)"(authorization|proxy-authorization)"\s*:\s*"[^"]*"`), `"$1":"***"`},
	{regexp.MustCompile(`(?i)"(cookie|set-cookie)"\s*:\s*"[^"]*"`), `"$1":"***"`},
	{regexp.MustCompile(`(?i)"(credit[_-]?card|card[_-]?number|cc[_-]?num|cvv|cvc)"\s*:\s*"[^"]*"`), `"$1":"***"`},
	{regexp.MustCompile(`(?i)"(bank[_-]?account|routing[_-]?number|iban|swift|bic)"\s*:\s*"[^"]*"`), `"$1":"***"`},
	{regexp.MustCompile(`(?i)"(ssn|social[_-]?security|tax[_-]?id|national[_-]?id|id[_-]?number|passport[_-]?number)"\s*:\s*"[^"]*"`), `"$1":"***"`},
	{regexp.MustCompile(`(?i)"(phone|mobile|tel|telephone)"\s*:\s*"[^"]*"`), `"$1":"***"`},
	{regexp.MustCompile(`(?i)"(email|e-mail)"\s*:\s*"[^"]*"`), `"$1":"***"`},
	{regexp.MustCompile(`(?i)"(address|street|postal[_-]?code|zip[_-]?code)"\s*:\s*"[^"]*"`), `"$1":"***"`},
	{regexp.MustCompile(`(?i)"(id[_-]?token|oauth[_-]?token|feishu[_-]?token|wecom[_-]?token|dingtalk[_-]?token)"\s*:\s*"[^"]*"`), `"$1":"***"`},
	{regexp.MustCompile(`(?i)"(db[_-]?password|database[_-]?password|redis[_-]?password|smtp[_-]?password)"\s*:\s*"[^"]*"`), `"$1":"***"`},
	// Generic "token" fallback — apply last so more specific rules above win
	{regexp.MustCompile(`(?i)"token"\s*:\s*"[^"]*"`), `"token":"***"`},
}

// MaskSensitiveFields masks sensitive fields in JSON request bodies for logging/audit.
// Uses an ordered rule list so specific keys are replaced before generic "token".
func MaskSensitiveFields(body string) string {
	masked := body
	for _, r := range maskRules {
		masked = r.re.ReplaceAllString(masked, r.repl)
	}
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

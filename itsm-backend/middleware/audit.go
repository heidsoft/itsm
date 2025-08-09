package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
)

// AuditMiddleware persists audit logs for write operations with request_id
func AuditMiddleware(client *ent.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch && method != http.MethodDelete {
			c.Next()
			return
		}

		// read body safely and restore
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// collect fields
		rid := c.GetString("request_id")
		tenantID := c.GetInt("tenant_id")
		userID := 0
		if v, ok := c.Get("user_id"); ok {
			switch t := v.(type) {
			case int:
				userID = t
			case int32:
				userID = int(t)
			case int64:
				userID = int(t)
			}
		}

		ip := c.ClientIP()
		path := c.Request.URL.Path
		methodStr := c.Request.Method
		status := c.Writer.Status()

		// limit stored body size to avoid oversized logs
		requestBody := string(bodyBytes)
		if len(requestBody) > 4096 {
			requestBody = requestBody[:4096] + "..."
		}
		// mask sensitive fields
		requestBody = MaskSensitiveFields(requestBody)

		// resource/action heuristic from path
		resource := ""
		action := strings.ToLower(methodStr)
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) >= 3 { // e.g. api v1 tickets
			resource = parts[2]
		} else if len(parts) >= 1 {
			resource = parts[len(parts)-1]
		}

		// write asynchronously to reduce latency
		go func() {
			_ = client.AuditLog.Create().
				SetCreatedAt(time.Now()).
				SetTenantID(tenantID).
				SetUserID(userID).
				SetRequestID(rid).
				SetIP(ip).
				SetPath(path).
				SetMethod(methodStr).
				SetStatusCode(status).
				SetResource(resource).
				SetAction(action).
				SetRequestBody(requestBody).
				Exec(c.Request.Context())

			if globalLogger != nil {
				globalLogger.Infow("AuditLog", "request_id", rid, "tenant_id", tenantID, "user_id", userID, "path", path, "method", methodStr, "status_code", status, "latency_ms", duration.Milliseconds())
			}
		}()
	}
}

package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
)

type loginAuditRequestKey struct{}

type LoginAuditRequest struct{ IP, UserAgent string }

func WithLoginAuditRequest(ctx context.Context, ip, userAgent string) context.Context {
	return context.WithValue(ctx, loginAuditRequestKey{}, LoginAuditRequest{IP: ip, UserAgent: userAgent})
}

// RecordLoginAudit writes only explicitly allow-listed authentication metadata.
// Passwords and issued/reset tokens are never accepted by this API.
func RecordLoginAudit(ctx context.Context, client *ent.Client, userID, tenantID int, username, action, failureReason string) {
	if client == nil {
		return
	}
	req, _ := ctx.Value(loginAuditRequestKey{}).(LoginAuditRequest)
	payload, _ := json.Marshal(map[string]string{"username": username, "userAgent": req.UserAgent, "failureReason": failureReason})
	auditCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := client.AuditLog.Create().SetCreatedAt(time.Now()).SetTenantID(tenantID).SetUserID(userID).
		SetIP(req.IP).SetResource("auth").SetAction(action).SetPath("/api/v1/auth/login").
		SetMethod("POST").SetStatusCode(map[bool]int{true: 200, false: 401}[action == "LOGIN_SUCCESS"]).
		SetRequestBody(string(payload)).Exec(auditCtx)
	if err != nil && globalLogger != nil {
		globalLogger.Errorw("failed to save login audit", "error", err, "action", action)
	}
}

// AuditableAction 可审计的操作类型
type AuditableAction string

const (
	ActionLogin         AuditableAction = "login"
	ActionLogout        AuditableAction = "logout"
	ActionCreate        AuditableAction = "create"
	ActionUpdate        AuditableAction = "update"
	ActionDelete        AuditableAction = "delete"
	ActionView          AuditableAction = "view"
	ActionSearch        AuditableAction = "search"
	ActionExport        AuditableAction = "export"
	ActionImport        AuditableAction = "import"
	ActionAssign        AuditableAction = "assign"
	ActionEscalate      AuditableAction = "escalate"
	ActionResolve       AuditableAction = "resolve"
	ActionClose         AuditableAction = "close"
	ActionReopen        AuditableAction = "reopen"
	ActionComment       AuditableAction = "comment"
	ActionAttachment    AuditableAction = "attachment"
	ActionPermission    AuditableAction = "permission"
	ActionConfiguration AuditableAction = "configuration"
)

// SensitiveResource 敏感资源类型
var SensitiveResources = map[string]bool{
	"users":          true,
	"roles":          true,
	"permissions":    true,
	"configurations": true,
	"audit_logs":     true,
	"system":         true,
}

// AuditMiddleware persists audit logs for operations with enhanced security tracking
func AuditMiddleware(client *ent.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 判断是否需要审计
		if !shouldAuditRequest(c) {
			c.Next()
			return
		}

		// read body safely and restore
		var bodyBytes []byte
		if c.Request.Body != nil {
			var err error
			bodyBytes, err = io.ReadAll(c.Request.Body)
			if err != nil {
				if globalLogger != nil {
					globalLogger.Warnw("failed to read request body for audit", "error", err)
				}
			}
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// collect fields
		rid := c.GetString("request_id")
		tenantID := c.GetInt("tenant_id")
		userID := 0
		username := ""

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

		if v, ok := c.Get("username"); ok {
			if uname, ok := v.(string); ok {
				username = uname
			}
		}

		ip := c.ClientIP()
		path := c.Request.URL.Path
		methodStr := c.Request.Method
		status := c.Writer.Status()
		userAgent := c.Request.UserAgent()

		// limit stored body size to avoid oversized logs
		requestBody := string(bodyBytes)
		if len(requestBody) > 4096 {
			requestBody = requestBody[:4096] + "..."
		}
		// mask sensitive fields
		requestBody = MaskSensitiveFields(requestBody)

		// 确定操作类型和资源
		action, resource, _ := determineActionAndResource(c)

		// 检查是否为敏感操作
		isSensitive := isSensitiveOperation(action, resource, status)

		// 审计记录是企业合规数据，不能使用无确认的 goroutine（进程退出时会静默丢失）。
		auditCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		auditCreate := client.AuditLog.Create().
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
			SetRequestBody(requestBody)

		err := auditCreate.Exec(auditCtx)
		if err != nil && globalLogger != nil {
			globalLogger.Errorw("Failed to save audit log", "error", err)
		}

		// 记录结构化日志
		logFields := []interface{}{
			"audit_type", "user_action",
			"request_id", rid,
			"tenant_id", tenantID,
			"user_id", userID,
			"username", username,
			"action", action,
			"resource", resource,
			"path", path,
			"method", methodStr,
			"status_code", status,
			"client_ip", ip,
			"user_agent", userAgent,
			"latency_ms", duration.Milliseconds(),
			"success", status < 400,
			"sensitive", isSensitive,
		}

		if globalLogger != nil {
			if isSensitive || status >= 400 {
				globalLogger.Warnw("Audit log - Sensitive/Failed operation", logFields...)
			} else {
				globalLogger.Infow("Audit log", logFields...)
			}
		}
	}
}

// shouldAuditRequest 判断是否需要审计请求
func shouldAuditRequest(c *gin.Context) bool {
	// 跳过健康检查和静态资源
	skipPaths := []string{
		"/health",
		"/metrics",
		"/favicon.ico",
		"/static/",
		"/assets/",
	}

	path := c.Request.URL.Path
	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return false
		}
	}

	// 审计所有POST、PUT、DELETE请求
	method := c.Request.Method
	if method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE" {
		return true
	}

	// 审计敏感资源的GET请求
	for resource := range SensitiveResources {
		if strings.Contains(path, resource) {
			return true
		}
	}

	return false
}

// determineActionAndResource 确定操作类型和资源
func determineActionAndResource(c *gin.Context) (string, string, string) {
	method := c.Request.Method
	path := c.Request.URL.Path

	// 解析路径获取资源和ID
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	var resource, resourceID, action string

	// 基本资源识别
	if len(pathParts) >= 3 && pathParts[0] == "api" && pathParts[1] == "v1" {
		resource = pathParts[2]
		if len(pathParts) >= 4 && !isActionPath(pathParts[3]) {
			resourceID = pathParts[3]
		}
	}

	// 根据HTTP方法和路径确定操作
	switch method {
	case "GET":
		if resourceID != "" {
			action = string(ActionView)
		} else {
			action = string(ActionSearch)
		}
	case "POST":
		if strings.Contains(path, "login") {
			action = string(ActionLogin)
			resource = "auth"
		} else if strings.Contains(path, "logout") {
			action = string(ActionLogout)
			resource = "auth"
		} else if strings.Contains(path, "assign") {
			action = string(ActionAssign)
		} else if strings.Contains(path, "escalate") {
			action = string(ActionEscalate)
		} else if strings.Contains(path, "resolve") {
			action = string(ActionResolve)
		} else if strings.Contains(path, "close") {
			action = string(ActionClose)
		} else if strings.Contains(path, "reopen") {
			action = string(ActionReopen)
		} else if strings.Contains(path, "comment") {
			action = string(ActionComment)
		} else if strings.Contains(path, "attachment") {
			action = string(ActionAttachment)
		} else if strings.Contains(path, "export") {
			action = string(ActionExport)
		} else if strings.Contains(path, "import") {
			action = string(ActionImport)
		} else {
			action = string(ActionCreate)
		}
	case "PUT", "PATCH":
		action = string(ActionUpdate)
	case "DELETE":
		action = string(ActionDelete)
	default:
		action = strings.ToLower(method)
	}

	return action, resource, resourceID
}

// isActionPath 判断路径部分是否为操作而非资源ID
func isActionPath(pathPart string) bool {
	actionPaths := []string{
		"assign", "escalate", "resolve", "close", "reopen",
		"comment", "attachment", "export", "import",
		"search", "stats", "analytics", "batch",
	}

	for _, actionPath := range actionPaths {
		if pathPart == actionPath {
			return true
		}
	}

	return false
}

// isSensitiveOperation 判断是否为敏感操作
func isSensitiveOperation(action, resource string, statusCode int) bool {
	// 敏感操作类型
	sensitiveActions := map[string]bool{
		string(ActionDelete):        true,
		string(ActionPermission):    true,
		string(ActionConfiguration): true,
	}

	// 敏感资源操作
	if SensitiveResources[resource] {
		return true
	}

	// 敏感操作类型
	if sensitiveActions[action] {
		return true
	}

	// 失败的登录尝试
	if action == string(ActionLogin) && statusCode >= 400 {
		return true
	}

	return false
}

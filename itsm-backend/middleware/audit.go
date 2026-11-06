package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync/atomic"
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
	if err != nil {
		auditWriteFailures.Add(1)
		if globalLogger != nil {
			globalLogger.Errorw("failed to save login audit", "error", err, "action", action, "tenant_id", tenantID, "user_id", userID)
		}
	}
}

// auditWriteFailures 累计审计落库失败次数（L4）。
//
// 审计写入与业务不同事务，且失败时业务已返回成功；旧实现只在日志里留一行
// "Failed to save audit log"，既无计数、也无请求上下文，运维无法知道「这一小时
// 丢了多少条合规记录」。这里提供进程内计数器供监控/测试读取，并在失败时把
// 完整（已脱敏）事件回退到结构化日志，让外部日志采集至少能补回这一条。
// 刻意**不自动重试**：审计表是 append-only 且无幂等键，一次结果不明的失败
// 重试会制造重复行，比丢一行更难对账。
var auditWriteFailures atomic.Int64

// AuditWriteFailureCount 返回进程启动以来审计落库失败次数。
func AuditWriteFailureCount() int64 { return auditWriteFailures.Load() }

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
	// 变更（及其他资源）生命周期动词，避免所有 POST 被笼统记为 create
	ActionSubmit   AuditableAction = "submit"
	ActionApprove  AuditableAction = "approve"
	ActionReject   AuditableAction = "reject"
	ActionSchedule AuditableAction = "schedule"
	ActionStart    AuditableAction = "start"
	ActionComplete AuditableAction = "complete"
	ActionRollback AuditableAction = "rollback"
	ActionCancel   AuditableAction = "cancel"
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
	// 未注入 ent 客户端时（部分集成测试、无 DB 的部署形态）直接放行：
	// 审计不该让业务请求因为「日志后端缺失」而 panic 或 500。
	if client == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		// 判断是否需要审计（决定要不要提前读 body）。
		// 注意不能在这里早退：L3 要求被鉴权链 Abort 的 401/403 也留痕，
		// 而 shouldAuditRequest 只对写操作与敏感 GET 返回 true。
		auditable := shouldAuditRequest(c)

		// read body safely and restore。
		// 只对 auditable 请求读：避免把大文件上传提前整段读进内存，
		// 绕过排在前面的 RequestSizeMiddleware 的保护。
		var bodyBytes []byte
		if auditable && c.Request.Body != nil {
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
		status := c.Writer.Status()

		// 被 Abort 的失败请求（401 令牌失效/爆破、403 RBAC 拒绝、
		// 跨租户试探、业务拒绝 409）一律补一条，即使 shouldAuditRequest
		// 原本判定为「不必审计」（例如非敏感资源 GET）。
		// 此时 requestBody 为空——不读 body 是刻意的：探测请求的正文可能含
		// 凭据，且大请求体不该在大小限制之前被整段缓存。
		if !auditable && status < 400 {
			return
		}

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

		err := auditCreate.Exec(auditCtx)
		if err != nil {
			auditWriteFailures.Add(1)
			if globalLogger != nil {
				// 回退事件必须带完整上下文与已脱敏正文：业务请求已经成功返回，
				// 数据库里这一行没了，只剩这条日志可供事后补录。
				failureFields := append([]interface{}{}, logFields...)
				failureFields = append(failureFields, "error", err, "request_body", requestBody)
				globalLogger.Errorw("AUDIT_WRITE_FAILED: audit row lost, fallback log emitted", failureFields...)
			}
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

	// 审计敏感资源的GET请求。
	//
	// L6：路由用的是连字符（GET /api/v1/audit-logs），词表用的是下划线
	// （audit_logs），strings.Contains 永不命中——「查看审计日志」这个最该留痕
	// 的动作反而零记录。先归一化再匹配；导出/下载类 GET 同样必须留痕
	// （旧规则只认写方法，GET /xxx/export 完全不审计）。
	normalized := strings.ReplaceAll(path, "-", "_")
	for resource := range SensitiveResources {
		if strings.Contains(normalized, resource) {
			return true
		}
	}
	for _, keyword := range []string{"export", "download"} {
		if strings.Contains(normalized, keyword) {
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
		if strings.Contains(path, "export") || strings.Contains(path, "download") {
			action = string(ActionExport)
		} else if resourceID != "" {
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
		} else if strings.Contains(path, "submit") {
			action = string(ActionSubmit)
		} else if strings.Contains(path, "approve") {
			action = string(ActionApprove)
		} else if strings.Contains(path, "reject") {
			action = string(ActionReject)
		} else if strings.Contains(path, "schedule") {
			action = string(ActionSchedule)
		} else if strings.Contains(path, "/start") || strings.HasSuffix(path, "/start") {
			// 避免与 /api/v1/start* 等无意义路径误匹配；本项目变更 /start 才是生命周期动作
			action = string(ActionStart)
		} else if strings.Contains(path, "complete") {
			action = string(ActionComplete)
		} else if strings.Contains(path, "rollback") || strings.Contains(path, "rolled_back") {
			action = string(ActionRollback)
		} else if strings.Contains(path, "cancel") {
			action = string(ActionCancel)
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
		"submit", "approve", "reject", "schedule", "start", "complete",
		"rollback", "cancel",
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

package middleware

import (
	"fmt"
	"itsm-backend/common"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorResponse 统一错误响应结构
type ErrorResponse struct {
	Code       string    `json:"code"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	RequestID  string    `json:"request_id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Path       string    `json:"path"`
	Method     string    `json:"method"`
	StackTrace string    `json:"stack_trace,omitempty"`
}

// ErrorType 错误类型枚举
type ErrorType string

const (
	ErrorTypeValidation    ErrorType = "VALIDATION_ERROR"
	ErrorTypeNotFound      ErrorType = "NOT_FOUND"
	ErrorTypeUnauthorized  ErrorType = "UNAUTHORIZED"
	ErrorTypeForbidden     ErrorType = "FORBIDDEN"
	ErrorTypeConflict      ErrorType = "CONFLICT"
	ErrorTypeInternal      ErrorType = "INTERNAL_ERROR"
	ErrorTypeService       ErrorType = "SERVICE_ERROR"
	ErrorTypeDatabase      ErrorType = "DATABASE_ERROR"
	ErrorTypeNetwork       ErrorType = "NETWORK_ERROR"
	ErrorTypeTimeout       ErrorType = "TIMEOUT_ERROR"
	ErrorTypeRateLimit     ErrorType = "RATE_LIMIT_ERROR"
	ErrorTypeQuotaExceeded ErrorType = "QUOTA_EXCEEDED"
)

// AppError 应用程序错误类型
type AppError struct {
	Type      ErrorType `json:"type"`
	Code      int       `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Cause     error     `json:"-"`
	RequestID string    `json:"-"`
	Sensitive bool      `json:"-"` // 是否为敏感错误
	Retryable bool      `json:"retryable"`
}

func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s - %s", e.Type, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewAppError 创建应用错误
func NewAppError(errorType ErrorType, code int, message, details string, cause error) *AppError {
	return &AppError{
		Type:      errorType,
		Code:      code,
		Message:   message,
		Details:   details,
		Cause:     cause,
		Retryable: isRetryableError(errorType),
	}
}

// 预定义错误创建函数
func NewValidationError(message, details string, cause error) *AppError {
	return NewAppError(ErrorTypeValidation, common.ParamErrorCode, message, details, cause)
}

func NewNotFoundError(resource string, cause error) *AppError {
	return NewAppError(ErrorTypeNotFound, common.NotFoundErrorCode, fmt.Sprintf("%s未找到", resource), "", cause)
}

func NewUnauthorizedError(message string, cause error) *AppError {
	return NewAppError(ErrorTypeUnauthorized, common.AuthErrorCode, message, "", cause)
}

func NewForbiddenError(message string, cause error) *AppError {
	return NewAppError(ErrorTypeForbidden, common.ForbiddenErrorCode, message, "", cause)
}

func NewConflictError(message, details string, cause error) *AppError {
	return NewAppError(ErrorTypeConflict, common.ParamErrorCode, message, details, cause)
}

func NewInternalError(message, details string, cause error) *AppError {
	return NewAppError(ErrorTypeInternal, common.InternalErrorCode, message, details, cause)
}

func NewServiceError(service, operation string, cause error) *AppError {
	return NewAppError(ErrorTypeService, common.InternalErrorCode, fmt.Sprintf("%s服务错误", service), operation, cause)
}

func NewDatabaseError(operation string, cause error) *AppError {
	return NewAppError(ErrorTypeDatabase, common.InternalErrorCode, fmt.Sprintf("数据库操作失败: %s", operation), "", cause)
}

func NewRateLimitError(message string) *AppError {
	return NewAppError(ErrorTypeRateLimit, 429, message, "", nil)
}

func NewQuotaExceededError(resource string) *AppError {
	return NewAppError(ErrorTypeQuotaExceeded, common.ForbiddenErrorCode, fmt.Sprintf("%s配额已超出", resource), "", nil)
}

// isRetryableError 判断错误是否可重试
func isRetryableError(errorType ErrorType) bool {
	retryableTypes := map[ErrorType]bool{
		ErrorTypeNetwork:       true,
		ErrorTypeTimeout:       true,
		ErrorTypeDatabase:      true,
		ErrorTypeService:       true,
		ErrorTypeInternal:      false,
		ErrorTypeValidation:    false,
		ErrorTypeNotFound:      false,
		ErrorTypeUnauthorized:  false,
		ErrorTypeForbidden:     false,
		ErrorTypeConflict:      false,
		ErrorTypeRateLimit:     true,
		ErrorTypeQuotaExceeded: false,
	}

	return retryableTypes[errorType]
}

// ErrorHandlerMiddleware 全局错误处理中间件
func ErrorHandlerMiddleware(logger *zap.SugaredLogger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		var err error

		switch x := recovered.(type) {
		case string:
			err = fmt.Errorf("%s", x)
		case error:
			err = x
		default:
			err = fmt.Errorf("unknown error: %v", x)
		}

		handleError(c, err, logger, true)
	})
}

// HandleError 处理错误的核心函数
func HandleError(c *gin.Context, err error, logger *zap.SugaredLogger) {
	handleError(c, err, logger, false)
}

func handleError(c *gin.Context, err error, logger *zap.SugaredLogger, isPanic bool) {
	if err == nil {
		return
	}

	// 获取请求ID
	requestID := c.GetString("request_id")
	if requestID == "" {
		requestID = generateRequestID()
	}

	// 构造错误响应
	var response *ErrorResponse
	var statusCode int

	// 根据错误类型构造响应
	switch e := err.(type) {
	case *AppError:
		response = &ErrorResponse{
			Code:      string(e.Type),
			Message:   e.Message,
			Details:   e.Details,
			RequestID: requestID,
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
			Method:    c.Request.Method,
		}

		// 根据错误类型设置HTTP状态码
		switch e.Type {
		case ErrorTypeValidation, ErrorTypeConflict:
			statusCode = http.StatusBadRequest
		case ErrorTypeNotFound:
			statusCode = http.StatusNotFound
		case ErrorTypeUnauthorized:
			statusCode = http.StatusUnauthorized
		case ErrorTypeForbidden:
			statusCode = http.StatusForbidden
		case ErrorTypeRateLimit:
			statusCode = http.StatusTooManyRequests
		default:
			statusCode = http.StatusInternalServerError
		}

		// 非生产环境下添加堆栈信息
		if gin.IsDebugging() && !e.Sensitive {
			response.StackTrace = string(debug.Stack())
		}

	default:
		// 处理标准错误
		response = &ErrorResponse{
			Code:      string(ErrorTypeInternal),
			Message:   "内部服务器错误",
			RequestID: requestID,
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
			Method:    c.Request.Method,
		}
		statusCode = http.StatusInternalServerError

		// 在调试模式下显示详细错误信息
		if gin.IsDebugging() {
			response.Details = err.Error()
			response.StackTrace = string(debug.Stack())
		}
	}

	// 记录错误日志
	logError(c, err, logger, isPanic, requestID)

	// 如果客户端已经接受了响应，则无法再次发送
	if c.Writer.Written() {
		return
	}

	// 发送错误响应
	c.JSON(statusCode, response)

	// 终止请求处理
	c.Abort()
}

// logError 记录错误日志
func logError(c *gin.Context, err error, logger *zap.SugaredLogger, isPanic bool, requestID string) {
	// 构建日志字段
	fields := []interface{}{
		"request_id", requestID,
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"error", err.Error(),
		"user_agent", c.Request.UserAgent(),
		"client_ip", c.ClientIP(),
	}

	// 添加用户和租户信息（如果存在）
	if userID := c.GetInt("user_id"); userID > 0 {
		fields = append(fields, "user_id", userID)
	}
	if tenantID := c.GetInt("tenant_id"); tenantID > 0 {
		fields = append(fields, "tenant_id", tenantID)
	}

	// 添加查询参数
	if len(c.Request.URL.RawQuery) > 0 {
		fields = append(fields, "query", c.Request.URL.RawQuery)
	}

	// 根据错误类型选择日志级别
	switch {
	case isPanic:
		logger.Errorw("系统panic", append(fields, "stack_trace", string(debug.Stack()))...)
	case strings.Contains(err.Error(), "未找到") || strings.Contains(err.Error(), "not found"):
		logger.Warnw("资源未找到", fields...)
	case strings.Contains(err.Error(), "权限") || strings.Contains(err.Error(), "unauthorized"):
		logger.Warnw("权限错误", fields...)
	case strings.Contains(err.Error(), "验证") || strings.Contains(err.Error(), "validation"):
		logger.Warnw("验证错误", fields...)
	default:
		logger.Errorw("系统错误", fields...)
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ErrorToAppError 将各种错误类型转换为AppError
func ErrorToAppError(err error) *AppError {
	if err == nil {
		return nil
	}

	// 如果已经是AppError，直接返回
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	// 根据错误消息推断错误类型
	errMsg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "未找到"):
		return NewNotFoundError("资源", err)
	case strings.Contains(errMsg, "unauthorized") || strings.Contains(errMsg, "未授权"):
		return NewUnauthorizedError("未授权访问", err)
	case strings.Contains(errMsg, "forbidden") || strings.Contains(errMsg, "禁止"):
		return NewForbiddenError("访问被禁止", err)
	case strings.Contains(errMsg, "conflict") || strings.Contains(errMsg, "冲突"):
		return NewConflictError("资源冲突", "", err)
	case strings.Contains(errMsg, "validation") || strings.Contains(errMsg, "验证"):
		return NewValidationError("数据验证失败", "", err)
	case strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "超时"):
		return NewAppError(ErrorTypeTimeout, common.InternalErrorCode, "请求超时", "", err)
	case strings.Contains(errMsg, "database") || strings.Contains(errMsg, "数据库"):
		return NewDatabaseError("数据库操作", err)
	default:
		return NewInternalError("内部服务器错误", "", err)
	}
}

// ValidateMiddleware 参数验证中间件
func ValidateMiddleware(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查Content-Type
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.GetHeader("Content-Type")
			if contentType == "" {
				contentType = c.GetHeader("content-type")
			}

			if contentType != "" &&
				!strings.Contains(contentType, "application/json") &&
				!strings.Contains(contentType, "multipart/form-data") &&
				!strings.Contains(contentType, "application/x-www-form-urlencoded") {

				err := NewValidationError("不支持的内容类型",
					fmt.Sprintf("期望: application/json, 实际: %s", contentType), nil)
				HandleError(c, err, logger)
				return
			}
		}

		// 添加请求ID
		requestID := c.GetString("request_id")
		if requestID == "" {
			requestID = generateRequestID()
			c.Set("request_id", requestID)
		}

		c.Next()
	}
}

// LogMiddleware 请求日志中间件
func LogMiddleware(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)

		// 获取状态码
		statusCode := c.Writer.Status()

		// 获取客户端IP
		clientIP := c.ClientIP()

		// 构建日志字段
		fields := []interface{}{
			"request_id", c.GetString("request_id"),
			"status_code", statusCode,
			"latency", latency.String(),
			"client_ip", clientIP,
			"method", c.Request.Method,
			"path", path,
			"user_agent", c.Request.UserAgent(),
		}

		// 添加查询参数
		if raw != "" {
			fields = append(fields, "query", raw)
		}

		// 添加响应大小
		if c.Writer.Size() > 0 {
			fields = append(fields, "response_size", c.Writer.Size())
		}

		// 添加用户和租户信息
		if userID := c.GetInt("user_id"); userID > 0 {
			fields = append(fields, "user_id", userID)
		}
		if tenantID := c.GetInt("tenant_id"); tenantID > 0 {
			fields = append(fields, "tenant_id", tenantID)
		}

		// 根据状态码选择日志级别
		switch {
		case statusCode >= 500:
			logger.Errorw("请求处理失败", fields...)
		case statusCode >= 400:
			logger.Warnw("请求错误", fields...)
		default:
			logger.Infow("请求处理成功", fields...)
		}
	}
}

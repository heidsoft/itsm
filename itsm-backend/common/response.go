package common

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 响应码定义
const (
	SuccessCode             = 0
	ParamErrorCode          = 1001
	ValidationError         = 1002
	AuthFailedCode          = 2001
	UnauthorizedCode        = 2002
	ForbiddenCode           = 2003
	NotFoundCode            = 4004
	BadRequestCode          = 4000
	ConflictCode            = 4090 // 版本冲突
	UnprocessableEntityCode = 4220
	InternalErrorCode       = 5001
	ServerErrorCode         = InternalErrorCode
	ServiceUnavailableCode  = 5003
	// P2-6 AI 工具 RBAC 校验
	ToolPermissionDeniedCode = 2004 // 工具权限不足
	UnknownToolCode          = 2005 // 未知工具

	// Aliases for compatibility
	NotFoundErrorCode  = NotFoundCode
	AuthErrorCode      = AuthFailedCode
	ForbiddenErrorCode = ForbiddenCode
)

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "success",
		Data:    data,
	})
}

// Fail 失败响应
func Fail(c *gin.Context, code int, message string) {
	statusCode := http.StatusOK
	switch code {
	case ParamErrorCode, ValidationError, BadRequestCode:
		statusCode = http.StatusBadRequest
	case AuthFailedCode, UnauthorizedCode:
		// 对齐审计 P0 #3:2001/2002 都映射到 401,避免未授权仍然 200。
		statusCode = http.StatusUnauthorized
	case ForbiddenCode, ToolPermissionDeniedCode:
		// 对齐审计 P0 #3:2003/2004 都映射到 403,工具 RBAC 拒绝同样不允许 200。
		statusCode = http.StatusForbidden
	case NotFoundCode, UnknownToolCode:
		// 对齐审计 P0 #3:未知工具(2005)与未找到资源(4004)都映射到 404。
		statusCode = http.StatusNotFound
	case ConflictCode:
		statusCode = http.StatusConflict
	case UnprocessableEntityCode:
		statusCode = http.StatusUnprocessableEntity
	case InternalErrorCode:
		statusCode = http.StatusInternalServerError
	case ServiceUnavailableCode:
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, Response{
		Code:    code,
		Message: message,
	})
	c.Abort()
}

// FailWithData 失败响应（带数据）
func FailWithData(c *gin.Context, code int, message string, data interface{}) {
	statusCode := http.StatusOK
	switch code {
	case ParamErrorCode, ValidationError, BadRequestCode:
		statusCode = http.StatusBadRequest
	case AuthFailedCode, UnauthorizedCode:
		statusCode = http.StatusUnauthorized
	case ForbiddenCode, ToolPermissionDeniedCode:
		statusCode = http.StatusForbidden
	case NotFoundCode, UnknownToolCode:
		statusCode = http.StatusNotFound
	case ConflictCode:
		statusCode = http.StatusConflict
	case UnprocessableEntityCode:
		statusCode = http.StatusUnprocessableEntity
	case InternalErrorCode:
		statusCode = http.StatusInternalServerError
	case ServiceUnavailableCode:
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
	c.Abort()
}

// Conflict 版本冲突响应
func Conflict(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusConflict, Response{
		Code:    ConflictCode,
		Message: message,
		Data:    data,
	})
}

// SuccessWithMessage 带自定义消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: message,
		Data:    data,
	})
}

// ParamError 参数错误响应
func ParamError(c *gin.Context, message string) {
	Fail(c, ParamErrorCode, message)
}

// ValidationErrorResponse 验证错误响应
func ValidationErrorResponse(c *gin.Context, message string) {
	Fail(c, ValidationError, message)
}

// AuthFailed 认证失败响应
func AuthFailed(c *gin.Context, message string) {
	Fail(c, AuthFailedCode, message)
}

// Forbidden 权限不足响应
func Forbidden(c *gin.Context, message string) {
	Fail(c, ForbiddenCode, message)
}

// NotFound 资源不存在响应
func NotFound(c *gin.Context, message string) {
	Fail(c, NotFoundCode, message)
}

// InternalError 内部错误响应（仅接收面向用户的安全消息；绝不要传入 err.Error()）
func InternalError(c *gin.Context, message string) {
	Fail(c, InternalErrorCode, message)
}

// InternalErrorf 内部错误响应（格式化面向用户的安全消息）
func InternalErrorf(c *gin.Context, format string, args ...any) {
	Fail(c, InternalErrorCode, fmt.Sprintf(format, args...))
}

// FailWithErr 是 Fail 的安全包装：
//   - rawErr 会被 zap.S().Error 记录到服务端日志（包含 request_id / method / path）。
//   - publicMsg 是真正返回给客户端的内容，绝不包含 err.Error()。
//
// 用法：
//
//	if err := svc.DoSomething(ctx); err != nil {
//	    common.FailWithErr(c, err, "operation failed")
//	    return
//	}
//
// 这样既保留了诊断所需的堆栈与驱动层错误信息（pq: ... / ent: ...），
// 又不把内部错误字符串原样透出给客户端。
func FailWithErr(c *gin.Context, rawErr error, publicMsg string) {
	if rawErr != nil {
		zap.S().Errorw(
			"handler returned internal error",
			"err", rawErr.Error(),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"public_message", publicMsg,
		)
	}
	Fail(c, InternalErrorCode, publicMsg)
}

// ParamErrorWithErr 与 FailWithErr 类似，但状态码是 1001 参数错误。
// 适合 bind / validate 阶段捕获到 driver 错误（如 unique 冲突被 bind 解析为校验错）
// 的场景：仍然要记录原始错误，但客户端消息保持稳定。
func ParamErrorWithErr(c *gin.Context, rawErr error, publicMsg string) {
	if rawErr != nil {
		zap.S().Errorw(
			"handler returned param error",
			"err", rawErr.Error(),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"public_message", publicMsg,
		)
	}
	Fail(c, ParamErrorCode, publicMsg)
}

// BadRequestWithErr 与 ParamErrorWithErr 类似但保持 BadRequestCode(4000) 的语义。
// 用于需要「请求语义错误」状态码的场景（例如必填字段缺失时的 400 Bad Request）。
// 与 ParamErrorCode(1001) 在 HTTP 状态码层面等价，都返回 400，区别仅在应用层
// code 字段：4000 表示请求结构 OK 但语义不被接受，1001 表示参数格式/绑定失败。
func BadRequestWithErr(c *gin.Context, rawErr error, publicMsg string) {
	if rawErr != nil {
		zap.S().Errorw(
			"handler returned bad request error",
			"err", rawErr.Error(),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"public_message", publicMsg,
		)
	}
	Fail(c, BadRequestCode, publicMsg)
}

// NotFoundWithErr 与 FailWithErr 类似但保持 NotFoundCode(4004) 的语义。
// 用于「资源不存在」场景。原常见 leak 模式是 common.NotFound(c, err.Error())
// 直接返回驱动层错误，该 helper 保留 4004 但隔离原始错误。
func NotFoundWithErr(c *gin.Context, rawErr error, publicMsg string) {
	if rawErr != nil {
		zap.S().Errorw(
			"handler returned not found error",
			"err", rawErr.Error(),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"public_message", publicMsg,
		)
	}
	Fail(c, NotFoundCode, publicMsg)
}

// BindValidationError 是 bind/validate 阶段的安全错误包装：
//   - 如果 err 是 validator.ValidationErrors，则仅提取字段名+规则（如 "Title is required"），
//     不输出 err.Error() 的原始堆栈 / driver 信息。
//   - 其他类型的错误走 generic 兜底（如 JSON 反序列化失败）。
//
// 适用场景：handler 中
//
//	if err := c.ShouldBindJSON(&req); err != nil {
//	    common.BindValidationError(c, err, "invalid request body")
//	    return
//	}
//
// 这样既向客户端暴露了「具体哪个字段需要修正」这种 UX 必要信息，
// 又不把 go-playground/validator 的 Key/Field/Tag 字符串原样泄露。
func BindValidationError(c *gin.Context, rawErr error, fallbackMsg string) {
	if rawErr != nil {
		zap.S().Errorw(
			"handler bind/validation failed",
			"err", rawErr.Error(),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"fallback_message", fallbackMsg,
		)
	}
	msg := fallbackMsg
	if ve, ok := rawErr.(validator.ValidationErrors); ok && len(ve) > 0 {
		parts := make([]string, 0, len(ve))
		for _, fe := range ve {
			field := fe.Field()
			if field == "" {
				field = fe.StructField()
			}
			if field != "" {
				parts = append(parts, fmt.Sprintf("%s is %s", field, humanizeTag(fe.Tag())))
			}
		}
		if len(parts) > 0 {
			msg = strings.Join(parts, "; ")
		}
	}
	Fail(c, ParamErrorCode, msg)
}

// humanizeTag 将 validator 的 tag 翻译成可读短语（如 required → required），
// 用于 BindValidationError 的字段级错误消息。仅返回面向用户的字符串，
// 不暴露 validator 内部机制。
func humanizeTag(tag string) string {
	switch tag {
	case "required":
		return "required"
	case "min":
		return "too short"
	case "max":
		return "too long"
	case "email":
		return "not a valid email"
	case "gte":
		return "below minimum"
	case "lte":
		return "above maximum"
	case "oneof":
		return "not in allowed values"
	default:
		return tag
	}
}

// SuccessWithList 返回列表数据的成功响应
func SuccessWithList(c *gin.Context, data interface{}, total int, page int, pageSize int) {
	listResponse := NewListResponse(data, NewPaginationResponse(page, pageSize, int64(total)))
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "success",
		Data:    listResponse,
	})
}

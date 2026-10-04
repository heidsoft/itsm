package common

import (
	"errors"
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

// codeStatusPairs 是「应用业务码 ⇄ HTTP 状态码」的唯一事实来源。
// 此前 Fail 与 FailWithData 各写一份正向 switch（2026-10-04 实测两份映射内容一致，但只有
// Fail 那份带审计 P0 #3 的理由注释），而 statusToAppCode 又独立写了第三份反向映射，
// 反向那份缺 ServiceUnavailable(503)：AppError 明确带 503 语义时会被降级成 5001/500。
// 两个方向都从这一张表读出，正向与反向不可能再各自漂移。
//
// 同组内的顺序即反向查找（statusToAppCode）的优先级：把契约测试已锁定的规范码放前面
// （400→4000、401→2002、403→2003、404→4004）。
var codeStatusPairs = []struct {
	code   int
	status int
}{
	{BadRequestCode, http.StatusBadRequest},
	{ParamErrorCode, http.StatusBadRequest},
	{ValidationError, http.StatusBadRequest},
	{UnauthorizedCode, http.StatusUnauthorized},
	{AuthFailedCode, http.StatusUnauthorized},
	{ForbiddenCode, http.StatusForbidden},
	{ToolPermissionDeniedCode, http.StatusForbidden},
	{NotFoundCode, http.StatusNotFound},
	{UnknownToolCode, http.StatusNotFound},
	{ConflictCode, http.StatusConflict},
	{UnprocessableEntityCode, http.StatusUnprocessableEntity},
	{InternalErrorCode, http.StatusInternalServerError},
	{ServiceUnavailableCode, http.StatusServiceUnavailable},
}

// statusForCode 是「应用业务码 → HTTP 状态码」方向的唯一入口。
// 未登记的业务码返回 200：这是既有 Fail 的兜底语义（未定义码不猜 HTTP 语义），
// 新业务码必须显式登记，不能靠 default 分支蒙混。
func statusForCode(code int) int {
	for _, p := range codeStatusPairs {
		if p.code == code {
			return p.status
		}
	}
	return http.StatusOK
}

// statusToAppCode 是「HTTP 状态码 → 应用业务码」方向的唯一入口，供 AppError.HTTPStatus
// 反查业务码。未登记的状态（如 408 请求超时、429 限流）目前在 common 码体系里没有对应
// 业务码，按内部错误处理；需要支持时要先评审新增业务码，而不是在此处随意映射。
func statusToAppCode(httpStatus int) int {
	for _, p := range codeStatusPairs {
		if p.status == httpStatus {
			return p.code
		}
	}
	return InternalErrorCode
}

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
	c.JSON(statusForCode(code), Response{
		Code:    code,
		Message: message,
	})
	c.Abort()
}

// FailWithData 失败响应（带数据）
func FailWithData(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(statusForCode(code), Response{
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
//   - 业务码由 classifyError 决定：领域错误（AppError/BusinessError/版本冲突）不再
//     被兜底成 5001，而是映射回 4004/4090/2003/…；只有真正未知的错误才走 5001。
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
//
// 注意：这里保留 handler 传入的 publicMsg 而不是采用领域错误自带文案。领域文案可能是
// 英文构造串（"Ticket not found"），透传会让中文界面出现混合文案；需要领域文案时用
// RespondError。
func FailWithErr(c *gin.Context, rawErr error, publicMsg string) {
	code, _, _ := classifyError(rawErr)
	if rawErr != nil {
		zap.S().Errorw(
			"handler returned error",
			"err", rawErr.Error(),
			"error_class", code,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"public_message", publicMsg,
		)
	}
	writeClassified(c, rawErr, code, publicMsg)
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

// RespondError 是 handler 的统一错误出口：把领域错误转成统一响应结构。
// 分类由 classifyError 唯一决定，消息权威在本入口保留领域错误自带文案
// （构造/哨兵时已是面向用户的安全文案）；未分类错误走 FailWithErr（5001 + fallbackMsg）。
//
// 用法：
//
//	if err := svc.Create(ctx, ...); err != nil {
//	    common.RespondError(c, err, "创建失败")
//	    return
//	}
//
// 该 helper 收敛了此前各域 handler 手写 switch 的 AppError 映射，是
// "领域哨兵错误 + handler 统一分流"范式的落点，避免业务拒绝被兜底成 500。
func RespondError(c *gin.Context, err error, fallbackMsg string) {
	if err == nil {
		return
	}
	code, safeMsg, ok := classifyError(err)
	if !ok {
		FailWithErr(c, err, fallbackMsg)
		return
	}
	// safeMsg 为空时（业务码显式但文案缺失）退回 handler 的兜底文案，绝不透出 err.Error()。
	if safeMsg == "" {
		safeMsg = fallbackMsg
	}
	writeClassified(c, err, code, safeMsg)
}

// classifyError 是「领域错误 → 应用业务码 + 领域安全文案」的唯一所有者。
// 三条匹配链均用 errors.As，兼容 fmt.Errorf 包装：
//  1. *AppError：按 HTTPStatus 反查业务码（400→4000、401→2002、403→2003、404→4004、
//     409→4090、422→4220、其余→5001），文案取 AppError.Message。
//  2. *BusinessError：原样业务码与文案（incident/approval 域哨兵错误模式）。
//  3. *VersionConflictError：ConflictCode(4090)，文案取自带的版本说明。
//
// ok=false 表示未分类（驱动层错误、第三方错误等），调用方必须按内部错误处理，
// 且不得透出原始错误字符串。
func classifyError(err error) (code int, safeMsg string, ok bool) {
	if err == nil {
		return InternalErrorCode, "", false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return statusToAppCode(appErr.HTTPStatus), appErr.Message, true
	}
	var bizErr *BusinessError
	if errors.As(err, &bizErr) {
		return bizErr.Code, bizErr.Message, true
	}
	var conflictErr *VersionConflictError
	if errors.As(err, &conflictErr) {
		return ConflictCode, conflictErr.Error(), true
	}
	return InternalErrorCode, "", false
}

// writeClassified 是 FailWithErr 与 RespondError 共用的落点：把已分类的业务码写成响应。
// 版本冲突需要携带 {resourceId,currentVersion,serverVersion} 载荷，让乐观锁冲突可被
// 客户端直接处理，而不是只有一段文字；其余分类走统一 Fail。
func writeClassified(c *gin.Context, rawErr error, code int, message string) {
	var conflictErr *VersionConflictError
	if code == ConflictCode && errors.As(rawErr, &conflictErr) {
		FailWithData(c, ConflictCode, message, gin.H{
			"resourceId":     conflictErr.ResourceID,
			"currentVersion": conflictErr.CurrentVersion,
			"serverVersion":  conflictErr.ServerVersion,
		})
		return
	}
	Fail(c, code, message)
}

// SuccessWithList 返回列表数据的成功响应
func SuccessWithList(c *gin.Context, items interface{}, total int, page int, pageSize int) {
	listResponse := NewListResponse(items, NewPaginationResponse(page, pageSize, int64(total)))
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "success",
		Data:    listResponse,
	})
}

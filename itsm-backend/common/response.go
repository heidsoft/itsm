package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 响应码定义
const (
	SuccessCode       = 0
	ParamErrorCode    = 1001
	ValidationError   = 1002
	AuthFailedCode    = 2001
	UnauthorizedCode  = 2002
	ForbiddenCode     = 2003
	NotFoundCode      = 4004
	BadRequestCode    = 4000
	InternalErrorCode = 5001
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
	case AuthFailedCode:
		statusCode = http.StatusUnauthorized
	case ForbiddenCode:
		statusCode = http.StatusForbidden
	case NotFoundCode:
		statusCode = http.StatusNotFound
	case InternalErrorCode:
		statusCode = http.StatusInternalServerError
	}

	c.JSON(statusCode, Response{
		Code:    code,
		Message: message,
	})
}

// FailWithData 失败响应（带数据）
func FailWithData(c *gin.Context, code int, message string, data interface{}) {
	statusCode := http.StatusOK
	switch code {
	case AuthFailedCode:
		statusCode = http.StatusUnauthorized
	case ForbiddenCode:
		statusCode = http.StatusForbidden
	case NotFoundCode:
		statusCode = http.StatusNotFound
	case InternalErrorCode:
		statusCode = http.StatusInternalServerError
	}

	c.JSON(statusCode, Response{
		Code:    code,
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

// InternalError 内部错误响应
func InternalError(c *gin.Context, message string) {
	Fail(c, InternalErrorCode, message)
}

// SuccessWithList 返回列表数据的成功响应
func SuccessWithList(c *gin.Context, data interface{}, total int, page int, pageSize int) {
	listResponse := NewListResponse(data, NewPaginationResponse(total, page, int64(pageSize)))
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "success",
		Data:    listResponse,
	})
}

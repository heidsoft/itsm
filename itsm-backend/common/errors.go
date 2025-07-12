package common

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BusinessError 业务错误
type BusinessError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func (e *BusinessError) Error() string {
	return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Detail)
}

// NewBusinessError 创建业务错误
func NewBusinessError(code int, message, detail string) *BusinessError {
	return &BusinessError{
		Code:    code,
		Message: message,
		Detail:  detail,
	}
}

// ErrorHandler 全局错误处理中间件
func ErrorHandler(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 处理panic
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			logger.Errorw("Request error", "error", err, "path", c.Request.URL.Path)

			if businessErr, ok := err.(*BusinessError); ok {
				Fail(c, businessErr.Code, businessErr.Message)
			} else {
				Fail(c, InternalErrorCode, "内部服务器错误")
			}
		}
	}
}

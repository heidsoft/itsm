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

// VersionConflictError 版本冲突错误（乐观锁）
type VersionConflictError struct {
	ResourceName   string // 资源名称（如 "工单"、"事件"）
	ResourceID     int    // 资源ID
	CurrentVersion int    // 客户端持有的版本
	ServerVersion  int    // 服务器当前版本
}

func (e *VersionConflictError) Error() string {
	return fmt.Sprintf(
		"%s(ID:%d) 版本冲突：客户端版本 %d，服务器版本 %d",
		e.ResourceName, e.ResourceID, e.CurrentVersion, e.ServerVersion,
	)
}

// NewVersionConflictError 创建版本冲突错误
func NewVersionConflictError(resourceName string, resourceID, currentVersion, serverVersion int) *VersionConflictError {
	return &VersionConflictError{
		ResourceName:   resourceName,
		ResourceID:     resourceID,
		CurrentVersion: currentVersion,
		ServerVersion:  serverVersion,
	}
}

// IsVersionConflictError 检查是否为版本冲突错误
func IsVersionConflictError(err error) bool {
	_, ok := err.(*VersionConflictError)
	return ok
}

// ErrorHandler 全局错误处理中间件
func ErrorHandler(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 处理panic
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			logger.Errorw("Request error", "error", err, "path", c.Request.URL.Path)

			// 分类唯一所有者是 classifyError；此前的类型断言链只认顶层错误，
			// 被 fmt.Errorf 包装过的业务拒绝会掉进 500 兜底。
			if code, safeMsg, ok := classifyError(err); ok {
				writeClassified(c, err, code, safeMsg)
			} else {
				Fail(c, InternalErrorCode, "内部服务器错误")
			}
		}
	}
}

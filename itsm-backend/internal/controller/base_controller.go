package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"itsm-backend/common"
	"itsm-backend/middleware"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BaseController 通用控制器基类，减少重复代码
type BaseController struct {
	logger *zap.SugaredLogger
}

// NewBaseController 创建基础控制器
func NewBaseController(logger *zap.SugaredLogger) *BaseController {
	return &BaseController{logger: logger}
}

// RequestHandler 通用请求处理器接口
type RequestHandler[T any, R any] interface {
	Handle(ctx context.Context, req T, tenantID int, userID int) (*R, error)
}

// BindAndValidate 绑定和验证请求参数
func (bc *BaseController) BindAndValidate(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		bc.logger.Errorw("参数绑定失败", "error", err, "path", c.Request.URL.Path)
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return false
	}
	return true
}

// GetContextParams 获取上下文参数（租户ID、用户ID）
func (bc *BaseController) GetContextParams(c *gin.Context) (int, int) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		// 尝试从TenantContext获取
		if tenantCtx, ok := middleware.GetTenantContext(c); ok {
			tenantID = tenantCtx.TenantID
		}
	}

	userID := c.GetInt("user_id")
	if userID == 0 {
		// 尝试从中间件获取
		if uid, err := middleware.GetUserID(c); err == nil {
			userID = uid
		}
	}

	return tenantID, userID
}

// ParseIDParam 解析ID参数
func (bc *BaseController) ParseIDParam(c *gin.Context, paramName string) (int, bool) {
	idStr := c.Param(paramName)
	if idStr == "" {
		bc.logger.Errorw("缺少ID参数", "param", paramName)
		common.Fail(c, common.ParamErrorCode, fmt.Sprintf("缺少%s参数", paramName))
		return 0, false
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		bc.logger.Errorw("无效的ID参数", "param", paramName, "value", idStr, "error", err)
		common.Fail(c, common.ParamErrorCode, fmt.Sprintf("无效的%s参数", paramName))
		return 0, false
	}

	return id, true
}

// HandleServiceError 处理服务层错误
func (bc *BaseController) HandleServiceError(c *gin.Context, err error, operation string) bool {
	if err == nil {
		return true
	}

	bc.logger.Errorw("服务层错误", "operation", operation, "error", err)

	// 根据错误类型返回不同的错误码
	errMsg := err.Error()

	// 参数验证错误
	if strings.Contains(errMsg, "不能为空") ||
		strings.Contains(errMsg, "无效") ||
		strings.Contains(errMsg, "格式错误") {
		common.Fail(c, common.ParamErrorCode, errMsg)
		return false
	}

	// 数据已存在错误
	if strings.Contains(errMsg, "已存在") ||
		strings.Contains(errMsg, "重复") {
		common.Fail(c, common.ParamErrorCode, errMsg)
		return false
	}

	// 权限错误
	if strings.Contains(errMsg, "权限") ||
		strings.Contains(errMsg, "禁止") ||
		strings.Contains(errMsg, "无权限") {
		common.Fail(c, common.ForbiddenErrorCode, errMsg)
		return false
	}

	// 资源未找到
	if strings.Contains(errMsg, "未找到") ||
		strings.Contains(errMsg, "不存在") {
		common.Fail(c, common.NotFoundErrorCode, errMsg)
		return false
	}

	// 默认内部服务器错误
	common.Fail(c, common.InternalErrorCode, errMsg)
	return false
}

// Success 成功响应
func (bc *BaseController) Success(c *gin.Context, data interface{}) {
	common.Success(c, data)
}

// SuccessWithMessage 带消息的成功响应
func (bc *BaseController) SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code":      common.SuccessCode,
		"message":   message,
		"data":      data,
		"timestamp": common.GetTimestamp(),
	})
}

// PaginatedRequest 分页请求参数
type PaginatedRequest struct {
	Page     int    `json:"page" form:"page" binding:"omitempty,min=1"`
	PageSize int    `json:"page_size" form:"page_size" binding:"omitempty,min=1,max=100"`
	Search   string `json:"search" form:"search"`
	SortBy   string `json:"sort_by" form:"sort_by"`
	SortDesc bool   `json:"sort_desc" form:"sort_desc"`
}

// ParsePaginationRequest 解析分页参数
func (bc *BaseController) ParsePaginationRequest(c *gin.Context) PaginatedRequest {
	var req PaginatedRequest

	// 从查询参数获取
	if err := c.ShouldBindQuery(&req); err != nil {
		// 设置默认值
		req.Page = 1
		req.PageSize = 20
	}

	// 确保默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	return req
}

// PaginatedResponse 分页响应
type PaginatedResponse[T any] struct {
	Items      []T `json:"items"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalPages int `json:"total_pages"`
}

// NewPaginatedResponse 创建分页响应
func NewPaginatedResponse[T any](items []T, total, page, pageSize int) *PaginatedResponse[T] {
	totalPages := (total + pageSize - 1) / pageSize
	return &PaginatedResponse[T]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// BatchRequest 批量操作请求
type BatchRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

// BatchOperationResult 批量操作结果
type BatchOperationResult struct {
	SuccessCount int      `json:"success_count"`
	FailedCount  int      `json:"failed_count"`
	FailedItems  []int    `json:"failed_items,omitempty"`
	Message      string   `json:"message"`
	Errors       []string `json:"errors,omitempty"`
}

// NewBatchSuccessResult 创建批量成功结果
func NewBatchSuccessResult(count int, message string) *BatchOperationResult {
	return &BatchOperationResult{
		SuccessCount: count,
		FailedCount:  0,
		Message:      message,
	}
}

// NewBatchPartialSuccessResult 创建部分成功结果
func NewBatchPartialSuccessResult(successCount int, failedItems []int, message string, errors []string) *BatchOperationResult {
	return &BatchOperationResult{
		SuccessCount: successCount,
		FailedCount:  len(failedItems),
		FailedItems:  failedItems,
		Message:      message,
		Errors:       errors,
	}
}

// LogRequest 记录请求日志
func (bc *BaseController) LogRequest(c *gin.Context, operation string, req interface{}) {
	bc.logger.Infow(
		operation,
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"tenant_id", c.GetInt("tenant_id"),
		"user_id", c.GetInt("user_id"),
		"request_body", bc.sanitizeLogData(req),
	)
}

// sanitizeLogData 清理日志数据中的敏感信息
func (bc *BaseController) sanitizeLogData(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	// 将数据转换为map以进行处理
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "[无法序列化的数据]"
	}

	var dataMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &dataMap); err != nil {
		return "[无法解析的数据]"
	}

	// 敏感字段列表
	sensitiveFields := []string{
		"password", "token", "secret", "key", "auth",
		"authorization", "bearer", "session",
	}

	// 递归清理敏感字段
	bc.sanitizeMap(dataMap, sensitiveFields)

	return dataMap
}

// sanitizeMap 递归清理map中的敏感字段
func (bc *BaseController) sanitizeMap(m map[string]interface{}, sensitiveFields []string) {
	for key, value := range m {
		keyLower := strings.ToLower(key)

		// 检查是否为敏感字段
		isSensitive := false
		for _, field := range sensitiveFields {
			if strings.Contains(keyLower, field) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			m[key] = "***"
		} else if nestedMap, ok := value.(map[string]interface{}); ok {
			bc.sanitizeMap(nestedMap, sensitiveFields)
		} else if slice, ok := value.([]interface{}); ok {
			for _, item := range slice {
				if nestedMap, ok := item.(map[string]interface{}); ok {
					bc.sanitizeMap(nestedMap, sensitiveFields)
				}
			}
		}
	}
}

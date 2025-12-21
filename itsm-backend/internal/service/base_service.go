package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// BaseService 通用服务基类
type BaseService struct {
	logger *zap.SugaredLogger
}

// NewBaseService 创建基础服务
func NewBaseService(logger *zap.SugaredLogger) *BaseService {
	return &BaseService{logger: logger}
}

// Validator 接口定义
type Validator interface {
	Validate() error
}

// ServiceError 服务层错误类型
type ServiceError struct {
	Code    string
	Message string
	Cause   error
}

func (e *ServiceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *ServiceError) Unwrap() error {
	return e.Cause
}

// 预定义错误类型
var (
	ErrNotFound           = &ServiceError{Code: "NOT_FOUND", Message: "资源未找到"}
	ErrAlreadyExists      = &ServiceError{Code: "ALREADY_EXISTS", Message: "资源已存在"}
	ErrInvalidInput       = &ServiceError{Code: "INVALID_INPUT", Message: "输入参数无效"}
	ErrPermissionDenied   = &ServiceError{Code: "PERMISSION_DENIED", Message: "权限不足"}
	ErrOperationFailed    = &ServiceError{Code: "OPERATION_FAILED", Message: "操作失败"}
	ErrResourceLocked     = &ServiceError{Code: "RESOURCE_LOCKED", Message: "资源被锁定"}
	ErrQuotaExceeded      = &ServiceError{Code: "QUOTA_EXCEEDED", Message: "配额超限"}
	ErrDependencyNotFound = &ServiceError{Code: "DEPENDENCY_NOT_FOUND", Message: "依赖资源未找到"}
)

// NewServiceError 创建服务错误
func NewServiceError(code, message string, cause error) *ServiceError {
	return &ServiceError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// IsServiceError 检查是否为服务错误
func IsServiceError(err error) (*ServiceError, bool) {
	if serviceErr, ok := err.(*ServiceError); ok {
		return serviceErr, true
	}
	return nil, false
}

// BaseServiceOperations 通用服务操作接口
type BaseServiceOperations[T any, ID comparable] interface {
	Create(ctx context.Context, req T) (T, error)
	GetByID(ctx context.Context, id ID) (T, error)
	Update(ctx context.Context, id ID, req T) (T, error)
	Delete(ctx context.Context, id ID) error
	List(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]T, int, error)
}

// PaginationRequest 分页请求
type PaginationRequest struct {
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
	Filters  map[string]interface{} `json:"filters"`
	SortBy   string                 `json:"sort_by"`
	SortDesc bool                   `json:"sort_desc"`
}

// PaginationResponse 分页响应
type PaginationResponse[T any] struct {
	Items      []T `json:"items"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalPages int `json:"total_pages"`
}

// NewPaginationResponse 创建分页响应
func NewPaginationResponse[T any](items []T, total, page, pageSize int) *PaginationResponse[T] {
	totalPages := (total + pageSize - 1) / pageSize
	return &PaginationResponse[T]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// ValidateAndLog 验证和记录日志
func (bs *BaseService) ValidateAndLog(ctx context.Context, operation string, validator Validator) error {
	if validator != nil {
		if err := validator.Validate(); err != nil {
			bs.logger.Errorw("验证失败", "operation", operation, "error", err)
			return NewServiceError("VALIDATION_FAILED", "数据验证失败: "+err.Error(), err)
		}
	}

	bs.logger.Infow("开始执行操作", "operation", operation)
	return nil
}

// LogSuccess 记录成功日志
func (bs *BaseService) LogSuccess(ctx context.Context, operation string, details ...interface{}) {
	keysAndValues := make([]interface{}, 0, 4+len(details))
	keysAndValues = append(keysAndValues, "operation", operation)
	keysAndValues = append(keysAndValues, details...)
	keysAndValues = append(keysAndValues, "timestamp", time.Now())
	bs.logger.Infow("操作成功", keysAndValues...)
}

// LogError 记录错误日志
func (bs *BaseService) LogError(ctx context.Context, operation string, err error, details ...interface{}) {
	keysAndValues := make([]interface{}, 0, 6+len(details))
	keysAndValues = append(keysAndValues, "operation", operation, "error", err)
	keysAndValues = append(keysAndValues, details...)
	keysAndValues = append(keysAndValues, "timestamp", time.Now())
	bs.logger.Errorw("操作失败", keysAndValues...)
}

// SanitizeString 清理字符串
func (bs *BaseService) SanitizeString(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// ValidateRequiredFields 验证必填字段
func (bs *BaseService) ValidateRequiredFields(obj map[string]interface{}, requiredFields []string) error {
	missingFields := make([]string, 0)

	for _, field := range requiredFields {
		if value, exists := obj[field]; !exists || value == nil || value == "" {
			missingFields = append(missingFields, field)
		}
	}

	if len(missingFields) > 0 {
		return NewServiceError("MISSING_REQUIRED_FIELDS",
			fmt.Sprintf("缺少必填字段: %s", strings.Join(missingFields, ", ")), nil)
	}

	return nil
}

// ValidateEmail 验证邮箱格式
func (bs *BaseService) ValidateEmail(email string) error {
	email = bs.SanitizeString(email)
	if email == "" {
		return NewServiceError("INVALID_EMAIL", "邮箱不能为空", nil)
	}

	// 简单的邮箱验证
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return NewServiceError("INVALID_EMAIL", "邮箱格式无效", nil)
	}

	return nil
}

// ValidatePhone 验证手机号格式
func (bs *BaseService) ValidatePhone(phone string) error {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return NewServiceError("INVALID_PHONE", "手机号不能为空", nil)
	}

	// 简单的手机号验证（中国大陆）
	if len(phone) != 11 || phone[0] != '1' {
		return NewServiceError("INVALID_PHONE", "手机号格式无效", nil)
	}

	return nil
}

// ValidatePriority 验证优先级
func (bs *BaseService) ValidatePriority(priority string) error {
	priority = strings.ToLower(strings.TrimSpace(priority))

	validPriorities := []string{"", "low", "medium", "high", "urgent", "critical"}
	for _, valid := range validPriorities {
		if priority == valid {
			return nil
		}
	}

	return NewServiceError("INVALID_PRIORITY",
		fmt.Sprintf("无效的优先级: %s，有效值: %s", priority, strings.Join(validPriorities[1:], ", ")), nil)
}

// ValidateStatus 验证状态
func (bs *BaseService) ValidateStatus(status string, validStatuses []string) error {
	status = strings.ToLower(strings.TrimSpace(status))

	if status == "" {
		return nil // 允许空状态（使用默认值）
	}

	for _, valid := range validStatuses {
		if status == valid {
			return nil
		}
	}

	return NewServiceError("INVALID_STATUS",
		fmt.Sprintf("无效的状态: %s，有效值: %s", status, strings.Join(validStatuses, ", ")), nil)
}

// BuildFilterWhere 构建过滤条件
func (bs *BaseService) BuildFilterWhere(filters map[string]interface{}) map[string]interface{} {
	if len(filters) == 0 {
		return nil
	}

	where := make(map[string]interface{})
	for key, value := range filters {
		if value != nil && value != "" {
			where[key] = value
		}
	}

	if len(where) == 0 {
		return nil
	}

	return where
}

// CalculateOffset 计算分页偏移量
func (bs *BaseService) CalculateOffset(page, pageSize int) int {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	return (page - 1) * pageSize
}

// ExecuteWithRetry 带重试的操作执行
func (bs *BaseService) ExecuteWithRetry(ctx context.Context, operation string, maxRetries int, fn func() error) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			bs.logger.Infow("重试操作", "operation", operation, "attempt", i+1)
			// 指数退避
			delay := time.Duration(1<<uint(i-1)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		if err := fn(); err != nil {
			lastErr = err
			bs.logger.Warnw("操作失败，准备重试", "operation", operation, "attempt", i+1, "error", err)

			// 某些错误不适合重试
			if serviceErr, ok := IsServiceError(err); ok {
				if serviceErr.Code == "NOT_FOUND" ||
					serviceErr.Code == "PERMISSION_DENIED" ||
					serviceErr.Code == "VALIDATION_FAILED" {
					break
				}
			}
			continue
		}

		// 成功
		return nil
	}

	bs.logger.Errorw("操作最终失败", "operation", operation, "maxRetries", maxRetries, "error", lastErr)
	return NewServiceError("OPERATION_FAILED",
		fmt.Sprintf("操作失败，已重试%d次", maxRetries), lastErr)
}

// BatchOperation 批量操作接口
type BatchOperation[T, R any] interface {
	Execute(ctx context.Context, items []T) ([]R, error)
}

// ExecuteBatch 执行批量操作
func ExecuteBatch[T, R any](ctx context.Context, operation string, batchSize int, items []T, op BatchOperation[T, R]) ([]R, error) {
	if len(items) == 0 {
		return nil, nil
	}

	var allResults []R

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		results, err := op.Execute(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("批量操作失败 (batch %d-%d): %w", i, end-1, err)
		}

		allResults = append(allResults, results...)
	}

	return allResults, nil
}

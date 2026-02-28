package service

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

// ProcessTriggerServiceInterface 流程触发服务接口
type ProcessTriggerServiceInterface interface {
	// TriggerProcess 触发流程
	TriggerProcess(ctx context.Context, req *dto.ProcessTriggerRequest) (*dto.ProcessTriggerResponse, error)

	// TriggerByBusinessType 根据业务类型自动匹配流程并触发
	TriggerByBusinessType(ctx context.Context, businessType dto.BusinessType, businessID int, variables map[string]interface{}, triggeredBy string, tenantID int) (*dto.ProcessTriggerResponse, error)

	// CancelProcess 取消流程
	CancelProcess(ctx context.Context, processInstanceID int, reason string, tenantID int) error

	// SuspendProcess 暂停流程
	SuspendProcess(ctx context.Context, processInstanceID int, reason string, tenantID int) error

	// ResumeProcess 恢复流程
	ResumeProcess(ctx context.Context, processInstanceID int, tenantID int) error

	// GetProcessStatus 获取流程状态
	GetProcessStatus(ctx context.Context, processInstanceID int, tenantID int) (*dto.ProcessTriggerResponse, error)
}

// ProcessBindingServiceInterface 流程绑定配置服务接口
type ProcessBindingServiceInterface interface {
	// CreateBinding 创建流程绑定
	CreateBinding(ctx context.Context, binding *dto.ProcessBinding) (*dto.ProcessBinding, error)

	// UpdateBinding 更新流程绑定
	UpdateBinding(ctx context.Context, id int, binding *dto.ProcessBinding) (*dto.ProcessBinding, error)

	// DeleteBinding 删除流程绑定
	DeleteBinding(ctx context.Context, id int, tenantID int) error

	// GetBinding 获取流程绑定
	GetBinding(ctx context.Context, id int, tenantID int) (*dto.ProcessBinding, error)

	// QueryBindings 查询流程绑定列表
	QueryBindings(ctx context.Context, req *dto.ProcessBindingQueryRequest) ([]*dto.ProcessBinding, error)

	// FindBestBinding 根据业务类型查找最佳流程绑定
	FindBestBinding(ctx context.Context, businessType dto.BusinessType, subType string, tenantID int) (*dto.ProcessBinding, error)

	// BatchCreateBindings 批量创建流程绑定
	BatchCreateBindings(ctx context.Context, req *dto.BatchProcessBindingRequest) error
}

// ServiceTaskHandlerInterface 服务任务处理器接口
type ServiceTaskHandlerInterface interface {
	// GetTaskType 返回处理器支持的任务类型
	GetTaskType() string

	// Execute 执行服务任务
	Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error)

	// Validate 验证参数
	Validate(ctx context.Context, config map[string]interface{}) error

	// GetHandlerID 返回处理器标识
	GetHandlerID() string
}

// ProcessCallbackServiceInterface 流程回调服务接口
type ProcessCallbackServiceInterface interface {
	// HandleCallback 处理流程回调
	HandleCallback(ctx context.Context, req *dto.CallbackRequest) error

	// RegisterHandler 注册服务任务处理器
	RegisterHandler(handler ServiceTaskHandlerInterface)

	// UnregisterHandler 注销处理器
	UnregisterHandler(handlerID string)
}

package bpmn

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"

	"go.uber.org/zap"
)

// GenericServiceTaskHandler 通用服务任务处理器
type GenericServiceTaskHandler struct {
	HandlerBase
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewGenericServiceTaskHandler 创建通用处理器
func NewGenericServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *GenericServiceTaskHandler {
	return &GenericServiceTaskHandler{
		client: client,
		logger: logger,
	}
}

// GetTaskType 返回任务类型
func (h *GenericServiceTaskHandler) GetTaskType() string {
	return "generic_task"
}

// GetHandlerID 返回处理器标识
func (h *GenericServiceTaskHandler) GetHandlerID() string {
	return "generic_service_handler"
}

// Execute 执行通用服务任务
func (h *GenericServiceTaskHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	// 通用处理器可以根据配置执行各种操作
	operation, _ := variables["operation"].(string)

	result := &dto.ServiceTaskResult{
		Success:    true,
		Message:    fmt.Sprintf("通用任务 %s 执行完成", operation),
		OutputVars: make(map[string]interface{}),
	}

	// 将输入变量透传到输出
	for k, v := range variables {
		result.OutputVars[k] = v
	}

	return result, nil
}

// Validate 验证配置
func (h *GenericServiceTaskHandler) Validate(ctx context.Context, config map[string]interface{}) error {
	return nil
}

// 确保 GenericServiceTaskHandler 实现了 ServiceTaskHandlerInterface
var _ ServiceTaskHandlerInterface = (*GenericServiceTaskHandler)(nil)

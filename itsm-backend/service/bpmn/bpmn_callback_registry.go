package bpmn

import (
	"context"
	"fmt"
	"sync"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/processtask"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// CallbackRegistry 流程回调注册中心
// 负责管理所有服务任务处理器的注册和处理流程回调
type CallbackRegistry struct {
	client     *ent.Client
	logger     *zap.SugaredLogger
	handlers   map[string]ServiceTaskHandlerInterface
	handlersMu sync.RWMutex
}

// NewCallbackRegistry 创建新的回调注册中心
func NewCallbackRegistry(client *ent.Client, logger *zap.SugaredLogger) *CallbackRegistry {
	registry := &CallbackRegistry{
		client:   client,
		logger:   logger,
		handlers: make(map[string]ServiceTaskHandlerInterface),
	}

	// 注册默认处理器
	registry.registerDefaultHandlers()

	return registry
}

// RegisterHandler 注册服务任务处理器
func (r *CallbackRegistry) RegisterHandler(handler ServiceTaskHandlerInterface) {
	r.handlersMu.Lock()
	defer r.handlersMu.Unlock()
	r.handlers[handler.GetHandlerID()] = handler
}

// UnregisterHandler 注销处理器
func (r *CallbackRegistry) UnregisterHandler(handlerID string) {
	r.handlersMu.Lock()
	defer r.handlersMu.Unlock()
	delete(r.handlers, handlerID)
}

// GetHandler 获取处理器
func (r *CallbackRegistry) GetHandler(handlerID string) ServiceTaskHandlerInterface {
	r.handlersMu.RLock()
	defer r.handlersMu.RUnlock()
	return r.handlers[handlerID]
}

// HandleCallback 处理流程回调
func (r *CallbackRegistry) HandleCallback(ctx context.Context, req *dto.CallbackRequest) error {
	// 1. 获取任务
	task, err := r.client.ProcessTask.Query().
		Where(processtask.ID(req.ProcessInstanceID)). // 实际应该是 task_id
		Only(ctx)
	if err != nil {
		return errors.Wrap(err, "查询任务失败")
	}

	// 2. 获取处理器
	handler := r.getHandler(req.ActivityType)
	if handler == nil {
		return fmt.Errorf("未找到任务类型 %s 的处理器", req.ActivityType)
	}

	// 3. 执行处理器
	_, err = handler.Execute(ctx, nil, req.Result.OutputVars)
	if err != nil {
		return errors.Wrap(err, "执行服务任务失败")
	}

	// 4. 更新任务状态
	_, err = r.client.ProcessTask.Update().
		Where(processtask.ID(task.ID)).
		SetStatus("completed").
		Save(ctx)
	if err != nil {
		return errors.Wrap(err, "更新任务状态失败")
	}

	// 5. 如果流程引擎需要，推进流程
	// （由流程引擎自己处理）

	return nil
}

// getHandler 获取处理器
func (r *CallbackRegistry) getHandler(taskType string) ServiceTaskHandlerInterface {
	r.handlersMu.RLock()
	defer r.handlersMu.RUnlock()

	// 精确匹配
	if handler, ok := r.handlers[taskType]; ok {
		return handler
	}

	// 通配匹配
	for _, handler := range r.handlers {
		if handler.GetTaskType() == taskType {
			return handler
		}
	}

	return nil
}

// registerDefaultHandlers 注册默认处理器
// 注意：通知服务的设置需要在外部通过 SetNotificationService 完成
func (r *CallbackRegistry) registerDefaultHandlers() {
	// 注册 Ticket 服务任务处理器
	r.RegisterHandler(NewTicketServiceTaskHandler(r.client, r.logger))

	// 注册 Change 服务任务处理器
	r.RegisterHandler(NewChangeServiceTaskHandler(r.client, r.logger))
	// 注册 Incident 服务任务处理器
	r.RegisterHandler(NewIncidentServiceTaskHandler(r.client, r.logger))
	// 注册通用服务任务处理器
	r.RegisterHandler(NewGenericServiceTaskHandler(r.client, r.logger))
	// 注册服务请求处理器
	r.RegisterHandler(NewServiceRequestServiceTaskHandler(r.client, r.logger))
	// 注册通知处理器
	r.RegisterHandler(NewNotificationHandler(r.client, r.logger))
	// 注册审批处理器
	r.RegisterHandler(NewApprovalHandler(r.client, r.logger))
	// 注册Webhook处理器
	r.RegisterHandler(NewWebhookHandler(r.client, r.logger))
}

// RegisterTicketHandlerWithNotification 注册带通知服务的工单处理器
func (r *CallbackRegistry) RegisterTicketHandlerWithNotification(handler *TicketServiceTaskHandler) {
	r.RegisterHandler(handler)
}

// ListHandlers 列出所有已注册的处理器
func (r *CallbackRegistry) ListHandlers() []ServiceTaskHandlerInterface {
	r.handlersMu.RLock()
	defer r.handlersMu.RUnlock()

	handlers := make([]ServiceTaskHandlerInterface, 0, len(r.handlers))
	for _, h := range r.handlers {
		handlers = append(handlers, h)
	}
	return handlers
}

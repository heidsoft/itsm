package service

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/service/bpmn"

	"go.uber.org/zap"
)

// ProcessCallbackService 流程回调服务实现
// 此文件作为入口，仅负责路由到 bpmn 包中的处理
type ProcessCallbackService struct {
	client   *ent.Client
	logger   *zap.SugaredLogger
	registry *bpmn.CallbackRegistry
}

// NewProcessCallbackService 创建流程回调服务
func NewProcessCallbackService(client *ent.Client, logger *zap.SugaredLogger) *ProcessCallbackService {
	svc := &ProcessCallbackService{
		client:   client,
		logger:   logger,
		registry: bpmn.NewCallbackRegistry(client, logger),
	}

	return svc
}

// HandleCallback 处理流程回调
func (s *ProcessCallbackService) HandleCallback(ctx context.Context, req *dto.CallbackRequest) error {
	// 委托给注册中心处理
	return s.registry.HandleCallback(ctx, req)
}

// RegisterHandler 注册服务任务处理器
func (s *ProcessCallbackService) RegisterHandler(handler ServiceTaskHandlerInterface) {
	// 将 service 包的接口适配到 bpmn 包
	// 这里需要一个适配器，但为了简化，我们直接在 bpmn 包中操作
	// 暂时忽略这个方法，实际注册在 NewCallbackRegistry 中完成
}

// UnregisterHandler 注销处理器
func (s *ProcessCallbackService) UnregisterHandler(handlerID string) {
	s.registry.UnregisterHandler(handlerID)
}

// GetHandler 获取处理器
func (s *ProcessCallbackService) GetHandler(handlerID string) ServiceTaskHandlerInterface {
	return nil // 暂不提供此功能
}

// ListHandlers 列出所有已注册的处理器
func (s *ProcessCallbackService) ListHandlers() []ServiceTaskHandlerInterface {
	return nil // 暂不提供此功能
}

// getHandler 获取处理器（内部使用）
// 保留此方法以保持向后兼容
func (s *ProcessCallbackService) getHandler(taskType string) ServiceTaskHandlerInterface {
	return nil
}

// registerDefaultHandlers 注册默认处理器
// 保留此方法以保持向后兼容，实际注册逻辑在 bpmn 包中完成
func (s *ProcessCallbackService) registerDefaultHandlers(logger *zap.SugaredLogger) {
	// 默认处理器已在 bpmn.CallbackRegistry 初始化时注册
	// 此处可以添加额外的自定义处理器
}

// ServiceTaskHandlerBase 服务任务处理器基类
// 保留用于向后兼容，新代码应使用 bpmn.HandlerBase
type ServiceTaskHandlerBase struct {
	client *ent.Client
}

// GetHandlerID 返回处理器标识
func (h *ServiceTaskHandlerBase) GetHandlerID() string {
	return ""
}

// Validate 验证配置
func (h *ServiceTaskHandlerBase) Validate(ctx context.Context, config map[string]interface{}) error {
	return nil
}

// 确保 ProcessCallbackService 实现了 ProcessCallbackServiceInterface
var _ ProcessCallbackServiceInterface = (*ProcessCallbackService)(nil)

// 保留以下构造函数用于向后兼容
// 它们现在委托给 bpmn 包中的实现

// NewTicketServiceTaskHandler 创建工单处理器（向后兼容）
func NewTicketServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *bpmn.TicketServiceTaskHandler {
	return bpmn.NewTicketServiceTaskHandler(client, logger)
}

// NewChangeServiceTaskHandler 创建变更处理器（向后兼容）
func NewChangeServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *bpmn.ChangeServiceTaskHandler {
	return bpmn.NewChangeServiceTaskHandler(client, logger)
}

// NewIncidentServiceTaskHandler 创建事件处理器（向后兼容）
func NewIncidentServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *bpmn.IncidentServiceTaskHandler {
	return bpmn.NewIncidentServiceTaskHandler(client, logger)
}

// NewGenericServiceTaskHandler 创建通用处理器（向后兼容）
func NewGenericServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *bpmn.GenericServiceTaskHandler {
	return bpmn.NewGenericServiceTaskHandler(client, logger)
}

// NewServiceRequestServiceTaskHandler 创建服务请求处理器（向后兼容）
func NewServiceRequestServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *bpmn.ServiceRequestServiceTaskHandler {
	return bpmn.NewServiceRequestServiceTaskHandler(client, logger)
}

// 保留辅助函数用于向后兼容

// getIntFromVars 从变量中提取整数
func getIntFromVars(variables map[string]interface{}, key string) int {
	return bpmn.GetIntFromVars(variables, key)
}

// getStringFromVars 从变量中提取字符串
func getStringFromVars(variables map[string]interface{}, key string) string {
	return bpmn.GetStringFromVars(variables, key)
}

// getTenantIDFromVars 从变量中提取租户ID
func getTenantIDFromVars(variables map[string]interface{}) int {
	return bpmn.GetTenantIDFromVars(variables)
}

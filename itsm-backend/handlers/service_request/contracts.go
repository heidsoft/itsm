package service_request

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/service/sla"
)

// ApprovalChainResolver 是服务请求域消费审批链求值引擎的窄契约。
// 方法集 = service_request 域全部生产文件的调用并集。
type ApprovalChainResolver interface {
	ResolveApprovalPlan(ctx context.Context, tenantID int, entityType string, evalCtx dto.ApprovalEvalContext, approvals map[int][]int) (*dto.ApprovalChainEvaluation, error)
	ShadowCompare(ctx context.Context, tenantID int, entityType string, evalCtx dto.ApprovalEvalContext, approvals map[int][]int, directPlan *dto.ApprovalChainEvaluation, directErr error) *dto.ShadowResult
}

// SRApprovalBridge 把服务请求生命周期动作推进到 BPMN 流程待办。
// 实现由组合根注入，本域不构造共享层对象。
type SRApprovalBridge interface {
	CompleteBusinessWorkTasks(ctx context.Context, tenantID, actorUserID int, businessType string, businessID int, variables map[string]interface{}, maxSteps int) (int, error)
	CompleteBusinessApprovalTaskWithClient(ctx context.Context, txc *ent.Client, tenantID, actorUserID int, businessType string, businessID int, action, comment string) (bool, error)
	CheckActionableApprovalTask(ctx context.Context, tenantID int, businessType string, businessID int) (bool, error)
	TerminateBusinessWorkflow(ctx context.Context, tenantID, actorUserID int, businessType string, businessID int, reason string) error
}

// SRSLAService 是服务请求域消费的 SLA 窄契约：只暴露「按请求计算截止时间」
// 与「拿到 sla_states 存储」两个用例（与 problem 域 SLAService 同一口径）。
type SRSLAService interface {
	CalculateSLADeadlineFromRequest(ctx context.Context, tenantID int, ticketType, priority string) (*dto.SLADeadlineResult, error)
	SLAStore() *sla.Store
}

// ProvisioningDeliveryListener 是交付事件监听契约（本地声明，不引用共享层接口）。
// 共享层共享层的交付编排器以此接口回调服务请求域：交付开始记 SLA 首响，
// 交付成功记解决时间并推进流程作业节点。
type ProvisioningDeliveryListener interface {
	OnProvisioningStarted(ctx context.Context, tenantID, serviceRequestID, actorUserID int)
	OnProvisioningFinished(ctx context.Context, tenantID, serviceRequestID, actorUserID int, delivered bool, detail string)
}

// ServiceOption 是服务请求域构造期依赖注入选项。
//
// 服务请求域的外部依赖（BPMN 流程桥接）由组合根显式注入，本域不构造
// 共享层对象；选项函数让组合根能在一行内传入全部依赖，app.go 净增 0 行。
type ServiceOption func(*Service)

// WithApprovalBridge 注入 BPMN 流程桥接（组合根传共享层具体指针，隐式满足窄契约）。
// 未注入时服务请求域照常流转，流程实例不随动——未接线不是阻断项。
func WithApprovalBridge(bridge SRApprovalBridge) ServiceOption {
	return func(s *Service) { s.approvalBridge = bridge }
}

package change

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

// ApprovalChainResolver 是变更域消费审批链求值引擎的窄契约。
//
// 方法集 = change 域全部生产文件的调用并集：resolveChangeChainPlan 用
// ResolveApprovalPlan 求值、用 ShadowCompare 做影子双轨对比。共享层具体
// 引擎隐式满足本接口（求值上下文与结果在两侧是同一 dto 类型，service 侧为别名）。
type ApprovalChainResolver interface {
	ResolveApprovalPlan(ctx context.Context, tenantID int, entityType string, evalCtx dto.ApprovalEvalContext, approvals map[int][]int) (*dto.ApprovalChainEvaluation, error)
	ShadowCompare(ctx context.Context, tenantID int, entityType string, evalCtx dto.ApprovalEvalContext, approvals map[int][]int, directPlan *dto.ApprovalChainEvaluation, directErr error) *dto.ShadowResult
}

// ChangeApprovalBridge 把变更生命周期动作推进到 BPMN 流程待办，必须在业务事务内
// 调用。实现由组合根注入，本域不构造共享层对象（与 problem 域 WorkflowBridge 同一口径）。
type ChangeApprovalBridge interface {
	AdvanceBusinessWorkTaskWithClient(ctx context.Context, txc *ent.Client, tenantID, actorUserID int, businessType string, businessID int, variables map[string]interface{}) (int, error)
	AdvanceBusinessWorkflowWithClient(ctx context.Context, txc *ent.Client, tenantID, actorUserID int, businessType string, businessID int, variables map[string]interface{}, maxSteps int) (bool, error)
	CompleteBusinessApprovalTaskWithClient(ctx context.Context, txc *ent.Client, tenantID, actorUserID int, businessType string, businessID int, action, comment string) (bool, error)
}

// ApprovalChainCloser 收口变更终态时残留的 pending 审批链节点。
//
// 共享层对应实现是包级函数（走全局 DB handle），无法下沉为值类型或窄接口，
// 故以函数类型契约由组合根注入。域内不感知其实现，也不复制第二份 SQL。
type ApprovalChainCloser func(ctx context.Context, changeID, tenantID int) error

// ServiceOption 是变更域构造期依赖注入选项。
//
// 变更域的外部依赖有两个（BPMN 流程桥接、审批链收口函数），都用变参注入会
// 与 Go「只能有一个变参」冲突，故统一收敛为选项函数：组合根在一行内传入
// 全部依赖，app.go 保持净增 0 行。
type ServiceOption func(*Service)

// WithApprovalBridge 注入 BPMN 流程桥接（组合根传共享层具体指针，隐式满足窄契约）。
// 未注入时变更域照常流转，流程实例不随动——未接线不是阻断项。
func WithApprovalBridge(bridge ChangeApprovalBridge) ServiceOption {
	return func(s *Service) { s.approvalBridge = bridge }
}

// WithApprovalChainCloser 注入审批链收口函数（共享层包级函数走全局 DB handle，
// 无法下沉为窄接口）。未注入时终态收口退化为「记日志待人工收口」。
func WithApprovalChainCloser(closer ApprovalChainCloser) ServiceOption {
	return func(s *Service) { s.closeApprovalChains = closer }
}

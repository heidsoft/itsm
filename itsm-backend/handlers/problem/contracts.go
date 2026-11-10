package problem

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/service/sla"
)

// SLAService 是 problem 域消费的 SLA 窄契约：只暴露「按请求计算截止时间」
// 与「拿到 sla_states 存储」两个用例，不感知共享层具体服务类型。
type SLAService interface {
	CalculateSLADeadlineFromRequest(ctx context.Context, tenantID int, ticketType, priority string) (*dto.SLADeadlineResult, error)
	SLAStore() *sla.Store
}

// WorkflowBridge 把问题生命周期动作推进到 BPMN 流程待办，必须在业务事务内
// 调用。实现由组合根注入，本域不构造共享层对象。
type WorkflowBridge interface {
	AdvanceBusinessWorkflowWithClient(ctx context.Context, txc *ent.Client, tenantID, actorUserID int, businessType string, businessID int, variables map[string]interface{}, maxSteps int) (bool, error)
}

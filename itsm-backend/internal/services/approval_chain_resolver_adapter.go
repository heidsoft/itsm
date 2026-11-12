package service

import (
	"context"
	"encoding/json"

	"itsm-backend/ent"

	"go.uber.org/zap"
)

// ApprovalChainResolverAdapter 实现 bpmn.ApprovalChainResolver 接口。
// 将 ApprovalChainService 适配为 BPMN Service Task 可调用的求值器。
type ApprovalChainResolverAdapter struct {
	client *ent.Client
	logger *zap.SugaredLogger
	svc    *ApprovalChainService
}

// NewApprovalChainResolverAdapter 创建适配器。
func NewApprovalChainResolverAdapter(client *ent.Client, logger *zap.SugaredLogger, svc *ApprovalChainService) *ApprovalChainResolverAdapter {
	return &ApprovalChainResolverAdapter{client: client, logger: logger, svc: svc}
}

// ResolveApprovalPlanRaw 查找匹配审批链并求值，返回原始 JSON 和聚合状态。
func (a *ApprovalChainResolverAdapter) ResolveApprovalPlanRaw(
	ctx context.Context,
	tenantID int,
	entityType string,
	requesterID int,
	priority string,
	amount float64,
	approvals map[int][]int,
) (chainID int, levelsJSON []byte, passed bool, pendingLevel int, blocked bool, err error) {
	evalCtx := ApprovalEvalContext{
		RequesterID: requesterID,
		Priority:    priority,
		Amount:      amount,
	}

	plan, planErr := a.svc.ResolveApprovalPlan(ctx, tenantID, entityType, evalCtx, approvals)
	if planErr != nil {
		return 0, nil, false, 0, false, planErr
	}

	raw, marshalErr := json.Marshal(plan.Levels)
	if marshalErr != nil {
		return 0, nil, false, 0, false, marshalErr
	}

	return plan.ChainID, raw, plan.Passed, plan.PendingLevel, plan.Blocked, nil
}

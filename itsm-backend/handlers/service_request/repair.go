package service_request

import (
	"context"

	domainRole "itsm-backend/domain/role"

	"go.uber.org/zap"
)

// PendingApprovalRepairer 对存量 pending 服务请求做审批链自愈。
//
// 背景（2026-09-07 场景深测 P1-A）：审批人解析修复前创建的请求，其
// service_request_approvals.node.approver_ids 可能退化为 super_admin
// 独审（同部门审批人缺失时旧逻辑直接兜底 admin）。修复只影响新请求，
// 存量请求需要按新解析逻辑重算。此任务在应用启动时执行一次，幂等：
//   - 仅处理未终态（pending 审批中）的请求；
//   - 仅当重算结果与现有 approver_ids 不同才写回；
//   - 已有明确人工决策（已 approved/rejected 的记录）不动。
type PendingApprovalRepairer struct {
	repo   Repository
	logger *zap.SugaredLogger
}

// NewPendingApprovalRepairer 创建审批链自愈器。
func NewPendingApprovalRepairer(repo Repository, logger *zap.SugaredLogger) *PendingApprovalRepairer {
	return &PendingApprovalRepairer{repo: repo, logger: logger}
}

// RunOnce 执行一轮自愈，返回修复的请求数。任何单条失败只记日志不中断。
func (r *PendingApprovalRepairer) RunOnce(ctx context.Context, tenantID int) (int, error) {
	// 复用 ListPendingApprovals 的过滤语义：只看待审批中的请求。
	// requiredStatus 为空表示不过滤审批状态；requesterDept 空表示不限部门。
	requests, _, err := r.repo.ListPendingApprovals(ctx, tenantID, 0, "", "", 1, 500)
	if err != nil {
		return 0, err
	}

	repaired := 0
	for _, req := range requests {
		approvals, err := r.loadApprovals(ctx, req.ID, tenantID)
		if err != nil {
			r.logger.Warnw("repair: 加载审批链失败", "request_id", req.ID, "error", err)
			continue
		}
		changed := false
		for _, approval := range approvals {
			if approval.Status != "pending" {
				continue // 人工已决策的不动
			}
			role := roleByStep(approval.Step)
			if role == "" {
				continue
			}
			requesterDept, _, err := r.repo.GetUserContext(ctx, req.RequesterID, tenantID)
			if err != nil {
				r.logger.Warnw("repair: 取申请人部门失败", "request_id", req.ID, "error", err)
				continue
			}
			newApprovers, err := r.repo.FindActiveUsersByRole(ctx, tenantID, role, requesterDept)
			if err != nil || len(newApprovers) == 0 {
				// 与 resolveApproversForStep 保持一致的最终兜底：super_admin
				if fallback, ferr := r.repo.FindActiveUsersByRole(ctx, tenantID, domainRole.SuperAdmin, ""); ferr == nil && len(fallback) > 0 {
					newApprovers = fallback
				}
			}
			if sameIntSlice(approval.Node["approver_ids"], newApprovers) {
				continue
			}
			if approval.Node == nil {
				approval.Node = map[string]interface{}{}
			}
			approval.Node["approver_ids"] = newApprovers
			if err := r.repo.UpdateApproval(ctx, approval); err != nil {
				r.logger.Warnw("repair: 写回审批人失败", "request_id", req.ID, "level", approval.Level, "error", err)
				continue
			}
			r.logger.Infow("repair: 存量审批人已重算",
				"request_id", req.ID, "level", approval.Level, "step", approval.Step, "approvers", newApprovers)
			changed = true
		}
		if changed {
			repaired++
		}
	}
	return repaired, nil
}

// loadApprovals 加载单条请求的审批链。
func (r *PendingApprovalRepairer) loadApprovals(ctx context.Context, requestID, tenantID int) ([]*ServiceRequestApproval, error) {
	_, approvals, err := r.repo.GetWithApprovals(ctx, requestID, tenantID)
	return approvals, err
}

// roleByStep 是 resolveApproversForStep 的 step→role 映射的复刻。
// 单一源对齐：与 service.go roleByStep 保持同步（两处均在 handlers/service_request 包内）。
func roleByStep(step string) string {
	switch step {
	case ApprovalStepManager:
		return domainRole.Manager
	case ApprovalStepIT:
		return domainRole.ITAdmin
	case ApprovalStepSecurity:
		return domainRole.SecurityAdmin
	default:
		return ""
	}
}

// sameIntSlice 比较 node.approver_ids（JSON 反序列化后为 []interface{}）与新列表。
func sameIntSlice(nodeVal interface{}, ids []int) bool {
	list, ok := nodeVal.([]interface{})
	if !ok {
		return false
	}
	if len(list) != len(ids) {
		return false
	}
	for i, v := range list {
		f, ok := v.(float64) // encoding/json 数字默认反序列化为 float64
		if !ok {
			// 兼容 int64/int 直接塞入的情形
			switch n := v.(type) {
			case int:
				if n != ids[i] {
					return false
				}
				continue
			case int64:
				if int(n) != ids[i] {
					return false
				}
				continue
			default:
				return false
			}
		}
		if int(f) != ids[i] {
			return false
		}
	}
	return true
}

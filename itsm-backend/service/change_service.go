package service

import (
	"context"
	"fmt"

	"itsm-backend/common"
	"itsm-backend/database"
	"itsm-backend/dto"
)

// CloseChangeApprovalChains 在变更进入终态时收口残留 pending 审批链节点（对外导出，handlers 包直接复用）。
// 注意：调用方需自行保证已通过状态机校验且已将变更写入终态。
func CloseChangeApprovalChains(ctx context.Context, changeID, tenantID int) error {
	rawDB := database.GetRawDB()
	if rawDB == nil {
		return fmt.Errorf("raw database handle unavailable, skip closing change_approval_chains")
	}
	_, err := rawDB.ExecContext(ctx, `
		UPDATE change_approval_chains
		SET status = 'obsolete'
		WHERE change_id = $1 AND tenant_id = $2 AND status = 'pending'
	`, changeID, tenantID)
	return err
}

// normalizeChangeTransitionStatus 把待审批的两个同义词归一到迁移表词表（submitted）。
func normalizeChangeTransitionStatus(status string) string {
	if status == string(dto.ChangeStatusPending) {
		return common.ChangeStatusSubmitted
	}
	return status
}

// IsValidChangeStatusTransition 检查变更状态转换是否合法
// Change状态转换规则:
// draft -> submitted, cancelled
// submitted -> approved, rejected, cancelled
// approved -> scheduled, cancelled
// rejected -> (不允许转换到其他状态)
// scheduled -> in_progress, cancelled
// in_progress -> completed, failed, cancelled
// completed -> (不允许转换到其他状态)
// failed -> scheduled, cancelled
// cancelled -> (不允许转换到其他状态)
//
// 根据ITIL标准，不同类型的变更有不同的状态转换规则
func IsValidChangeStatusTransition(currentStatus, newStatus, changeType string) bool {
	// 词表归一：待审批在 API/枚举/存量库里分别是 pending 与 submitted，两者等价。
	// 迁移表以 submitted 为词表，因此源和目标都要归一 —— 只归一源会让
	// draft -> pending（提交审批）被判成非法迁移。
	currentStatus = normalizeChangeTransitionStatus(currentStatus)
	newStatus = normalizeChangeTransitionStatus(newStatus)

	// 基础转换规则（适用于所有变更类型）
	baseTransitions := map[string][]string{
		common.ChangeStatusRejected:        {},                          // 被拒绝后不允许转换
		common.ChangeStatusCompleted:       {common.ChangeStatusClosed}, // 已完成 → 可显式关闭以归档
		common.ChangeStatusCancelled:       {},                          // 已取消不允许转换
		string(dto.ChangeStatusRolledBack): {common.ChangeStatusClosed}, // 已回滚 → 也可归档
		common.ChangeStatusClosed:          {},                          // 已关闭为终态
	}

	// 不同变更类型的特殊转换规则
	var typeSpecificTransitions map[string][]string
	switch changeType {
	case string(dto.ChangeTypeStandard):
		// 标准变更：预授权，可以跳过审批步骤
		typeSpecificTransitions = map[string][]string{
			common.ChangeStatusDraft:      {common.ChangeStatusSubmitted, common.ChangeStatusApproved, common.ChangeStatusScheduled, common.ChangeStatusInProgress, common.ChangeStatusCancelled},
			common.ChangeStatusSubmitted:  {common.ChangeStatusApproved, common.ChangeStatusRejected, common.ChangeStatusCancelled},
			common.ChangeStatusApproved:   {common.ChangeStatusScheduled, common.ChangeStatusInProgress, common.ChangeStatusCancelled},
			common.ChangeStatusScheduled:  {common.ChangeStatusInProgress, common.ChangeStatusCancelled},
			common.ChangeStatusInProgress: {common.ChangeStatusCompleted, common.ChangeStatusFailed, string(dto.ChangeStatusRolledBack), common.ChangeStatusCancelled},
			common.ChangeStatusFailed:     {common.ChangeStatusScheduled, string(dto.ChangeStatusRolledBack), common.ChangeStatusCancelled},
		}
	case string(dto.ChangeTypeEmergency):
		// 紧急变更：可以跳过多个步骤，快速实施
		typeSpecificTransitions = map[string][]string{
			common.ChangeStatusDraft:      {common.ChangeStatusSubmitted, common.ChangeStatusApproved, common.ChangeStatusInProgress, common.ChangeStatusCancelled},
			common.ChangeStatusSubmitted:  {common.ChangeStatusApproved, common.ChangeStatusRejected, common.ChangeStatusCancelled},
			common.ChangeStatusApproved:   {common.ChangeStatusInProgress, common.ChangeStatusCancelled},
			common.ChangeStatusInProgress: {common.ChangeStatusCompleted, common.ChangeStatusFailed, string(dto.ChangeStatusRolledBack), common.ChangeStatusCancelled},
			common.ChangeStatusFailed:     {common.ChangeStatusScheduled, string(dto.ChangeStatusRolledBack), common.ChangeStatusCancelled},
		}
	default: // 普通变更：严格的ITIL流程
		typeSpecificTransitions = map[string][]string{
			common.ChangeStatusDraft:      {common.ChangeStatusSubmitted, common.ChangeStatusCancelled},
			common.ChangeStatusSubmitted:  {common.ChangeStatusApproved, common.ChangeStatusRejected, common.ChangeStatusCancelled},
			common.ChangeStatusApproved:   {common.ChangeStatusScheduled, common.ChangeStatusCancelled},
			common.ChangeStatusScheduled:  {common.ChangeStatusInProgress, common.ChangeStatusCancelled},
			common.ChangeStatusInProgress: {common.ChangeStatusCompleted, common.ChangeStatusFailed, string(dto.ChangeStatusRolledBack), common.ChangeStatusCancelled},
			common.ChangeStatusFailed:     {common.ChangeStatusScheduled, string(dto.ChangeStatusRolledBack), common.ChangeStatusCancelled},
		}
	}

	// 合并基础规则和类型特定规则
	validTransitions := make(map[string][]string)
	for k, v := range baseTransitions {
		validTransitions[k] = v
	}
	for k, v := range typeSpecificTransitions {
		validTransitions[k] = v
	}

	allowed, ok := validTransitions[currentStatus]
	if !ok {
		// 未知状态必须失败关闭，避免绕过变更生命周期约束。
		return false
	}

	for _, status := range allowed {
		if status == newStatus {
			return true
		}
	}
	return false
}

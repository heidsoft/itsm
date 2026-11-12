package service

import (
	"context"
	"fmt"

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
// 唯一实现在 dto 包；此处保留薄委托以保持 service 包既有调用点不变。
func normalizeChangeTransitionStatus(status string) string {
	return dto.NormalizeChangeTransitionStatus(status)
}

// IsValidChangeStatusTransition 检查变更状态转换是否合法。
// 唯一实现在 dto 包；此处保留薄委托以保持 service 包既有调用点不变。
func IsValidChangeStatusTransition(currentStatus, newStatus, changeType string) bool {
	return dto.IsValidChangeStatusTransition(currentStatus, newStatus, changeType)
}

// ---------------------------------------------------------------------------
// 变更状态机迁移表（包级事实源）
//
// 提为包级变量而非函数内字面量，是为了让守卫测试能枚举「状态机实际承认的
// 值集合」，与 common.StatusVocabulary 的数据库 CHECK 词表做包含关系断言。
// 函数内字面量无法被外部观察，词表漂移就只能靠人工 review 发现——而
// 状态机新增一个值、数据库约束没跟上时，mock 单测全绿、生产写入撞 23514。
// ---------------------------------------------------------------------------

// changeBaseTransitions 的唯一实现在 dto 包；此处保留同名别名，守卫测试仍可枚举状态机承认的值集合。
var changeBaseTransitions = dto.ChangeBaseTransitions

// changeStandardTransitions 的唯一实现在 dto 包；此处保留同名别名，守卫测试仍可枚举状态机承认的值集合。
var changeStandardTransitions = dto.ChangeStandardTransitions

// changeEmergencyTransitions 的唯一实现在 dto 包；此处保留同名别名，守卫测试仍可枚举状态机承认的值集合。
var changeEmergencyTransitions = dto.ChangeEmergencyTransitions

// changeNormalTransitions 的唯一实现在 dto 包；此处保留同名别名，守卫测试仍可枚举状态机承认的值集合。
var changeNormalTransitions = dto.ChangeNormalTransitions

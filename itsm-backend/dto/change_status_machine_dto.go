package dto

import "itsm-backend/common"

// 变更状态机（纯函数组）：唯一实现在本包，service 侧保留薄委托与迁移表变量别名。
//
// 这组函数的闭包 = 词表归一 + 四张迁移表 + 判定函数，全部无状态、可枚举。
// 迁表提为包级变量而非函数内字面量，是为了让守卫测试能枚举「状态机实际
// 承认的值集合」，与 common.StatusVocabulary 的数据库 CHECK 词表做包含关系
// 断言——函数内字面量无法被外部观察，词表漂移只能靠人工 review 发现，而
// 状态机新增一个值、数据库约束没跟上时 mock 单测全绿、生产写入撞 23514。
//
// 状态词表单一源在 common 包（dto 引用之，不另起一份），避免双源漂移。

// ChangeBaseTransitions 适用于所有变更类型的终态/归档规则。
var ChangeBaseTransitions = map[string][]string{
	common.ChangeStatusRejected:    {},                          // 被拒绝后不允许转换
	common.ChangeStatusCompleted:   {common.ChangeStatusClosed}, // 已完成 → 可显式关闭以归档
	common.ChangeStatusCancelled:   {},                          // 已取消不允许转换
	string(ChangeStatusRolledBack): {common.ChangeStatusClosed}, // 已回滚 → 也可归档
	common.ChangeStatusClosed:      {},                          // 已关闭为终态
}

// ChangeStandardTransitions 标准变更：预授权，可以跳过审批步骤。
var ChangeStandardTransitions = map[string][]string{
	common.ChangeStatusDraft:      {common.ChangeStatusSubmitted, common.ChangeStatusApproved, common.ChangeStatusScheduled, common.ChangeStatusInProgress, common.ChangeStatusCancelled},
	common.ChangeStatusSubmitted:  {common.ChangeStatusApproved, common.ChangeStatusRejected, common.ChangeStatusCancelled},
	common.ChangeStatusApproved:   {common.ChangeStatusScheduled, common.ChangeStatusInProgress, common.ChangeStatusCancelled},
	common.ChangeStatusScheduled:  {common.ChangeStatusInProgress, common.ChangeStatusCancelled},
	common.ChangeStatusInProgress: {common.ChangeStatusCompleted, common.ChangeStatusFailed, string(ChangeStatusRolledBack), common.ChangeStatusCancelled},
	common.ChangeStatusFailed:     {common.ChangeStatusScheduled, string(ChangeStatusRolledBack), common.ChangeStatusCancelled},
}

// ChangeEmergencyTransitions 紧急变更：可以跳过多个步骤，快速实施。
var ChangeEmergencyTransitions = map[string][]string{
	common.ChangeStatusDraft:      {common.ChangeStatusSubmitted, common.ChangeStatusApproved, common.ChangeStatusInProgress, common.ChangeStatusCancelled},
	common.ChangeStatusSubmitted:  {common.ChangeStatusApproved, common.ChangeStatusRejected, common.ChangeStatusCancelled},
	common.ChangeStatusApproved:   {common.ChangeStatusInProgress, common.ChangeStatusCancelled},
	common.ChangeStatusInProgress: {common.ChangeStatusCompleted, common.ChangeStatusFailed, string(ChangeStatusRolledBack), common.ChangeStatusCancelled},
	common.ChangeStatusFailed:     {common.ChangeStatusScheduled, string(ChangeStatusRolledBack), common.ChangeStatusCancelled},
}

// ChangeNormalTransitions 普通变更：严格的 ITIL 流程。
var ChangeNormalTransitions = map[string][]string{
	common.ChangeStatusDraft:      {common.ChangeStatusSubmitted, common.ChangeStatusCancelled},
	common.ChangeStatusSubmitted:  {common.ChangeStatusApproved, common.ChangeStatusRejected, common.ChangeStatusCancelled},
	common.ChangeStatusApproved:   {common.ChangeStatusScheduled, common.ChangeStatusCancelled},
	common.ChangeStatusScheduled:  {common.ChangeStatusInProgress, common.ChangeStatusCancelled},
	common.ChangeStatusInProgress: {common.ChangeStatusCompleted, common.ChangeStatusFailed, string(ChangeStatusRolledBack), common.ChangeStatusCancelled},
	common.ChangeStatusFailed:     {common.ChangeStatusScheduled, string(ChangeStatusRolledBack), common.ChangeStatusCancelled},
}

// NormalizeChangeTransitionStatus 把待审批的两个同义词归一到迁移表词表（submitted）。
func NormalizeChangeTransitionStatus(status string) string {
	if status == string(ChangeStatusPending) {
		return common.ChangeStatusSubmitted
	}
	return status
}

// IsValidChangeStatusTransition 检查变更状态转换是否合法。
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
	currentStatus = NormalizeChangeTransitionStatus(currentStatus)
	newStatus = NormalizeChangeTransitionStatus(newStatus)

	baseTransitions := ChangeBaseTransitions

	var typeSpecificTransitions map[string][]string
	switch changeType {
	case string(ChangeTypeStandard):
		// 标准变更：预授权，可以跳过审批步骤
		typeSpecificTransitions = ChangeStandardTransitions
	case string(ChangeTypeEmergency):
		// 紧急变更：可以跳过多个步骤，快速实施
		typeSpecificTransitions = ChangeEmergencyTransitions
	default: // 普通变更：严格的ITIL流程
		typeSpecificTransitions = ChangeNormalTransitions
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

package common

import (
	"errors"
)

// CI 生命周期状态机（单一事实源）。
//
// 词表与 common/constants.go 的 CILifecycleStatus* 一致；scrapped 是终态兼
// 软删除态——读拦截器（database/security.go）对所有查询过滤
// lifecycle_status = scrapped，因此 scrapped 不可逆，恢复需 DBA 通道。
// 服务层（configuration_item_service）与 handlers/cmdb/data_governance 的
// 退役原语都从这里取矩阵，禁止各自再维护一份。
var ciLifecycleTransitions = map[string]map[string]struct{}{
	CILifecycleStatusDraft: {
		CILifecycleStatusOnline: {},
	},
	CILifecycleStatusOnline: {
		CILifecycleStatusMaintenance: {},
		CILifecycleStatusOffline:     {},
		CILifecycleStatusScrapped:    {},
	},
	CILifecycleStatusMaintenance: {
		CILifecycleStatusOnline:   {},
		CILifecycleStatusOffline:  {},
		CILifecycleStatusScrapped: {},
	},
	CILifecycleStatusOffline: {
		CILifecycleStatusOnline:      {},
		CILifecycleStatusMaintenance: {},
		CILifecycleStatusScrapped:    {},
	},
	CILifecycleStatusScrapped: {},
}

// CI 退役理由受控词表。退役（目标 scrapped）必须从词表选择理由，
// 保证审计字段可分类统计，自由文本放 remark。
var ciRetirementReasons = map[string]struct{}{
	"manual":            {},
	"decommissioned":    {},
	"replaced":          {},
	"discovery_missing": {},
	"end_of_life":       {},
}

var (
	ErrInvalidCILifecycleTransition = errors.New("invalid CI lifecycle transition")
	ErrUnknownCIRetirementReason    = errors.New("unknown CI retirement reason")
)

// IsValidCILifecycleStatus 判断状态是否在生命周期词表内。
func IsValidCILifecycleStatus(status string) bool {
	_, ok := ciLifecycleTransitions[status]
	return ok
}

// CanTransitionCILifecycle 判断 from -> to 是否为矩阵内合法迁移。
func CanTransitionCILifecycle(from, to string) bool {
	allowed, ok := ciLifecycleTransitions[from]
	if !ok {
		return false
	}
	_, ok = allowed[to]
	return ok
}

// CanRetireCILifecycle 校验处于 from 状态的 CI 能否以 reason 退役（目标 scrapped）：
// 状态必须在矩阵内允许直接迁往 scrapped，且 reason 在受控词表内。
func CanRetireCILifecycle(from, reason string) error {
	if !CanTransitionCILifecycle(from, CILifecycleStatusScrapped) {
		return ErrInvalidCILifecycleTransition
	}
	if _, ok := ciRetirementReasons[reason]; !ok {
		return ErrUnknownCIRetirementReason
	}
	return nil
}

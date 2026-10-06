package cmdb

import (
	"errors"
	"fmt"
	"time"
)

// CMDB 数据治理原语（v1.6.15）
//
// 目的：
//   - 把 CI 退役（retirement）、发现差异（diff）、治理质量（quality metrics）
//     从零散 handler 代码中抽离成可测试的纯函数；
//   - 退役状态机使用显式 allowed transition 表，避免散落条件分支；
//   - 差异分类返回受控 action 词表，落地时只能在该词表内选择；
//   - 质量指标输出 `CompletenessPct` 字段供治理看板渲染，0..100 保留两位小数。
//
// 本文件不依赖 ent / gin，可独立单测；落地事务与权限校验由 service 层调用方负责。

// CI 生命周期状态枚举。Schema 默认 `online`，退役流程经 `retiring` 中间态
// 收敛到 `retired`；从 `retired` 可显式恢复到 `online`，`online -> retiring`
// 是单向的入口。
const (
	CILifecycleOnline   = "online"
	CILifecycleRetiring = "retiring"
	CILifecycleRetired  = "retired"
)

// CI 退役理由常量词表。调用方写入时只能从该词表选择，避免审计出现自由文本。
var allowedRetirementReasons = map[string]struct{}{
	"manual":            {},
	"decommissioned":    {},
	"replaced":          {},
	"discovery_missing":  {},
	"end_of_life":       {},
}

// allowedRetirementTransitions 描述合法的 (from -> to) 退役状态机转移。
// 非法转移必须返回 ErrInvalidCIRetirementTransition，由 service 层映射为 409。
var allowedRetirementTransitions = map[string]map[string]struct{}{
	CILifecycleOnline: {
		CILifecycleRetiring: {},
	},
	CILifecycleRetiring: {
		CILifecycleRetired: {},
		CILifecycleOnline:  {},
	},
	CILifecycleRetired: {
		CILifecycleOnline: {},
	},
}

// ErrInvalidCIRetirementTransition 标识非法 CI 退役状态机转移。
var ErrInvalidCIRetirementTransition = errors.New("invalid CI retirement lifecycle transition")

// ErrUnknownRetirementReason 标识未登记的退役理由，强制走受控词表。
var ErrUnknownRetirementReason = errors.New("unknown CI retirement reason")

// CanRetireCI 校验给定 CI 是否可发起退役流程：
//   - 必传非 nil；
//   - 当前状态必须是 `online` 或 `retiring`；已退役 (`retired`) 不能再发起；
//   - 退役理由必须在受控词表内。
func CanRetireCI(ci *ConfigurationItem, reason string) error {
	if ci == nil {
		return errors.New("ci is nil")
	}
	switch ci.LifecycleStatus {
	case CILifecycleRetired:
		return fmt.Errorf("%w: already retired", ErrInvalidCIRetirementTransition)
	case CILifecycleOnline, CILifecycleRetiring:
		// 合法起始状态
	default:
		return fmt.Errorf("%w: unknown lifecycle %q", ErrInvalidCIRetirementTransition, ci.LifecycleStatus)
	}
	if _, ok := allowedRetirementReasons[reason]; !ok {
		return fmt.Errorf("%w: %q", ErrUnknownRetirementReason, reason)
	}
	return nil
}

// ApplyCIRetirement 把 CI 的 LifecycleStatus 推进到 target；同步更新 UpdatedAt 与
// Status（仅当进入终态 retired 时把 Status 同步为 "retired" 以便老 SQL/筛选器兼容）。
// 调用方负责把 ci 落库；纯函数不产生副作用，只修改入参对象。
func ApplyCIRetirement(ci *ConfigurationItem, target, reason string, now time.Time) error {
	if ci == nil {
		return errors.New("ci is nil")
	}
	allowed, ok := allowedRetirementTransitions[ci.LifecycleStatus]
	if !ok {
		return fmt.Errorf("%w: unknown lifecycle %q", ErrInvalidCIRetirementTransition, ci.LifecycleStatus)
	}
	if _, ok := allowed[target]; !ok {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidCIRetirementTransition, ci.LifecycleStatus, target)
	}
	ci.LifecycleStatus = target
	ci.UpdatedAt = now
	if target == CILifecycleRetired {
		ci.Status = CILifecycleRetired
	}
	return nil
}

// IsStaleCI 判断 CI 是否超过 threshold 时间未被任何发现源同步。仅在 CloudSyncTime
// 非零时判定为 true；从未发现的 CI（人工创建、零发现历史）不算陈旧。
func IsStaleCI(ci *ConfigurationItem, threshold time.Duration, now time.Time) bool {
	if ci == nil || ci.CloudSyncTime == nil || threshold <= 0 {
		return false
	}
	return now.Sub(*ci.CloudSyncTime) > threshold
}

// IsOrphanCI 判断 CI 是否既无 owner/assignee 又无任何发现/手工来源登记。
// 用于治理看板的"孤儿资产"分类。
func IsOrphanCI(ci *ConfigurationItem) bool {
	if ci == nil {
		return false
	}
	return ci.OwnedBy == "" && ci.AssignedTo == "" && ci.DiscoverySource == "" && ci.Source == ""
}

// QualityMetrics 描述单租户 CMDB 健康快照。
type QualityMetrics struct {
	Total           int     `json:"total"`
	Active          int     `json:"active"`
	Retired         int     `json:"retired"`
	Stale           int     `json:"stale"`
	Orphan          int     `json:"orphan"`
	Incomplete      int     `json:"incomplete"`
	CompletenessPct float64 `json:"completenessPct"`
}

// ComputeQualityMetrics 计算给定 CI 集合的治理指标。threshold 仅对
// IsStaleCI 生效；空集合返回全 0 指标，避免前端除零。CompletenessPct
// 按 100.00 上界四舍五入到 0.01。
func ComputeQualityMetrics(cis []*ConfigurationItem, threshold time.Duration, now time.Time) QualityMetrics {
	m := QualityMetrics{Total: len(cis)}
	for _, ci := range cis {
		if ci == nil {
			continue
		}
		if ci.LifecycleStatus == CILifecycleRetired {
			m.Retired++
		} else {
			m.Active++
		}
		if IsStaleCI(ci, threshold, now) {
			m.Stale++
		}
		if IsOrphanCI(ci) {
			m.Orphan++
		}
		if ci.DiscoverySource == "" && ci.Source == "" && ci.OwnedBy == "" {
			m.Incomplete++
		}
	}
	if m.Total > 0 {
		complete := m.Total - m.Incomplete
		ratio := float64(complete) / float64(m.Total) * 100
		m.CompletenessPct = float64(int(ratio*100+0.5)) / 100
	}
	return m
}

// ReconciliationAction 受控差异分类词表。
type ReconciliationAction string

const (
	ReconcileActionAdd           ReconciliationAction = "add"
	ReconcileActionNoOp          ReconciliationAction = "noop"
	ReconcileActionRetireConfirm ReconciliationAction = "retire_confirm"
	ReconcileActionDuplicate     ReconciliationAction = "duplicate"
)

// ReconciliationEntry 单条已分类的差异行。
type ReconciliationEntry struct {
	CloudResourceID int                  `json:"cloudResourceId"`
	CINumber        string               `json:"ciNumber,omitempty"`
	ExistingCIID    int                  `json:"existingCiId,omitempty"`
	Action          ReconciliationAction `json:"action"`
	Reason          string               `json:"reason"`
}

// DiffReconciliation 把"发现侧资源列表"对照"权威侧 CI 列表"做 4 类分类：
//   - add           ：发现侧存在但 CI 侧无 CloudResourceRefID 指向；
//   - noop          ：已被同 CloudResourceRefID 的 CI 关联（指纹未变）；
//   - retire_confirm：上次发现仍存在的 CI 在本批次发现结果中消失（调用方按 CI 列表传入）；
//   - duplicate     ：本批次内同一 CloudResourceID 出现多次（不应发生，作为告警）。
//
// 函数无副作用，调用方负责持久化与事务边界。
func DiffReconciliation(cis []*ConfigurationItem, resources []*CloudResource) []ReconciliationEntry {
	const (
		reasonAlreadyLinked      = "already linked via CloudResourceRefID"
		reasonNewCloudResource   = "no existing CI references this cloud resource"
		reasonDuplicateInBatch   = "cloud resource id seen twice in batch"
		reasonRetireConfirmation = "existing CI no longer present in latest discovery batch"
	)
	existingByRef := make(map[int]*ConfigurationItem, len(cis))
	for _, ci := range cis {
		if ci == nil || ci.CloudResourceRefID <= 0 {
			continue
		}
		existingByRef[ci.CloudResourceRefID] = ci
	}
	seenRefs := make(map[int]struct{}, len(resources))
	out := make([]ReconciliationEntry, 0, len(resources))
	for _, res := range resources {
		if res == nil {
			continue
		}
		if _, dup := seenRefs[res.ID]; dup {
			out = append(out, ReconciliationEntry{
				CloudResourceID: res.ID,
				Action:          ReconcileActionDuplicate,
				Reason:          reasonDuplicateInBatch,
			})
			continue
		}
		seenRefs[res.ID] = struct{}{}
		if existing, ok := existingByRef[res.ID]; ok {
			out = append(out, ReconciliationEntry{
				CloudResourceID: res.ID,
				CINumber:        existing.CINumber,
				ExistingCIID:    existing.ID,
				Action:          ReconcileActionNoOp,
				Reason:          reasonAlreadyLinked,
			})
		} else {
			out = append(out, ReconciliationEntry{
				CloudResourceID: res.ID,
				Action:          ReconcileActionAdd,
				Reason:          reasonNewCloudResource,
			})
		}
	}
	for refID, existing := range existingByRef {
		if _, seen := seenRefs[refID]; seen {
			continue
		}
		out = append(out, ReconciliationEntry{
			CloudResourceID: refID,
			CINumber:        existing.CINumber,
			ExistingCIID:    existing.ID,
			Action:          ReconcileActionRetireConfirm,
			Reason:          reasonRetireConfirmation,
		})
	}
	return out
}

// CountByAction 汇总一次 DiffReconciliation 结果，按 action 词表分桶计数。
// 词表外的 action 不计入，便于上层审计。
func CountByAction(entries []ReconciliationEntry) map[ReconciliationAction]int {
	out := map[ReconciliationAction]int{
		ReconcileActionAdd:           0,
		ReconcileActionNoOp:          0,
		ReconcileActionRetireConfirm: 0,
		ReconcileActionDuplicate:     0,
	}
	for _, e := range entries {
		if _, known := out[e.Action]; !known {
			continue
		}
		out[e.Action]++
	}
	return out
}

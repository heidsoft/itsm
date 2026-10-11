package cmdb

import (
	"time"

	"itsm-backend/common"
)

// CMDB 数据治理原语。
//
// CI 退役状态机与理由词表统一收敛在 common/ci_lifecycle.go（生产路径由
// service.UpdateLifecycleStatus 消费）；本文件只保留治理看板所需的纯函数：
// 陈旧/孤儿判定、质量指标与发现差异分类。
//
// 本文件不依赖 ent / gin，可独立单测；落地事务与权限校验由 service 层调用方负责。

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
		if ci.LifecycleStatus == common.CILifecycleStatusScrapped {
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

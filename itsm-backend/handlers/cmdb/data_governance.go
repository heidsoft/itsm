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
//
// 🔴 Incomplete / CompletenessPct 的语义已改为「属性级完整率」，计算入口是
// attribute_quality.go 的 ComputeAttributeQuality（按该 CI 类型的必填属性定义
// 判定）。本结构只承载结果，不再由 ComputeQualityMetrics 产出——
// 后者保留仅为兼容旧单测与其 owner 覆盖率口径，生产路径不再调用。
// 判据差别很关键：旧口径把「有 owner/来源」当完整，与属性填了多少无关。
type QualityMetrics struct {
	Total   int `json:"total"`
	Active  int `json:"active"`
	Retired int `json:"retired"`
	Stale   int `json:"stale"`
	Orphan  int `json:"orphan"`
	// Incomplete 是必填属性未齐的 CI 数。
	Incomplete int `json:"incomplete"`
	// CompletenessPct 只在 Assessable 内有意义；Assessable=0 时恒为 0（不可评估）。
	CompletenessPct float64 `json:"completenessPct"`
	// Assessable 是有必填属性定义的 CI 数（完整率分母）。
	Assessable int `json:"assessable"`
	// AttributeDefinedTypes 是参与评估的 CI 类型数，用于区分「完整率低」与
	// 「根本没配属性定义」。
	AttributeDefinedTypes int `json:"attributeDefinedTypes"`
	// MissingByAttribute 汇总最缺的必填属性，按缺失次数降序。
	MissingByAttribute []AttributeMissingStat `json:"missingByAttribute"`
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

// CertificationMetrics 是「数据认证新鲜度」维度的治理指标（CMDB Health 的
// Compliance）。与完整率是两件独立的事：完整率问「必填属性填齐了吗」，
// 认证问「这些值还有人担保是真的吗」——一条属性全都填齐、但三年没人确认过的
// CI，完整率高而可信度为零。
//
// 🔴 分母口径刻意与完整率不同，不能合并：
//   - 完整率的分母 Assessable = 「该 CI 类型有必填属性定义」的 CI 数；
//   - 认证的分母 Verified 在此之上再排除退役 CI。
//
// 退役资产不该被要求「确认它仍然准确」——它已经不在了，要求确认只会把
// 认证率永久拉低，让运维对这个指标失去信任。这个差异必须写在结构里，
// 否则下一个人很容易「顺手统一口径」。
type CertificationMetrics struct {
	// Verified 是认证率的有效分母：在用（非 scrapped）且可评估的 CI 数。
	Verified int `json:"verified"`
	// NeverConfirmed 是从未被任何人在任何时点确认过的 CI 数。
	// 它不是「过期」——过期是「确认过但太久了」，两者对运维的含义不同：
	// 从未确认说明这条资产**从来没人认领过责任**，是治理的空白区，优先级更高。
	NeverConfirmed int `json:"neverConfirmed"`
	// StaleConfirmed 是确认过但距今超过 freshThreshold 的 CI 数。
	StaleConfirmed int `json:"staleConfirmed"`
	// FreshConfirmed 是确认过且在阈值内的 CI 数（认证率的分子）。
	FreshConfirmed int `json:"freshConfirmed"`
	// FreshnessPct = FreshConfirmed / Verified，0.01 精度，上界 100。
	// Verified=0 时恒为 0（不可评估），**不可解读为「全部过期」**。
	FreshnessPct float64 `json:"freshnessPct"`
	// FreshThresholdDays 本次判定使用的阈值天数。随响应下发而不是只写在文档里：
	// 前端要在卡片上写「确认于 N 天内视为新鲜」，硬编码 90 会在后端调阈值后
	// 静默说谎。
	FreshThresholdDays int `json:"freshThresholdDays"`
}

// ComputeCertificationMetrics 计算确认新鲜度。
//
// freshThreshold 是「确认多久之内算新鲜」；now 由调用方传入（便于测试固定时间）。
//
// 三个计数互斥且完备：每条可评估 CI 必落在 NeverConfirmed / StaleConfirmed /
// FreshConfirmed 之一，三者之和恒等于 Verified。这条不变量是下游能放心把
// 三个数字画成饼图/堆叠条的前提，由单测锁定。
func ComputeCertificationMetrics(cis []*ConfigurationItem, freshThreshold time.Duration, now time.Time) CertificationMetrics {
	m := CertificationMetrics{}
	if freshThreshold <= 0 {
		freshThreshold = 0
	}
	m.FreshThresholdDays = int(freshThreshold.Hours() / 24)

	for _, ci := range cis {
		if ci == nil {
			continue
		}
		// 退役 CI 不进分母：要求「已退役的资产」确认仍然准确是无意义的要求。
		if ci.LifecycleStatus == common.CILifecycleStatusScrapped {
			continue
		}
		m.Verified++
		switch {
		case ci.LastConfirmedAt == nil || ci.LastConfirmedAt.IsZero():
			m.NeverConfirmed++
		case now.Sub(*ci.LastConfirmedAt) > freshThreshold:
			m.StaleConfirmed++
		default:
			m.FreshConfirmed++
		}
	}

	if m.Verified > 0 {
		ratio := float64(m.FreshConfirmed) / float64(m.Verified) * 100
		m.FreshnessPct = float64(int(ratio*100+0.5)) / 100
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

package change

// 审批链 quorum 求值的**唯一**实现。
//
// 背景（2026-10-22 收敛）：会签（parallel/all）与阈值（N of M）语义此前只写在
// legacy 复核器 checkAndTransitionChange 里，而该函数仅被无路由的 ProcessApproval
// 调用 —— 生产走的是 TransitionStatus，它在桥接完成后直接把状态 CAS 成 approved，
// 于是「两人会签」的变更被一个人就批掉了。求值逻辑抽成纯函数后由 TransitionStatus
// 唯一消费；复核器本身已于 2026-10-23 连同 legacy 入口的最后残骸一并删除。

// levelQuorum 是单个审批层的会签快照。
type levelQuorum struct {
	level     int
	required  bool
	approvers map[int]struct{}
	approved  map[int]struct{}
	rejected  bool
	// threshold 为该层通过所需的有效批准票数（会签人数 / N of M / 或签 1）。
	threshold int
	// approvalType 保留原始类型，用于缺省阈值推导（parallel/all = 全体，其余 = 1）。
	approvalType string
}

// ApprovalQuorumOutcome 汇总整条审批链的求值结果。
type ApprovalQuorumOutcome struct {
	// ChainEmpty 表示该变更没有配置审批链（标准变更预授权 / 免审路径）：
	// 此时不挡批准，交由状态机与业务门禁决定。
	ChainEmpty bool
	// Rejected 表示必需层已出现驳回票，整条链应判为拒绝。
	Rejected bool
	// AllRequiredMet 表示所有必需层的有效批准票均达到该层阈值。
	AllRequiredMet bool
	// ApprovedVotes / RequiredVotes 是跨层的票数摘要，用于日志与"还差几票"的
	// 可行动提示（不参与判定）。
	ApprovedVotes int
	RequiredVotes int
	// MissingLevels 列出尚未达标的层级（升序），便于把原因回给调用方。
	MissingLevels []int
}

// applyChainLevels 把审批链的层级信息补到审批决定上。
//
// quorum 按 (approver, level) 双重匹配计票，缺少 Levels 的决定会被静默丢弃
// （等于没投票）；ProcessApprovalDecision 本身不带层级，必须由审批链派生。
func applyChainLevels(chain []*ApprovalChain, decisions []*ApprovalRecord) {
	levelsByApprover := make(map[int][]int, len(chain))
	for _, item := range chain {
		levelsByApprover[item.ApproverID] = append(levelsByApprover[item.ApproverID], item.Level)
	}
	for _, d := range decisions {
		if len(d.Levels) == 0 {
			d.Levels = levelsByApprover[d.ApproverID]
		}
	}
}

// evaluateApprovalQuorum 按审批链与审批历史判定整体去向。
//
// 规则（与 ITIL 会签语义一致）：
//  1. 按 (approver, level) 双重匹配计票——只有该审批人确实属于该层时票才计入，
//     避免一次性审批跨层串票；
//  2. 任一层出现驳回（且该层被视为必需）⇒ 整体 Rejected；
//  3. 所有必需层的批准票数达到该层阈值 ⇒ AllRequiredMet；
//  4. 没有层被标记 required 时，**按全部层都是必需层**处理。审批链存在却没有任何
//     必需层，若按字面放行等于审批链形同虚设，故 fail-closed。
func evaluateApprovalQuorum(chain []*ApprovalChain, history []*ApprovalRecord) ApprovalQuorumOutcome {
	if len(chain) == 0 {
		return ApprovalQuorumOutcome{ChainEmpty: true, AllRequiredMet: true}
	}

	levels := map[int]*levelQuorum{}
	order := make([]int, 0, len(chain))
	for _, item := range chain {
		agg, ok := levels[item.Level]
		if !ok {
			agg = &levelQuorum{
				level:     item.Level,
				approvers: map[int]struct{}{},
				approved:  map[int]struct{}{},
			}
			levels[item.Level] = agg
			order = append(order, item.Level)
		}
		agg.approvers[item.ApproverID] = struct{}{}
		if item.IsRequired {
			agg.required = true
		}
		// 同一层可能由多条链步骤组成（多审批人），阈值为层级属性；
		// 有显式 threshold 时以最大值为准（重复写同一值不影响）。
		if item.Threshold > agg.threshold {
			agg.threshold = item.Threshold
		}
		if item.ApprovalType != "" {
			agg.approvalType = item.ApprovalType
		}
	}

	// 阈值缺省推导：parallel/all 需全体候选，serial/or 只需 1 票。
	for _, lvl := range order {
		agg := levels[lvl]
		if agg.threshold <= 0 {
			if agg.approvalType == "parallel" || agg.approvalType == "all" {
				agg.threshold = len(agg.approvers)
			} else {
				agg.threshold = 1
			}
		}
		if agg.threshold <= 0 {
			agg.threshold = 1
		}
	}

	// 计票：只认该审批人所属层级的决定。
	for _, h := range history {
		for _, lvl := range h.Levels {
			agg, ok := levels[lvl]
			if !ok {
				continue
			}
			if _, member := agg.approvers[h.ApproverID]; !member {
				continue
			}
			switch h.Status {
			case "approved":
				agg.approved[h.ApproverID] = struct{}{}
			case "rejected":
				agg.rejected = true
			}
		}
	}

	hasRequired := false
	for _, lvl := range order {
		if levels[lvl].required {
			hasRequired = true
			break
		}
	}

	out := ApprovalQuorumOutcome{AllRequiredMet: true}
	for _, lvl := range order {
		agg := levels[lvl]
		considered := agg.required || !hasRequired
		if !considered {
			continue
		}
		out.RequiredVotes += agg.threshold
		if len(agg.approved) > agg.threshold {
			out.ApprovedVotes += agg.threshold
		} else {
			out.ApprovedVotes += len(agg.approved)
		}
		if agg.rejected {
			out.Rejected = true
		}
		if len(agg.approved) < agg.threshold {
			out.AllRequiredMet = false
			out.MissingLevels = append(out.MissingLevels, agg.level)
		}
	}
	return out
}

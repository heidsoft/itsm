package dto

// 审批链求值/影子比对的值类型（原定义在共享层 service/approval_chain_*.go）。
// 唯一实现在本包；service 侧保留同名别名，类型身份与字段形状完全一致。

// ApprovalEvalContext 审批链求值上下文。
type ApprovalEvalContext struct {
	TenantID    int
	EntityType  string // ticket, change, service_request
	RequesterID int
	Priority    string
	Amount      float64
}

// ApprovalLevelEval 单层（一个 level）求值结果。
type ApprovalLevelEval struct {
	Level             int
	Required          bool
	ApprovalType      string // serial | parallel | all | or
	Threshold         int    // 需要的批准人数
	ApproverIDs       []int
	ApproverNames     []string
	FallbackTriggered bool
	FallbackAction    string
	Status            string // pending | satisfied | blocked
	// i3 P0 动态适配：上下文不匹配条件时该层被跳过，视为自动通过。
	Skipped    bool   // 本层是否被条件过滤跳过
	SkipReason string // 跳过原因，如 "priority_not_match"/"amount_below_min"/"amount_above_max"
}

// ApprovalChainEvaluation 一条审批链的完整求值结果。
type ApprovalChainEvaluation struct {
	ChainID      int
	Levels       []ApprovalLevelEval
	Passed       bool // 所有必需层均满足（含 fallback 解决）
	PendingLevel int  // 下一个待审批层（1-based），0=已完成/无待办
	Blocked      bool // 触发 block/auto_reject，需人工干预
}

// ShadowResult 影子双轨对比结果（新旧路径差异记录）。
type ShadowResult struct {
	ChainID      int
	Passed       bool
	PendingLevel int
	Blocked      bool
	LevelsJSON   []byte
	Err          error
}

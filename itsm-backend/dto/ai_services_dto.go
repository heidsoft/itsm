package dto

import "time"

// TriageResult 分诊建议结果（API JSON 形状）。唯一实现在本包；
// service 侧保留同名别名，类型身份与 JSON 形状与本类型完全一致。
//
// Model/PromptVersion 是溯源字段：Model 标注决策来源链路（guidance/llm/keyword），
// PromptVersion 是口径版本，两者随响应透出并写入服务端 AI 审计底账。
// ModelName 是具体模型名：keyword/guidance 路径没有模型参与，字段留空。
// Degraded/DegradedReason 标注「这条建议没有模型参与」：降级结果本身不丢人，
// 冒充模型结果才丢人，故降级路径一律置 Degraded=true 并随响应透出。
type TriageResult struct {
	Category       string  `json:"category"`
	Priority       string  `json:"priority"`
	AssigneeID     int     `json:"assigneeId"`
	Confidence     float64 `json:"confidence"`
	Explanation    string  `json:"explanation"`
	SuggestedFix   string  `json:"suggestedFix,omitempty"`
	Model          string  `json:"model,omitempty"`
	ModelName      string  `json:"modelName,omitempty"`
	PromptVersion  string  `json:"promptVersion,omitempty"`
	Degraded       bool    `json:"degraded,omitempty"`
	DegradedReason string  `json:"degradedReason,omitempty"`
}

// AIMetrics AI 指标聚合结果（API JSON 形状）。前端 TS 类型 AIMetrics 字段对齐，
// 避免 map[string]interface{} 拼写漂移。
type AIMetrics struct {
	TotalRequests          int                    `json:"totalRequests"`
	TotalFeedback          int                    `json:"totalFeedback"`
	UsefulFeedback         int                    `json:"usefulFeedback"`
	UsefulRate             float64                `json:"usefulRate"`
	ByKind                 map[string]interface{} `json:"byKind"`
	AvgResponseTimeSeconds float64                `json:"avgResponseTimeSeconds"`
	LLMCallCount           int                    `json:"llmCallCount"`
	TotalTokens            int                    `json:"totalTokens"`
	ResponseTimeAvailable  bool                   `json:"responseTimeAvailable"`
	// Quota 月度配额用量（nil = 配额服务未装配，如实省略）。
	Quota *QuotaUsage `json:"quota,omitempty"`
	// Cost 成本估算口径（nil = 配额服务未装配）。
	Cost *CostEstimate `json:"cost,omitempty"`
}

// QuotaUsage 配额用量评估结果。
type QuotaUsage struct {
	QuotaConfigured bool    `json:"quotaConfigured"`
	MonthlyQuota    int64   `json:"monthlyQuota"`
	TokensUsed      int64   `json:"tokensUsed"`
	TokensRemaining int64   `json:"tokensRemaining"`
	UsedRatio       float64 `json:"usedRatio"`
	// Alert: "" | "warning" | "exceeded"
	Alert string `json:"alert"`
	// AlertMessage 人类可读告警文案（Alert 非空时才有意义）。
	AlertMessage string `json:"alertMessage,omitempty"`
}

// CostEstimate 按单价口径估算成本。
type CostEstimate struct {
	CostConfigured bool    `json:"costConfigured"`
	TotalTokens    int64   `json:"totalTokens"`
	EstimatedCost  float64 `json:"estimatedCost"`
	Currency       string  `json:"currency"`
	PricePer1K     float64 `json:"pricePer1k"`
}

// SummaryWithMetadata 带溯源元数据的工单摘要。
type SummaryWithMetadata struct {
	Summary   string        `json:"summary"`
	Method    string        `json:"method"` // "llm" | "direct" | "fallback"
	Latency   time.Duration `json:"latencyMs"`
	CharCount int           `json:"charCount"`
	// 溯源三字段
	PromptVersion string  `json:"promptVersion"`
	Model         string  `json:"model,omitempty"` // 空 = 无模型参与（direct/fallback）
	Confidence    float64 `json:"confidence"`      // 摘要场景恒为 0，见类型注释
}

package ai

import (
	"time"
)

// Conversation represents a chat session with AI
type Conversation struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	UserID    int       `json:"userId"`
	TenantID  int       `json:"tenantId"`
	CreatedAt time.Time `json:"createdAt"`
}

// Message represents a single message in a conversation
type Message struct {
	ID             int       `json:"id"`
	ConversationID int       `json:"conversationId"`
	Role           string    `json:"role"` // user, assistant, system
	Content        string    `json:"content"`
	RequestID      string    `json:"requestId"`
	CreatedAt      time.Time `json:"createdAt"`
}

// ToolInvocation represents an AI tool execution
type ToolInvocation struct {
	ID             int        `json:"id"`
	TenantID       int        `json:"tenantId"`
	ConversationID int        `json:"conversationId"`
	ToolName       string     `json:"toolName"`
	Arguments      string     `json:"arguments"` // JSON string
	Status         string     `json:"status"`    // pending, running, done, failed
	Result         *string    `json:"result"`
	Error          *string    `json:"error"`
	NeedsApproval  bool       `json:"needsApproval"`
	ApprovalState  string     `json:"approvalState"` // pending, approved, rejected
	ApprovedBy     int        `json:"approvedBy"`
	ApprovalReason string     `json:"approvalReason"`
	ApprovedAt     *time.Time `json:"approvedAt"`
	RequestID      string     `json:"requestId"`
	CreatedAt      time.Time  `json:"createdAt"`
	// AI 工具 RBAC 校验 审计字段
	UserID           int    `json:"userId"`
	PermissionCheck  string `json:"permissionCheck"`  // passed|denied|skipped
	PermissionReason string `json:"permissionReason"` // 校验/拒绝原因
	RoleSnapshot     string `json:"roleSnapshot"`     // 调用时角色快照
	// Phase B6（2026-11-17）：业务单据强关联（可空，0 表示无关联）
	TicketID int `json:"ticketId"`
	ChangeID int `json:"changeId"`
	// 审计回放（2026-11-21）：策略版本与运行轮次。
	//
	// PolicyVersion 三态：
	//   ""        → 本次判定无策略参与（human 主体走角色 RBAC）；
	//   "unknown" → 有策略语义但读不出来（已 fail-closed 拒绝）；
	//   32 位 hex → 决策输入指纹，可与 DB 现值重算比对。
	//
	// RunID 空串 → 不属于任何可标识的运行轮次。
	PolicyVersion string `json:"policyVersion"`
	RunID         string `json:"runId"`
	// DryRun 标记这条记录是「预演」而非真实调用（2026-11-21 起真实写入）。
	//
	// 语义：预演跑完完整的判定链（Gate 2 角色 RBAC + Gate 2.5 主体策略 +
	// 审批人数计算），但**不执行工具、不创建待审批行**，只把决策结论落成一条
	// dry_run=true 的审计记录。
	//
	// 为什么必须独立成列而不是靠 status 或 approval_state 区分：
	//   - status 表达「执行结果」（success/failed/...），预演没有执行结果；
	//   - approval_state 表达「审批走到哪一步」，预演根本不进审批流。
	// 用其中任何一个兼职，都会让审批列表/执行统计把预演当成真实调用计数。
	// 因此 dry_run 行的 approval_state 恒为 "none"，且被待办查询显式排除。
	DryRun bool `json:"dryRun"`
}

// RootCauseAnalysis represents an RCA record for a ticket
type RootCauseAnalysis struct {
	ID              int                      `json:"id"`
	TicketID        int                      `json:"ticketId"`
	TicketNumber    string                   `json:"ticketNumber"`
	TicketTitle     string                   `json:"ticketTitle"`
	AnalysisDate    string                   `json:"analysisDate"`
	RootCauses      []map[string]interface{} `json:"rootCauses"`
	AnalysisSummary string                   `json:"analysisSummary"`
	ConfidenceScore float64                  `json:"confidenceScore"`
	AnalysisMethod  string                   `json:"analysisMethod"`
	TenantID        int                      `json:"tenantId"`
	CreatedAt       time.Time                `json:"createdAt"`
	UpdatedAt       time.Time                `json:"updatedAt"`
}

// AIAnalysisResult stores AI analysis outputs for tickets/incidents.
type AIAnalysisResult struct {
	ID              int       `json:"id"`
	TenantID        int       `json:"tenantId"`
	UserID          int       `json:"userId"`
	AnalysisType    string    `json:"analysisType"` // triage|summary|rca|deep_analytics|trend_prediction|incident_impact
	TicketID        int       `json:"ticketId,omitempty"`
	IncidentID      int       `json:"incidentId,omitempty"`
	TicketNumber    string    `json:"ticketNumber,omitempty"`
	TicketTitle     string    `json:"ticketTitle,omitempty"`
	RequestPrompt   string    `json:"requestPrompt"`
	ResultJSON      string    `json:"resultJson"`
	Model           string    `json:"model,omitempty"`
	LatencyMs       int       `json:"latencyMs,omitempty"`
	TotalTokens     int       `json:"totalTokens,omitempty"`
	CostUSD         float64   `json:"costUsd,omitempty"`
	ConfidenceScore float64   `json:"confidenceScore,omitempty"`
	Degraded        bool      `json:"degraded"`
	CreatedAt       time.Time `json:"createdAt"`
}

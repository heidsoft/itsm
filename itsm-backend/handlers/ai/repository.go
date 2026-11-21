package ai

import (
	"context"
)

// Repository interface for AI domain
type Repository interface {
	// Conversations
	CreateConversation(ctx context.Context, c *Conversation) (*Conversation, error)
	// GetConversation 按 id + 租户 + 属主读会话。
	//
	// 必须带 userID：会话是**用户私有**的（ListConversations 也按 userID 过滤），
	// 只校验租户意味着同租户内 A 用户可读 B 用户的对话内容。租户是资源边界，
	// 用户是所有权边界，两者都要校验。
	GetConversation(ctx context.Context, id int, tenantID int, userID int) (*Conversation, error)
	ListConversations(ctx context.Context, tenantID int, userID int) ([]*Conversation, error)
	DeleteConversation(ctx context.Context, id int, tenantID int, userID int) error

	// Messages
	CreateMessage(ctx context.Context, m *Message) (*Message, error)
	// GetMessages 读会话内的消息，必须带租户谓词。
	//
	// message 表自身没有 tenant_id 字段（消息挂在会话上），所以租户过滤要
	// 经 conversation 边做；不带就是无边界读取——任何已知 conversation_id
	// 都能被读走。
	GetMessages(ctx context.Context, conversationID int, tenantID int) ([]*Message, error)

	// Tool Invocations
	CreateToolInvocation(ctx context.Context, i *ToolInvocation) (*ToolInvocation, error)
	GetToolInvocation(ctx context.Context, id int, tenantID int) (*ToolInvocation, error)
	DecideToolInvocation(ctx context.Context, id, tenantID, actorID int, approve bool, reason string) (*ToolInvocation, error)
	ListToolInvocations(ctx context.Context, tenantID int, state string) ([]*ToolInvocation, error)
	// CountToolInvocationsByApprovalState 按审批态分组计数（键 = approval_state）。
	//
	// 为什么必须新增而不是复用 ListToolInvocations 逐态调用：
	// ListToolInvocations 的 state="pending" 是**业务语义的合并查询**——它一次
	// 返回 pending / pending_agent / pending_agent_two / awaiting_second 四种态
	// （审批页需要一次看全，否则高危操作会卡在 awaiting_second 无人处理）。
	// 拿它当「单态计数」用，Pending 会变成四态之和，而其余三态各自又精确，
	// 于是 Total 被重复累加（实测：期望 10、实得 19）。
	//
	// 合并语义不能改（审批页依赖它），所以精确分组只能另开一条读面。
	// 顺带解决一个真实开销：为计数把整行捞回来，待批队列一大就是
	// 「查 500 行只为了知道 500 这个数字」。
	//
	// dry_run 排除规则必须在这里与 ListToolInvocations 保持一致——
	// 预演行不是调用，任何队列视图都不该把它算进待办。
	CountToolInvocationsByApprovalState(ctx context.Context, tenantID int) (map[string]int, error)
	// ListToolInvocationsByRun 按运行轮次取出该轮全部工具调用（含只读审计行）。
	//
	// 存在的理由：审计要求「不可抵赖 + 可回放」，而一次 Agent 会话的多次调用
	// 此前只能按 id 逐条看。run_id 贯穿后，这一条查询就是「这个 Agent 这一轮
	// 做了什么」的唯一读面——判定依据（policy_version）、审批态、结果一次取出。
	//
	// 必须带 tenant_id 谓词：run_id 是客户端可影响的字符串，仅按 run 查等于
	// 允许跨租户读取他人运行记录。
	ListToolInvocationsByRun(ctx context.Context, tenantID int, runID string) ([]*ToolInvocation, error)

	// Root Cause Analysis
	CreateRCA(ctx context.Context, r *RootCauseAnalysis) (*RootCauseAnalysis, error)
	GetRCAByTicket(ctx context.Context, ticketID int, tenantID int) (*RootCauseAnalysis, error)
	UpdateRCA(ctx context.Context, r *RootCauseAnalysis) (*RootCauseAnalysis, error)

	// AI Analysis Results
	SaveAIAnalysisResult(ctx context.Context, r *AIAnalysisResult) (*AIAnalysisResult, error)
	ListAIAnalysisResults(ctx context.Context, tenantID int, analysisType string, limit int) ([]*AIAnalysisResult, error)
	GetAIAnalysisResult(ctx context.Context, id int, tenantID int) (*AIAnalysisResult, error)
	DeleteAIAnalysisResult(ctx context.Context, id int, tenantID int) error
	// GetLatestAIAnalysisResultByTicket 返回该工单指定分析类型的最近一条结果。
	//
	// 用于「按工单幂等」：同一工单重复进入详情页时复用既有分诊结论，
	// 不再重复调 LLM（prod 实测 F2：每次进详情都重跑一次，分类还会漂移）。
	GetLatestAIAnalysisResultByTicket(ctx context.Context, ticketID int, tenantID int, analysisType string) (*AIAnalysisResult, error)
}

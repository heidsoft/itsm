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

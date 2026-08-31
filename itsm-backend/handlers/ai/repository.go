package ai

import (
	"context"
)

// Repository interface for AI domain
type Repository interface {
	// Conversations
	CreateConversation(ctx context.Context, c *Conversation) (*Conversation, error)
	GetConversation(ctx context.Context, id int, tenantID int) (*Conversation, error)
	ListConversations(ctx context.Context, tenantID int, userID int) ([]*Conversation, error)
	DeleteConversation(ctx context.Context, id int, tenantID int) error

	// Messages
	CreateMessage(ctx context.Context, m *Message) (*Message, error)
	GetMessages(ctx context.Context, conversationID int) ([]*Message, error)

	// Tool Invocations
	CreateToolInvocation(ctx context.Context, i *ToolInvocation) (*ToolInvocation, error)
	GetToolInvocation(ctx context.Context, id int, tenantID int) (*ToolInvocation, error)
	UpdateToolInvocation(ctx context.Context, i *ToolInvocation) (*ToolInvocation, error)
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
}

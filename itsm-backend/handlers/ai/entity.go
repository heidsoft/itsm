package ai

import (
	"time"
)

// Conversation represents a chat session with AI
type Conversation struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	UserID    int       `json:"user_id"`
	TenantID  int       `json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Message represents a single message in a conversation
type Message struct {
	ID             int       `json:"id"`
	ConversationID int       `json:"conversation_id"`
	Role           string    `json:"role"` // user, assistant, system
	Content        string    `json:"content"`
	RequestID      string    `json:"request_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// ToolInvocation represents an AI tool execution
type ToolInvocation struct {
	ID             int        `json:"id"`
	ConversationID int        `json:"conversation_id"`
	ToolName       string     `json:"tool_name"`
	Arguments      string     `json:"arguments"` // JSON string
	Status         string     `json:"status"`    // pending, running, completed, failed
	Result         *string    `json:"result"`
	Error          *string    `json:"error"`
	NeedsApproval  bool       `json:"needs_approval"`
	ApprovalState  string     `json:"approval_state"` // pending, approved, rejected
	ApprovedBy     int        `json:"approved_by"`
	ApprovalReason string     `json:"approval_reason"`
	ApprovedAt     *time.Time `json:"approved_at"`
	RequestID      string     `json:"request_id"`
	CreatedAt      time.Time  `json:"created_at"`
}

// RootCauseAnalysis represents an RCA record for a ticket
type RootCauseAnalysis struct {
	ID              int                      `json:"id"`
	TicketID        int                      `json:"ticket_id"`
	TicketNumber    string                   `json:"ticket_number"`
	TicketTitle     string                   `json:"ticket_title"`
	AnalysisDate    string                   `json:"analysis_date"`
	RootCauses      []map[string]interface{} `json:"root_causes"`
	AnalysisSummary string                   `json:"analysis_summary"`
	ConfidenceScore float64                  `json:"confidence_score"`
	AnalysisMethod  string                   `json:"analysis_method"`
	TenantID        int                      `json:"tenant_id"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

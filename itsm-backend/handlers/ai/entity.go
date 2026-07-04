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
	ConversationID int        `json:"conversationId"`
	ToolName       string     `json:"toolName"`
	Arguments      string     `json:"arguments"` // JSON string
	Status         string     `json:"status"`    // pending, running, completed, failed
	Result         *string    `json:"result"`
	Error          *string    `json:"error"`
	NeedsApproval  bool       `json:"needsApproval"`
	ApprovalState  string     `json:"approvalState"` // pending, approved, rejected
	ApprovedBy     int        `json:"approvedBy"`
	ApprovalReason string     `json:"approvalReason"`
	ApprovedAt     *time.Time `json:"approvedAt"`
	RequestID      string     `json:"requestId"`
	CreatedAt      time.Time  `json:"createdAt"`
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

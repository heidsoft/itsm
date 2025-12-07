package dto

import "time"

// 工单根因分析相关DTO（使用TicketRootCause前缀避免与problem_investigation_dto.go中的RootCauseAnalysis冲突）
type TicketRootCauseAnalysisRequest struct {
	TicketID int `json:"ticket_id" binding:"required" example:"1001"`
}

type TicketRootCauseAnalysisResponse struct {
	TicketID        int                    `json:"ticket_id" example:"1001"`
	TicketNumber    string                 `json:"ticket_number" example:"T-2024-001"`
	TicketTitle     string                 `json:"ticket_title" example:"系统响应缓慢"`
	AnalysisDate    string                 `json:"analysis_date" example:"2024-01-15"`
	RootCauses      []TicketRootCauseResponse    `json:"root_causes"`
	AnalysisSummary string                 `json:"analysis_summary" example:"系统自动分析识别出2个可能的根本原因"`
	ConfidenceScore float64                `json:"confidence_score" example:"0.85"`
	AnalysisMethod  string                 `json:"analysis_method" example:"automatic"`
	GeneratedAt     time.Time              `json:"generated_at" example:"2024-01-01T00:00:00Z"`
}

type TicketRootCauseResponse struct {
	ID            string                  `json:"id" example:"rc1"`
	Title         string                  `json:"title" example:"数据库连接池耗尽"`
	Description   string                  `json:"description" example:"系统检测到数据库连接池在高峰期耗尽"`
	Confidence    float64                 `json:"confidence" example:"0.92"`
	Category      string                  `json:"category" example:"database"`
	Evidence      []RootCauseEvidenceItem          `json:"evidence"`
	RelatedTickets []RootCauseRelatedTicketInfo    `json:"related_tickets"`
	ImpactScope   RootCauseImpactScopeInfo         `json:"impact_scope"`
	Recommendations []string              `json:"recommendations" example:"[\"增加数据库连接池大小\"]"`
	Status        string                  `json:"status" example:"identified"`
	CreatedAt     time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt     time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

type RootCauseEvidenceItem struct {
	Type      string    `json:"type" example:"log"`
	Content   string    `json:"content" example:"ERROR: Connection pool exhausted"`
	Timestamp string    `json:"timestamp" example:"2024-01-15 14:30:00"`
	Relevance float64   `json:"relevance" example:"0.95"`
}

type RootCauseRelatedTicketInfo struct {
	ID         int     `json:"id" example:"1001"`
	Number     string  `json:"number" example:"T-2024-001"`
	Title      string  `json:"title" example:"系统响应缓慢"`
	Similarity float64 `json:"similarity" example:"0.89"`
}

type RootCauseImpactScopeInfo struct {
	AffectedTickets int      `json:"affected_tickets" example:"15"`
	AffectedUsers   int      `json:"affected_users" example:"120"`
	AffectedSystems []string `json:"affected_systems" example:"[\"CRM系统\",\"订单系统\"]"`
}


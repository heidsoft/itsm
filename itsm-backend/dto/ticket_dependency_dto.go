package dto

// 依赖关系影响分析相关DTO
type RelationImpactAnalysisRequest struct {
	TicketID  int     `json:"ticket_id" binding:"required" example:"1001"`
	Action    string  `json:"action" binding:"required,oneof=close delete change_status" example:"close"`
	NewStatus *string `json:"new_status,omitempty" example:"closed"`
}

type RelationImpactAnalysis struct {
	TicketID         int                 `json:"ticket_id" example:"1001"`
	TicketNumber     string              `json:"ticket_number" example:"T-2024-001"`
	TicketTitle      string              `json:"ticket_title" example:"系统响应缓慢"`
	Action           string              `json:"action" example:"close"`
	AffectedCount    int                 `json:"affected_count" example:"3"`
	AffectedTickets  []AffectedTicketInfo `json:"affected_tickets"`
	Warnings         []string            `json:"warnings"`
	Recommendations  []string            `json:"recommendations"`
	RiskLevel        string              `json:"risk_level" example:"medium"`
}

type AffectedTicketInfo struct {
	ID          int    `json:"id" example:"1002"`
	Number      string `json:"number" example:"T-2024-002"`
	Title       string `json:"title" example:"数据库优化"`
	Status      string `json:"status" example:"in_progress"`
	ImpactType  string `json:"impact_type" example:"blocked"`
	Description string `json:"description" example:"父工单关闭可能导致此工单无法继续"`
}

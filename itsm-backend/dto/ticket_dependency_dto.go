package dto

// 依赖关系影响分析相关DTO
type RelationImpactAnalysisRequest struct {
	TicketID  int     `json:"ticketId" binding:"required" example:"1001"`
	Action    string  `json:"action" binding:"required,oneof=close delete change_status" example:"close"`
	NewStatus *string `json:"newStatus,omitempty" example:"closed"`
}

type RelationImpactAnalysis struct {
	TicketID        int                  `json:"ticketId" example:"1001"`
	TicketNumber    string               `json:"ticketNumber" example:"T-2024-001"`
	TicketTitle     string               `json:"ticketTitle" example:"系统响应缓慢"`
	Action          string               `json:"action" example:"close"`
	AffectedCount   int                  `json:"affectedCount" example:"3"`
	AffectedTickets []AffectedTicketInfo `json:"affectedTickets"`
	Warnings        []string             `json:"warnings"`
	Recommendations []string             `json:"recommendations"`
	RiskLevel       string               `json:"riskLevel" example:"medium"`
}

type AffectedTicketInfo struct {
	ID          int    `json:"id" example:"1002"`
	Number      string `json:"number" example:"T-2024-002"`
	Title       string `json:"title" example:"数据库优化"`
	Status      string `json:"status" example:"in_progress"`
	ImpactType  string `json:"impactType" example:"blocked"`
	Description string `json:"description" example:"父工单关闭可能导致此工单无法继续"`
}

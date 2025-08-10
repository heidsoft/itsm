package dto

import (
	"time"

	"itsm-backend/ent"
)

// ConfigurationItemInfo 配置项信息
type ConfigurationItemInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
}

// Incident 事件DTO
type Incident struct {
	ID                  int                    `json:"id"`
	IncidentNumber      string                 `json:"incident_number"`
	Title               string                 `json:"title"`
	Description         string                 `json:"description"`
	Status              string                 `json:"status"`
	Priority            string                 `json:"priority"`
	ReporterID          int                    `json:"reporter_id"`
	AssigneeID          *int                   `json:"assignee_id,omitempty"`
	ConfigurationItemID *int                   `json:"configuration_item_id,omitempty"`
	ConfigurationItem   *ConfigurationItemInfo `json:"configuration_item,omitempty"`
	TenantID            int                    `json:"tenant_id"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
	ResolvedAt          *time.Time             `json:"resolved_at,omitempty"`
	ClosedAt            *time.Time             `json:"closed_at,omitempty"`
}

// CreateIncidentRequest 创建事件请求
type CreateIncidentRequest struct {
	Title               string `json:"title" binding:"required,max=255"`
	Description         string `json:"description" binding:"required"`
	Priority            string `json:"priority" binding:"required,oneof=low medium high urgent"`
	AssigneeID          *int   `json:"assignee_id,omitempty"`
	ConfigurationItemID *int   `json:"configuration_item_id,omitempty"`
}

// UpdateIncidentRequest 更新事件请求
type UpdateIncidentRequest struct {
	Title               *string `json:"title,omitempty" binding:"omitempty,max=255"`
	Description         *string `json:"description,omitempty"`
	Status              *string `json:"status,omitempty" binding:"omitempty,oneof=new in_progress waiting_customer resolved closed"`
	Priority            *string `json:"priority,omitempty" binding:"omitempty,oneof=low medium high urgent"`
	AssigneeID          *int    `json:"assignee_id,omitempty"`
	ConfigurationItemID *int    `json:"configuration_item_id,omitempty"`
}

// GetIncidentsRequest 获取事件列表请求
type GetIncidentsRequest struct {
	Page                int    `form:"page" binding:"min=1"`
	Size                int    `form:"size" binding:"min=1,max=100"`
	Status              string `form:"status"`
	Priority            string `form:"priority"`
	ReporterID          int    `form:"reporter_id"`
	AssigneeID          int    `form:"assignee_id"`
	ConfigurationItemID int    `form:"configuration_item_id"`
	TenantID            int    `form:"tenant_id"`
}

// IncidentListResponse 事件列表响应
type IncidentListResponse struct {
	Incidents []Incident `json:"incidents"`
	Total     int        `json:"total"`
	Page      int        `json:"page"`
	Size      int        `json:"size"`
}

// IncidentStatsResponse 事件统计响应
type IncidentStatsResponse struct {
	TotalIncidents        int `json:"total_incidents"`
	OpenIncidents         int `json:"open_incidents"`
	ResolvedIncidents     int `json:"resolved_incidents"`
	ClosedIncidents       int `json:"closed_incidents"`
	UrgentIncidents       int `json:"urgent_incidents"`
	HighPriorityIncidents int `json:"high_priority_incidents"`
}

// ToIncidentResponse 转换为事件响应
func ToIncidentResponse(incident *ent.Incident) *Incident {
	resp := &Incident{
		ID:             incident.ID,
		IncidentNumber: incident.IncidentNumber,
		Title:          incident.Title,
		Description:    incident.Description,
		Status:         incident.Status,
		Priority:       incident.Priority,
		ReporterID:     incident.ReporterID,
		TenantID:       incident.TenantID,
		CreatedAt:      incident.CreatedAt,
		UpdatedAt:      incident.UpdatedAt,
	}

	// 处理可选字段 - 检查是否为0值
	if incident.AssigneeID > 0 {
		resp.AssigneeID = &incident.AssigneeID
	}
	if incident.ConfigurationItemID > 0 {
		resp.ConfigurationItemID = &incident.ConfigurationItemID
	}
	if !incident.ResolvedAt.IsZero() {
		resp.ResolvedAt = &incident.ResolvedAt
	}
	if !incident.ClosedAt.IsZero() {
		resp.ClosedAt = &incident.ClosedAt
	}

	return resp
}

// IncidentStatus 事件状态常量
const (
	IncidentStatusNew             = "new"
	IncidentStatusInProgress      = "in_progress"
	IncidentStatusWaitingCustomer = "waiting_customer"
	IncidentStatusResolved        = "resolved"
	IncidentStatusClosed          = "closed"
)

// IncidentPriority 事件优先级常量
const (
	IncidentPriorityLow    = "low"
	IncidentPriorityMedium = "medium"
	IncidentPriorityHigh   = "high"
	IncidentPriorityUrgent = "urgent"
)

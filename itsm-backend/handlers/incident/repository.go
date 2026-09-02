package incident

import (
	"context"
	"time"

	"itsm-backend/ent"
	"itsm-backend/handlers/common/datascope"
)

// Repository defines the interface for incident data access
type Repository interface {
	// Incident operations
	Create(ctx context.Context, incident *Incident) (*Incident, error)
	Get(ctx context.Context, id int, tenantID int) (*Incident, error)
	List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}, dataScope datascope.DataScope, currentUserID int) ([]*Incident, int, error)
	Update(ctx context.Context, incident *Incident) (*Incident, error)
	Delete(ctx context.Context, id int, tenantID int) error
	GenerateIncidentNumber(ctx context.Context, tenantID int, year int, month int) (string, error)
	CountByPeriod(ctx context.Context, tenantID int, start, end time.Time) (int, error)

	// Stats operations
	GetStats(ctx context.Context, tenantID int) (*IncidentStats, error)

	// Event operations
	CreateEvent(ctx context.Context, event *IncidentEvent) (*IncidentEvent, error)
	ListEvents(ctx context.Context, incidentID int, tenantID int) ([]*IncidentEvent, error)

	// Rule operations
	ListActiveRules(ctx context.Context, tenantID int) ([]*IncidentRule, error)
	UpdateRuleStats(ctx context.Context, ruleID int, count int, lastExecutedAt time.Time) error

	// Read-model queries for handler-facing sub-resources
	GetIncidentWithEdges(ctx context.Context, id, tenantID int, edges ...string) (*ent.Incident, error)
	ListIncidentComments(ctx context.Context, incidentID, tenantID int) ([]*ent.IncidentEvent, error)
	CreateIncidentCommentEvent(ctx context.Context, event *ent.IncidentEvent) (*ent.IncidentEvent, error)
	CountTenantSLAViolations(ctx context.Context, tenantID int) (int, error)
}

// IncidentStats 聚合统计（按 tenant 隔离）
// 字段名已统一为 camelCase JSON tag，对外暴露即前端期望字段。
type IncidentStats struct {
	TotalIncidents    int `json:"totalIncidents"`
	OpenIncidents     int `json:"openIncidents"`
	CriticalIncidents int `json:"criticalIncidents"`
	MajorIncidents    int `json:"majorIncidents"`
	ResolvedIncidents int `json:"resolvedIncidents"`
	// AvgResolutionTime 平均解决时长（分钟），仅统计已解决/已关闭事件。
	AvgResolutionTime int `json:"avgResolutionTime"`
}

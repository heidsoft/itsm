package incident

import (
	"context"
	"time"
)

// Repository defines the interface for incident data access
type Repository interface {
	// Incident operations
	Create(ctx context.Context, incident *Incident) (*Incident, error)
	Get(ctx context.Context, id int, tenantID int) (*Incident, error)
	List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*Incident, int, error)
	Update(ctx context.Context, incident *Incident) (*Incident, error)
	Delete(ctx context.Context, id int, tenantID int) error
	GenerateIncidentNumber(ctx context.Context, tenantID int, year int, month int) (string, error)
	CountByPeriod(ctx context.Context, tenantID int, start, end time.Time) (int, error)

	// Event operations
	CreateEvent(ctx context.Context, event *IncidentEvent) (*IncidentEvent, error)
	ListEvents(ctx context.Context, incidentID int, tenantID int) ([]*IncidentEvent, error)

	// Rule operations
	ListActiveRules(ctx context.Context, tenantID int) ([]*IncidentRule, error)
	UpdateRuleStats(ctx context.Context, ruleID int, count int, lastExecutedAt time.Time) error
}

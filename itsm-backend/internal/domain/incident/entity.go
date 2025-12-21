package incident

import (
	"time"
)

// Incident represents the core incident entity
type Incident struct {
	ID                  int
	Title               string
	Description         string
	Status              string // new, in_progress, resolved, closed
	Priority            string // low, medium, high, urgent
	Severity            string // low, medium, high, critical
	IncidentNumber      string
	ReporterID          int
	AssigneeID          *int
	ConfigurationItemID *int
	Category            string
	Subcategory         string
	ImpactAnalysis      map[string]interface{}
	RootCause           map[string]interface{}
	ResolutionSteps     []map[string]interface{}
	Metadata            map[string]interface{}
	DetectedAt          time.Time
	ResolvedAt          *time.Time
	ClosedAt            *time.Time
	EscalatedAt         *time.Time
	EscalationLevel     int
	IsAutomated         bool
	Source              string
	TenantID            int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// IncidentEvent represents an audit or activity log for an incident
type IncidentEvent struct {
	ID          int
	IncidentID  int
	EventType   string // e.g., creation, update, escalation, comment
	EventName   string
	Description string
	Status      string
	Severity    string
	Data        map[string]interface{}
	OccurredAt  time.Time
	UserID      int
	Source      string
	Metadata    map[string]interface{}
	TenantID    int
	CreatedAt   time.Time
}

// IncidentRule represents an automation rule
type IncidentRule struct {
	ID             int
	Name           string
	Description    string
	Conditions     map[string]interface{}
	Actions        []map[string]interface{}
	IsActive       bool
	Priority       string // Ent has this as string
	ExecutionCount int
	LastExecutedAt *time.Time
	TenantID       int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

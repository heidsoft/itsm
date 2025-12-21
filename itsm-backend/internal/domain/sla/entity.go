package sla

import (
	"time"
)

// SLADefinition represents an SLA policy
type SLADefinition struct {
	ID              int                    `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	ServiceType     string                 `json:"service_type"`
	Priority        string                 `json:"priority"`
	ResponseTime    int                    `json:"response_time"`   // in minutes
	ResolutionTime  int                    `json:"resolution_time"` // in minutes
	BusinessHours   map[string]interface{} `json:"business_hours"`
	EscalationRules map[string]interface{} `json:"escalation_rules"`
	Conditions      map[string]interface{} `json:"conditions"`
	IsActive        bool                   `json:"is_active"`
	TenantID        int                    `json:"tenant_id"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// SLAViolation represents a breach of SLA
type SLAViolation struct {
	ID              int        `json:"id"`
	TicketID        int        `json:"ticket_id"`
	SLADefinitionID int        `json:"sla_definition_id"`
	ViolationType   string     `json:"violation_type"` // e.g., "response", "resolution"
	ViolationTime   time.Time  `json:"violation_time"`
	Description     string     `json:"description"`
	Severity        string     `json:"severity"`
	IsResolved      bool       `json:"is_resolved"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	ResolutionNotes string     `json:"resolution_notes"`
	TenantID        int        `json:"tenant_id"`
}

// SLAMetric represents SLA performance indicators
type SLAMetric struct {
	ID              int       `json:"id"`
	SLADefinitionID int       `json:"sla_definition_id"`
	MetricType      string    `json:"metric_type"`
	MetricName      string    `json:"metric_name"`
	MetricValue     float64   `json:"metric_value"`
	Unit            string    `json:"unit"`
	MeasurementTime time.Time `json:"measurement_time"`
	TenantID        int       `json:"tenant_id"`
}

// SLAAlertRule defines when to trigger an alert before violation
type SLAAlertRule struct {
	ID                   int      `json:"id"`
	SLADefinitionID      int      `json:"sla_definition_id"`
	Name                 string   `json:"name"`
	ThresholdPercentage  int      `json:"threshold_percentage"` // 0-100
	AlertLevel           string   `json:"alert_level"`
	NotificationChannels []string `json:"notification_channels"`
	IsActive             bool     `json:"is_active"`
	TenantID             int      `json:"tenant_id"`
}

// SLAAlertHistory records triggered alerts
type SLAAlertHistory struct {
	ID                  int        `json:"id"`
	AlertRuleID         int        `json:"alert_rule_id"`
	TicketID            int        `json:"ticket_id"`
	TicketNumber        string     `json:"ticket_number"`
	TicketTitle         string     `json:"ticket_title"`
	AlertLevel          string     `json:"alert_level"`
	ThresholdPercentage int        `json:"threshold_percentage"`
	ActualPercentage    float64    `json:"actual_percentage"`
	NotificationSent    bool       `json:"notification_sent"`
	CreatedAt           time.Time  `json:"created_at"` // trigger time
	ResolvedAt          *time.Time `json:"resolved_at,omitempty"`
	TenantID            int        `json:"tenant_id"`
}

package sla

import (
	"time"
)

// SLADefinition represents an SLA policy
type SLADefinition struct {
	ID              int                    `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	ServiceType     string                 `json:"serviceType"`
	Priority        string                 `json:"priority"`
	ResponseTime    int                    `json:"responseTime"`   // in minutes
	ResolutionTime  int                    `json:"resolutionTime"` // in minutes
	BusinessHours   map[string]interface{} `json:"businessHours"`
	EscalationRules map[string]interface{} `json:"escalationRules"`
	Conditions      map[string]interface{} `json:"conditions"`
	IsActive        bool                   `json:"isActive"`
	TenantID        int                    `json:"tenantId"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

// SLAViolation represents a breach of SLA
type SLAViolation struct {
	ID              int        `json:"id"`
	CreatedBy       int        `json:"createdBy"`
	TicketID        int        `json:"ticketId"`
	SLADefinitionID int        `json:"slaDefinitionId"`
	ViolationType   string     `json:"violationType"` // e.g., "response", "resolution"
	ViolationTime   time.Time  `json:"violationTime"`
	Description     string     `json:"description"`
	Severity        string     `json:"severity"`
	IsResolved      bool       `json:"isResolved"`
	ResolvedAt      *time.Time `json:"resolvedAt,omitempty"`
	ResolutionNotes string     `json:"resolutionNotes"`
	TenantID        int        `json:"tenantId"`
}

// SLAMetric represents SLA performance indicators
type SLAMetric struct {
	ID              int       `json:"id"`
	SLADefinitionID int       `json:"slaDefinitionId"`
	MetricType      string    `json:"metricType"`
	MetricName      string    `json:"metricName"`
	MetricValue     float64   `json:"metricValue"`
	Unit            string    `json:"unit"`
	MeasurementTime time.Time `json:"measurementTime"`
	TenantID        int       `json:"tenantId"`
}

// SLAAlertRule defines when to trigger an alert before violation
type SLAAlertRule struct {
	ID                   int      `json:"id"`
	SLADefinitionID      int      `json:"slaDefinitionId"`
	Name                 string   `json:"name"`
	ThresholdPercentage  int      `json:"thresholdPercentage"` // 0-100
	AlertLevel           string   `json:"alertLevel"`
	NotificationChannels []string `json:"notificationChannels"`
	IsActive             bool     `json:"isActive"`
	TenantID             int      `json:"tenantId"`
}

// SLAAlertHistory records triggered alerts
type SLAAlertHistory struct {
	ID                       int        `json:"id"`
	AlertRuleID              int        `json:"alertRuleId"`
	TicketID                 int        `json:"ticketId"`
	TicketNumber             string     `json:"ticketNumber"`
	TicketTitle              string     `json:"ticketTitle"`
	AlertLevel               string     `json:"alertLevel"`
	ThresholdPercentage      int        `json:"thresholdPercentage"`
	ActualPercentage         float64    `json:"actualPercentage"`
	NotificationSent         bool       `json:"notificationSent"`
	CreatedAt                time.Time  `json:"createdAt"` // trigger time
	ResolvedAt               *time.Time `json:"resolvedAt,omitempty"`
	TenantID                 int        `json:"tenantId"`
	CooldownRemainingSeconds int        `json:"cooldownRemainingSeconds"`
	CooldownMinutes          int        `json:"cooldownMinutes"`
	SuppressedByCooldown     bool       `json:"suppressedByCooldown"`
}

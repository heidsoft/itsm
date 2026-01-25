package sla

import (
	"context"
)

// Repository interface for SLA domain
type Repository interface {
	// SLA Definitions
	CreateDefinition(ctx context.Context, s *SLADefinition) (*SLADefinition, error)
	GetDefinition(ctx context.Context, id int, tenantID int) (*SLADefinition, error)
	ListDefinitions(ctx context.Context, tenantID int, page, size int) ([]*SLADefinition, int, error)
	UpdateDefinition(ctx context.Context, s *SLADefinition) (*SLADefinition, error)
	DeleteDefinition(ctx context.Context, id int, tenantID int) error

	// Violations
	CreateViolation(ctx context.Context, v *SLAViolation) (*SLAViolation, error)
	ListViolations(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*SLAViolation, int, error)
	UpdateViolationStatus(ctx context.Context, id int, isResolved bool, notes string, tenantID int) error

	// Metrics
	CreateMetric(ctx context.Context, m *SLAMetric) (*SLAMetric, error)
	GetMetrics(ctx context.Context, tenantID int, filters map[string]interface{}) ([]*SLAMetric, error)
	GetSLAMonitoring(ctx context.Context, tenantID int, startTime, endTime string) (map[string]interface{}, error)

	// Alert Rules
	CreateAlertRule(ctx context.Context, r *SLAAlertRule) (*SLAAlertRule, error)
	GetAlertRule(ctx context.Context, id int, tenantID int) (*SLAAlertRule, error)
	ListAlertRules(ctx context.Context, tenantID int, filters map[string]interface{}) ([]*SLAAlertRule, error)
	UpdateAlertRule(ctx context.Context, r *SLAAlertRule) (*SLAAlertRule, error)
	DeleteAlertRule(ctx context.Context, id int, tenantID int) error

	// Alert History
	CreateAlertHistory(ctx context.Context, h *SLAAlertHistory) (*SLAAlertHistory, error)
	ListAlertHistory(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*SLAAlertHistory, int, error)
}

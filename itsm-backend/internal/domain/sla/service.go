package sla

import (
	"context"

	"go.uber.org/zap"
)

type Service struct {
	repo   Repository
	logger *zap.SugaredLogger
}

func NewService(repo Repository, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// SLADefinition CRUD

func (s *Service) CreateDefinition(ctx context.Context, def *SLADefinition) (*SLADefinition, error) {
	s.logger.Infow("Creating SLA Definition", "name", def.Name)
	return s.repo.CreateDefinition(ctx, def)
}

func (s *Service) GetDefinition(ctx context.Context, id int, tenantID int) (*SLADefinition, error) {
	return s.repo.GetDefinition(ctx, id, tenantID)
}

func (s *Service) ListDefinitions(ctx context.Context, tenantID int, page, size int) ([]*SLADefinition, int, error) {
	return s.repo.ListDefinitions(ctx, tenantID, page, size)
}

func (s *Service) UpdateDefinition(ctx context.Context, def *SLADefinition) (*SLADefinition, error) {
	s.logger.Infow("Updating SLA Definition", "id", def.ID)
	return s.repo.UpdateDefinition(ctx, def)
}

func (s *Service) DeleteDefinition(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting SLA Definition", "id", id)
	return s.repo.DeleteDefinition(ctx, id, tenantID)
}

// SLA Compliance and Monitoring

func (s *Service) CheckSLACompliance(ctx context.Context, ticketID int, tenantID int) error {
	s.logger.Infow("Checking SLA Compliance", "ticketID", ticketID)
	// Logic to find applicable SLA, check timestamps, and record violations
	// For now, this is a placeholder to match existing controller logic
	return nil
}

func (s *Service) GetSLAMonitoring(ctx context.Context, tenantID int, startTime, endTime string) (interface{}, error) {
	// Logic to aggregate metrics
	return map[string]interface{}{
		"compliance_rate":  0.95,
		"violations_count": 5,
	}, nil
}

// Alert Rules

func (s *Service) CreateAlertRule(ctx context.Context, rule *SLAAlertRule) (*SLAAlertRule, error) {
	return s.repo.CreateAlertRule(ctx, rule)
}

func (s *Service) GetAlertRule(ctx context.Context, id int, tenantID int) (*SLAAlertRule, error) {
	return s.repo.GetAlertRule(ctx, id, tenantID)
}

func (s *Service) ListAlertRules(ctx context.Context, tenantID int, filters map[string]interface{}) ([]*SLAAlertRule, error) {
	return s.repo.ListAlertRules(ctx, tenantID, filters)
}

func (s *Service) UpdateAlertRule(ctx context.Context, rule *SLAAlertRule) (*SLAAlertRule, error) {
	return s.repo.UpdateAlertRule(ctx, rule)
}

func (s *Service) DeleteAlertRule(ctx context.Context, id int, tenantID int) error {
	return s.repo.DeleteAlertRule(ctx, id, tenantID)
}

func (s *Service) GetAlertHistory(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*SLAAlertHistory, int, error) {
	return s.repo.ListAlertHistory(ctx, tenantID, page, size, filters)
}

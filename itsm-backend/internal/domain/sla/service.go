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
	return nil
}

// Violations

func (s *Service) GetSLAViolations(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*SLAViolation, int, error) {
	return s.repo.ListViolations(ctx, tenantID, page, size, filters)
}

func (s *Service) UpdateSLAViolationStatus(ctx context.Context, id int, isResolved bool, notes string, tenantID int) (*SLAViolation, error) {
	s.logger.Infow("Updating SLA Violation status", "id", id, "isResolved", isResolved)
	err := s.repo.UpdateViolationStatus(ctx, id, isResolved, notes, tenantID)
	if err != nil {
		return nil, err
	}
	// Return updated violation - fetch from repo
	violations, _, err := s.repo.ListViolations(ctx, tenantID, 1, 1, map[string]interface{}{"id": id})
	if err != nil || len(violations) == 0 {
		return nil, err
	}
	return violations[0], nil
}

// Metrics

func (s *Service) GetSLAMetrics(ctx context.Context, tenantID int, filters map[string]interface{}) ([]*SLAMetric, error) {
	return s.repo.GetMetrics(ctx, tenantID, filters)
}

func (s *Service) GetSLAMonitoring(ctx context.Context, tenantID int, startTime, endTime string) (map[string]interface{}, error) {
	return s.repo.GetSLAMonitoring(ctx, tenantID, startTime, endTime)
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

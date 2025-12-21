package incident

import (
	"context"
	"fmt"
	"time"

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

func (s *Service) Create(ctx context.Context, tenantID int, i *Incident) (*Incident, error) {
	s.logger.Infow("Creating incident", "title", i.Title, "tenant_id", tenantID)

	// Generate number
	year, month, _ := time.Now().Date()
	number, err := s.repo.GenerateIncidentNumber(ctx, tenantID, year, int(month))
	if err != nil {
		return nil, fmt.Errorf("failed to generate incident number: %w", err)
	}
	i.IncidentNumber = number
	i.TenantID = tenantID
	i.Status = "new" // default
	if i.DetectedAt.IsZero() {
		i.DetectedAt = time.Now()
	}

	created, err := s.repo.Create(ctx, i)
	if err != nil {
		return nil, err
	}

	// Audit Log
	s.repo.CreateEvent(ctx, &IncidentEvent{
		IncidentID:  created.ID,
		EventType:   "creation",
		EventName:   "事件创建",
		Description: fmt.Sprintf("事件 %s 已创建", number),
		Status:      "active",
		Severity:    "info",
		Source:      "system",
		UserID:      i.ReporterID,
		OccurredAt:  time.Now(),
		TenantID:    tenantID,
	})

	// Execute Rules Async
	go s.executeRules(context.Background(), created, tenantID) // Use background context for async

	return created, nil
}

func (s *Service) Get(ctx context.Context, id int, tenantID int) (*Incident, error) {
	return s.repo.Get(ctx, id, tenantID)
}

func (s *Service) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*Incident, int, error) {
	return s.repo.List(ctx, tenantID, page, size, filters)
}

func (s *Service) Update(ctx context.Context, tenantID int, id int, updates *Incident) (*Incident, error) {
	current, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	// Note: This logic depends on what 'updates' contains.
	// In a real separate domain, we might pass specific fields or a map.
	// For simplicity, we assume 'updates' has non-zero values for fields to change.

	if updates.Title != "" {
		current.Title = updates.Title
	}
	if updates.Description != "" {
		current.Description = updates.Description
	}
	if updates.Status != "" {
		current.Status = updates.Status
		if updates.Status == "resolved" {
			now := time.Now()
			current.ResolvedAt = &now
		}
		if updates.Status == "closed" {
			now := time.Now()
			current.ClosedAt = &now
		}
	}
	if updates.Priority != "" {
		current.Priority = updates.Priority
	}
	if updates.Severity != "" {
		current.Severity = updates.Severity
	}
	if updates.AssigneeID != nil {
		current.AssigneeID = updates.AssigneeID
	}
	// ... other fields

	updated, err := s.repo.Update(ctx, current)
	if err != nil {
		return nil, err
	}

	s.repo.CreateEvent(ctx, &IncidentEvent{
		IncidentID:  id,
		EventType:   "update",
		EventName:   "事件更新",
		Description: "事件信息已更新",
		OccurredAt:  time.Now(),
		TenantID:    tenantID,
		Source:      "system",
	})

	return updated, nil
}

func (s *Service) Escalate(ctx context.Context, tenantID int, id int, level int, reason string) (*Incident, error) {
	current, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	current.EscalationLevel = level
	now := time.Now()
	current.EscalatedAt = &now

	updated, err := s.repo.Update(ctx, current)
	if err != nil {
		return nil, err
	}

	s.repo.CreateEvent(ctx, &IncidentEvent{
		IncidentID:  id,
		EventType:   "escalation",
		EventName:   "事件升级",
		Description: fmt.Sprintf("事件升级到级别 %d: %s", level, reason),
		Data: map[string]interface{}{
			"level":  level,
			"reason": reason,
		},
		OccurredAt: time.Now(),
		TenantID:   tenantID,
		Source:     "system",
	})

	return updated, nil
}

// executeRules Logic
func (s *Service) executeRules(ctx context.Context, incident *Incident, tenantID int) {
	rules, err := s.repo.ListActiveRules(ctx, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to list active rules", "error", err)
		return
	}

	for _, rule := range rules {
		if s.evaluateCondition(rule.Conditions, incident) {
			s.executeAction(ctx, rule, incident, tenantID)
		}
	}
}

func (s *Service) evaluateCondition(conditions map[string]interface{}, incident *Incident) bool {
	// Simplified evaluation logic
	if priority, ok := conditions["priority"].([]string); ok {
		match := false
		for _, p := range priority {
			if incident.Priority == p {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	if status, ok := conditions["status"].(string); ok {
		if incident.Status != status {
			return false
		}
	}
	// Add more conditions as needed
	return true
}

func (s *Service) executeAction(ctx context.Context, rule *IncidentRule, incident *Incident, tenantID int) {
	// Execute implementation
	// ... (Simplification: updating stats and logging for now to avoid circular service dependencies if action updates incident again)
	s.logger.Infow("Rule Executed", "rule_id", rule.ID, "incident_id", incident.ID)

	// Update stats
	s.repo.UpdateRuleStats(ctx, rule.ID, rule.ExecutionCount+1, time.Now())
}

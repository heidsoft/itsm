package problem

import (
	"context"
	"fmt"
	"strings"
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

func (s *Service) Create(ctx context.Context, tenantID int, p *Problem) (*Problem, error) {
	if strings.TrimSpace(p.Title) == "" {
		return nil, fmt.Errorf("problem title is required")
	}
	if !isValidProblemPriority(p.Priority) {
		return nil, fmt.Errorf("invalid problem priority: %s", p.Priority)
	}
	p.Title = strings.TrimSpace(p.Title)
	p.Status = "open"
	p.TenantID = tenantID
	return s.repo.Create(ctx, p)
}

func (s *Service) Get(ctx context.Context, id int, tenantID int) (*Problem, error) {
	return s.repo.Get(ctx, id, tenantID)
}

func (s *Service) GetWithAssociations(ctx context.Context, id int, tenantID int) (*Problem, error) {
	return s.repo.GetWithAssociations(ctx, id, tenantID)
}

func (s *Service) AddAssociations(ctx context.Context, tenantID, problemID int, relatedType string, relatedIDs []int) error {
	relatedIDs = uniquePositiveIDs(relatedIDs)
	if len(relatedIDs) == 0 {
		return fmt.Errorf("at least one related id is required")
	}
	return s.repo.AddAssociations(ctx, tenantID, problemID, relatedType, relatedIDs)
}

func (s *Service) RemoveAssociation(ctx context.Context, tenantID, problemID int, relatedType string, relatedID int) error {
	if relatedID <= 0 {
		return fmt.Errorf("invalid related id")
	}
	return s.repo.RemoveAssociation(ctx, tenantID, problemID, relatedType, relatedID)
}

func (s *Service) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*Problem, int, error) {
	return s.repo.List(ctx, tenantID, page, size, filters)
}

func (s *Service) Update(ctx context.Context, tenantID int, id int, p *Problem) (*Problem, error) {
	existing, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Update fields if they are set (non-zero/non-empty check in Handler or here)
	// Here assuming 'p' contains only fields to update usually, but domain entity isn't partial.
	// We merge changes here.
	if p.Title != "" {
		existing.Title = p.Title
	}
	if p.Description != "" {
		existing.Description = p.Description
	}
	if p.Status != "" {
		if !isValidProblemStatusTransition(existing.Status, p.Status) {
			return nil, fmt.Errorf("invalid problem status transition: %s -> %s", existing.Status, p.Status)
		}
		existing.Status = p.Status
		now := time.Now()
		switch p.Status {
		case "resolved":
			existing.ResolvedAt = &now
			existing.ClosedAt = nil
		case "closed":
			existing.ClosedAt = &now
		case "investigating":
			existing.ResolvedAt = nil
			existing.ClosedAt = nil
		}
	}
	if p.Priority != "" {
		if !isValidProblemPriority(p.Priority) {
			return nil, fmt.Errorf("invalid problem priority: %s", p.Priority)
		}
		existing.Priority = p.Priority
	}
	if p.Category != "" {
		existing.Category = p.Category
	}
	if p.RootCause != "" {
		existing.RootCause = p.RootCause
	}
	if p.Impact != "" {
		existing.Impact = p.Impact
	}
	if p.AssigneeID != nil {
		existing.AssigneeID = p.AssigneeID
	}

	return s.repo.Update(ctx, existing)
}

func isValidProblemPriority(priority string) bool {
	switch priority {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func isValidProblemStatusTransition(current, next string) bool {
	if current == next {
		return true
	}
	transitions := map[string]map[string]struct{}{
		"open":          {"investigating": {}, "identified": {}, "resolved": {}, "closed": {}},
		"investigating": {"identified": {}, "resolved": {}, "closed": {}},
		"identified":    {"investigating": {}, "resolved": {}, "closed": {}},
		"resolved":      {"investigating": {}, "closed": {}},
		"closed":        {},
		// 兼容存量 in_progress 数据，仅允许进入规范状态。
		"in_progress": {"identified": {}, "resolved": {}, "closed": {}},
	}
	allowed, ok := transitions[current]
	if !ok {
		return false
	}
	_, ok = allowed[next]
	return ok
}

func uniquePositiveIDs(ids []int) []int {
	seen := make(map[int]struct{}, len(ids))
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func (s *Service) Delete(ctx context.Context, id int, tenantID int) error {
	return s.repo.Delete(ctx, id, tenantID)
}

func (s *Service) GetStats(ctx context.Context, tenantID int) (*ProblemStats, error) {
	return s.repo.GetStats(ctx, tenantID)
}

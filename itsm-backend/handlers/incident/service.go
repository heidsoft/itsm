package incident

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/common/tenantctx"

	"go.uber.org/zap"
)

// ProcessTriggerServiceInterface 流程触发服务接口
type ProcessTriggerServiceInterface interface {
	TriggerProcess(ctx context.Context, req interface{}) (interface{}, error)
}

type Service struct {
	repo                  Repository
	logger                *zap.SugaredLogger
	processTriggerService ProcessTriggerServiceInterface
}

func NewService(repo Repository, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// SetProcessTriggerService 设置流程触发服务
func (s *Service) SetProcessTriggerService(triggerService ProcessTriggerServiceInterface) {
	s.processTriggerService = triggerService
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
	if strings.TrimSpace(i.Priority) == "" {
		i.Priority = inferIncidentPriority(i.Title, i.Description)
	}
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

	// Execute Rules Async（派生带租户上下文的独立 context，避免 RLS enforce 后异步规则失效）
	go s.executeRules(tenantctx.WithTenantID(context.Background(), tenantID), created, tenantID)

	return created, nil
}

// inferIncidentPriority provides a deterministic fallback when an operator or
// integration does not specify priority. Explicit priorities are never
// overwritten; automation rules can still refine the result after creation.
func inferIncidentPriority(title, description string) string {
	content := strings.ToLower(title + " " + description)
	for _, keyword := range []string{"down", "outage", "critical", "production"} {
		if strings.Contains(content, keyword) {
			return "urgent"
		}
	}
	for _, keyword := range []string{"slow", "error", "issue"} {
		if strings.Contains(content, keyword) {
			return "medium"
		}
	}
	return "low"
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
		// 阻断6 修复：必须走事件状态机白名单校验，禁止裸写 SetStatus。
		// 旧逻辑允许终态（closed/cancelled）事件被任意改写回 active，
		// 也允许从 new 直接跳到 resolved 绕过处理流程。
		if !common.IsValidIncidentStatusTransition(current.Status, updates.Status) {
			return nil, fmt.Errorf("invalid incident status transition from '%s' to '%s'", current.Status, updates.Status)
		}
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

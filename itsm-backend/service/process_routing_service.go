package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"itsm-backend/ent"
	"itsm-backend/ent/processbinding"

	"go.uber.org/zap"
)

// ProcessRoutingService provides multi-dimensional process routing
type ProcessRoutingService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewProcessRoutingService creates a new ProcessRoutingService
func NewProcessRoutingService(client *ent.Client, logger *zap.SugaredLogger) *ProcessRoutingService {
	return &ProcessRoutingService{
		client: client,
		logger: logger,
	}
}

// RoutingContext contains the context for process routing decisions
type RoutingContext struct {
	BusinessType    string                 `json:"business_type"`
	BusinessSubType string                 `json:"business_sub_type,omitempty"`
	TenantID        int                    `json:"tenant_id"`
	DepartmentID    int                    `json:"department_id,omitempty"`
	TeamID          int                    `json:"team_id,omitempty"`
	ProjectID       int                    `json:"project_id,omitempty"`
	Scenario        string                 `json:"scenario,omitempty"`
	Category        string                 `json:"category,omitempty"`
	Variables       map[string]interface{} `json:"variables,omitempty"`
}

// RoutingResult contains the result of a routing decision
type RoutingResult struct {
	ProcessDefinitionKey string                 `json:"process_definition_key"`
	ApprovalChainID      string                 `json:"approval_chain_id,omitempty"`
	SLAPolicyID          string                 `json:"sla_policy_id,omitempty"`
	Overrides            map[string]interface{} `json:"overrides,omitempty"`
	MatchedRuleID        int                    `json:"matched_rule_id"`
	MatchedRuleName      string                 `json:"matched_rule_name,omitempty"`
}

// FindBestRoute finds the best matching process binding using priority-based scoring
func (s *ProcessRoutingService) FindBestRoute(ctx context.Context, reqCtx *RoutingContext) (*RoutingResult, error) {
	s.logger.Infow(
		"Finding best route",
		"business_type", reqCtx.BusinessType,
		"department_id", reqCtx.DepartmentID,
		"team_id", reqCtx.TeamID,
		"scenario", reqCtx.Scenario,
	)

	// Query all active bindings for this tenant and business type
	query := s.client.ProcessBinding.Query().
		Where(
			processbinding.TenantID(reqCtx.TenantID),
			processbinding.IsActive(true),
		)

	// Filter by business type if specified
	if reqCtx.BusinessType != "" {
		query = query.Where(processbinding.BusinessType(reqCtx.BusinessType))
	}

	bindings, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query process bindings: %w", err)
	}

	if len(bindings) == 0 {
		s.logger.Warnw("No active process bindings found", "tenant_id", reqCtx.TenantID)
		return nil, nil
	}

	// Score and rank bindings
	type scoredBinding struct {
		binding *ent.ProcessBinding
		score   int
	}

	scored := make([]scoredBinding, 0, len(bindings))
	for _, b := range bindings {
		score := s.calculateMatchScore(b, reqCtx)
		if score > 0 {
			scored = append(scored, scoredBinding{binding: b, score: score})
		}
	}

	if len(scored) == 0 {
		s.logger.Warnw("No matching process bindings found")
		return nil, nil
	}

	// Sort by score (desc), then priority (desc)
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].binding.Priority > scored[j].binding.Priority
	})

	best := scored[0].binding
	s.logger.Infow(
		"Found best route",
		"binding_id", best.ID,
		"process_key", best.ProcessDefinitionKey,
		"score", scored[0].score,
	)

	return &RoutingResult{
		ProcessDefinitionKey: best.ProcessDefinitionKey,
		ApprovalChainID:      best.ApprovalChainID,
		SLAPolicyID:          best.SLAPolicyID,
		Overrides:            best.Overrides,
		MatchedRuleID:        best.ID,
	}, nil
}

// calculateMatchScore calculates match score for a binding
// Higher score = more specific match
func (s *ProcessRoutingService) calculateMatchScore(binding *ent.ProcessBinding, reqCtx *RoutingContext) int {
	score := 0

	// Business sub-type match (required if specified in binding)
	if binding.BusinessSubType != "" {
		if binding.BusinessSubType != reqCtx.BusinessSubType {
			return 0 // No match
		}
		score += 100
	}

	// Department match (0 = global/wildcard)
	if binding.DepartmentID > 0 {
		if binding.DepartmentID == reqCtx.DepartmentID {
			score += 200 // Exact department match
		} else {
			// Department doesn't match - skip this binding
			return 0
		}
	} else {
		score += 10 // Global binding (lowest priority)
	}

	// Team match (0 = global/wildcard)
	if binding.TeamID > 0 {
		if binding.TeamID == reqCtx.TeamID {
			score += 150 // Exact team match
		} else {
			return 0
		}
	} else if reqCtx.TeamID > 0 {
		// Request has team but binding is global - still valid but lower priority
		score += 5
	}

	// Scenario match
	if binding.Scenario != "" {
		if binding.Scenario == reqCtx.Scenario {
			score += 300 // Highest priority dimension
		} else {
			return 0
		}
	}

	// Category match
	if binding.Category != "" {
		if binding.Category == reqCtx.Category {
			score += 50
		} else {
			return 0
		}
	}

	// Conditions evaluation
	if binding.Conditions != nil && len(binding.Conditions) > 0 {
		if s.evaluateConditions(binding.Conditions, reqCtx.Variables) {
			score += 250
		} else {
			return 0
		}
	}

	return score
}

// evaluateConditions evaluates condition expressions against variables
func (s *ProcessRoutingService) evaluateConditions(conditions map[string]interface{}, variables map[string]interface{}) bool {
	if variables == nil {
		return len(conditions) == 0
	}

	for key, expected := range conditions {
		if key == "min_amount" || strings.HasPrefix(key, "min_") {
			field := strings.TrimPrefix(key, "min_")
			if !s.compareNumericCondition(variables[field], expected, func(actual, threshold float64) bool { return actual >= threshold }) {
				return false
			}
			continue
		}
		if key == "max_amount" || strings.HasPrefix(key, "max_") {
			field := strings.TrimPrefix(key, "max_")
			if !s.compareNumericCondition(variables[field], expected, func(actual, threshold float64) bool { return actual <= threshold }) {
				return false
			}
			continue
		}

		actual, exists := variables[key]
		if !exists {
			return false
		}
		// Simple equality check - can be extended for complex expressions
		if fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected) {
			return false
		}
	}
	return true
}

func (s *ProcessRoutingService) compareNumericCondition(actualValue, expectedValue interface{}, compare func(actual, threshold float64) bool) bool {
	actual, ok := routingValueToFloat64(actualValue)
	if !ok {
		return false
	}
	threshold, ok := routingValueToFloat64(expectedValue)
	if !ok {
		return false
	}
	return compare(actual, threshold)
}

func routingValueToFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

// CreateBinding creates a new process binding
func (s *ProcessRoutingService) CreateBinding(ctx context.Context, binding *ent.ProcessBinding) (*ent.ProcessBinding, error) {
	created, err := s.client.ProcessBinding.Create().
		SetBusinessType(binding.BusinessType).
		SetNillableBusinessSubType(&binding.BusinessSubType).
		SetProcessDefinitionKey(binding.ProcessDefinitionKey).
		SetProcessVersion(binding.ProcessVersion).
		SetIsDefault(binding.IsDefault).
		SetPriority(binding.Priority).
		SetIsActive(binding.IsActive).
		SetDepartmentID(binding.DepartmentID).
		SetTeamID(binding.TeamID).
		SetNillableScenario(&binding.Scenario).
		SetNillableCategory(&binding.Category).
		SetConditions(binding.Conditions).
		SetNillableApprovalChainID(&binding.ApprovalChainID).
		SetNillableSLAPolicyID(&binding.SLAPolicyID).
		SetOverrides(binding.Overrides).
		SetTenantID(binding.TenantID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create process binding: %w", err)
	}
	return created, nil
}

// UpdateBinding updates an existing process binding
func (s *ProcessRoutingService) UpdateBinding(ctx context.Context, id int, updates map[string]interface{}) (*ent.ProcessBinding, error) {
	update := s.client.ProcessBinding.UpdateOneID(id)

	if v, ok := updates["department_id"].(int); ok {
		update = update.SetDepartmentID(v)
	}
	if v, ok := updates["team_id"].(int); ok {
		update = update.SetTeamID(v)
	}
	if v, ok := updates["scenario"].(string); ok {
		update = update.SetScenario(v)
	}
	if v, ok := updates["category"].(string); ok {
		update = update.SetCategory(v)
	}
	if v, ok := updates["conditions"].(map[string]interface{}); ok {
		update = update.SetConditions(v)
	}
	if v, ok := updates["priority"].(int); ok {
		update = update.SetPriority(v)
	}
	if v, ok := updates["is_active"].(bool); ok {
		update = update.SetIsActive(v)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update process binding: %w", err)
	}
	return updated, nil
}

// ListBindings lists all process bindings for a tenant
func (s *ProcessRoutingService) ListBindings(ctx context.Context, tenantID int, filters map[string]interface{}) ([]*ent.ProcessBinding, error) {
	query := s.client.ProcessBinding.Query().
		Where(processbinding.TenantID(tenantID))

	if deptID, ok := filters["department_id"].(int); ok && deptID > 0 {
		query = query.Where(processbinding.DepartmentID(deptID))
	}
	if scenario, ok := filters["scenario"].(string); ok && scenario != "" {
		query = query.Where(processbinding.Scenario(scenario))
	}
	if businessType, ok := filters["business_type"].(string); ok && businessType != "" {
		query = query.Where(processbinding.BusinessType(businessType))
	}

	return query.Order(ent.Desc(processbinding.FieldPriority)).All(ctx)
}

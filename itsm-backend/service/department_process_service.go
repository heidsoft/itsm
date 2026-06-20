package service

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/department"
	"itsm-backend/service/scenario"

	"go.uber.org/zap"
)

// DepartmentProcessService manages department-specific process routing
type DepartmentProcessService struct {
	client         *ent.Client
	logger         *zap.SugaredLogger
	routingService *ProcessRoutingService
}

// NewDepartmentProcessService creates a new DepartmentProcessService
func NewDepartmentProcessService(client *ent.Client, logger *zap.SugaredLogger) *DepartmentProcessService {
	return &DepartmentProcessService{
		client:         client,
		logger:         logger,
		routingService: NewProcessRoutingService(client, logger),
	}
}

// GetDepartmentProcess returns the appropriate process for a department and scenario
func (s *DepartmentProcessService) GetDepartmentProcess(
	ctx context.Context,
	tenantID, departmentID int,
	scenarioType scenario.ScenarioType,
	variables map[string]interface{},
) (*RoutingResult, error) {
	s.logger.Infow("Getting department process",
		"tenant_id", tenantID,
		"department_id", departmentID,
		"scenario", scenarioType,
	)

	// Build routing context
	routingCtx := &RoutingContext{
		TenantID:     tenantID,
		DepartmentID: departmentID,
		Scenario:     string(scenarioType),
		Variables:    variables,
	}

	// Map scenario to business type
	s.mapScenarioToBusinessType(scenarioType, routingCtx, variables)

	// Find best route
	result, err := s.routingService.FindBestRoute(ctx, routingCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to find route: %w", err)
	}

	if result == nil {
		s.logger.Warnw("No process found for department scenario",
			"department_id", departmentID,
			"scenario", scenarioType,
		)
		return nil, nil
	}

	s.logger.Infow("Found department process",
		"process_key", result.ProcessDefinitionKey,
		"matched_rule_id", result.MatchedRuleID,
	)

	return result, nil
}

// mapScenarioToBusinessType maps scenario to business type and sub-type
func (s *DepartmentProcessService) mapScenarioToBusinessType(
	scenarioType scenario.ScenarioType,
	ctx *RoutingContext,
	variables map[string]interface{},
) {
	switch scenarioType {
	// Operations scenarios
	case scenario.ScenarioAlertHandling:
		ctx.BusinessType = "incident"
		if severity, ok := variables["severity"].(string); ok {
			ctx.BusinessSubType = fmt.Sprintf("alert_%s", severity)
		}

	case scenario.ScenarioChangeRelease:
		ctx.BusinessType = "change"
		ctx.BusinessSubType = "normal"

	case scenario.ScenarioEmergencyChange:
		ctx.BusinessType = "change"
		ctx.BusinessSubType = "emergency"

	case scenario.ScenarioStandardChange:
		ctx.BusinessType = "change"
		ctx.BusinessSubType = "standard"

	// R&D scenarios
	case scenario.ScenarioCodeReleaseProd:
		ctx.BusinessType = "release"
		ctx.BusinessSubType = "production"

	case scenario.ScenarioCodeReleaseTest:
		ctx.BusinessType = "release"
		ctx.BusinessSubType = "testing"

	case scenario.ScenarioRequirementChange:
		ctx.BusinessType = "change"
		ctx.BusinessSubType = "requirement"

	// Finance scenarios
	case scenario.ScenarioExpenseApproval:
		ctx.BusinessType = "service_request"
		ctx.Category = "finance"

	case scenario.ScenarioBudgetApproval:
		ctx.BusinessType = "service_request"
		ctx.Category = "finance"

	case scenario.ScenarioProcurement:
		ctx.BusinessType = "service_request"
		ctx.Category = "procurement"

	// HR scenarios
	case scenario.ScenarioLeaveApproval:
		ctx.BusinessType = "service_request"
		ctx.Category = "hr"

	case scenario.ScenarioRecruitmentApproval:
		ctx.BusinessType = "service_request"
		ctx.Category = "hr"

	// General scenarios
	case scenario.ScenarioGeneralTicket:
		ctx.BusinessType = "ticket"

	case scenario.ScenarioServiceRequest:
		ctx.BusinessType = "service_request"

	default:
		ctx.BusinessType = "ticket"
	}
}

// InitDepartmentDefaults initializes default process templates for a department
func (s *DepartmentProcessService) InitDepartmentDefaults(
	ctx context.Context,
	tenantID, departmentID int,
	departmentType string,
) error {
	s.logger.Infow("Initializing department defaults",
		"tenant_id", tenantID,
		"department_id", departmentID,
		"department_type", departmentType,
	)

	templates := scenario.GetAllTemplates()[departmentType]
	if templates == nil {
		return fmt.Errorf("unknown department type: %s", departmentType)
	}

	created := 0
	for _, tmpl := range templates {
		// Check if binding already exists
		existing, err := s.routingService.ListBindings(ctx, tenantID, map[string]interface{}{
			"department_id": departmentID,
			"scenario":      string(tmpl.Scenario),
		})
		if err != nil {
			s.logger.Warnw("Failed to check existing bindings", "error", err)
			continue
		}

		if len(existing) > 0 {
			s.logger.Infow("Binding already exists, skipping",
				"scenario", tmpl.Scenario,
			)
			continue
		}

		// Create new binding
		binding := &ent.ProcessBinding{
			BusinessType:         "ticket", // Default, will be overridden by scenario mapping
			ProcessDefinitionKey: tmpl.ProcessKey,
			ProcessVersion:       1,
			Priority:             tmpl.Priority,
			IsActive:             true,
			DepartmentID:         departmentID,
			Scenario:             string(tmpl.Scenario),
			Category:             departmentType,
			Conditions:           tmpl.Conditions,
			TenantID:             tenantID,
		}

		_, err = s.routingService.CreateBinding(ctx, binding)
		if err != nil {
			s.logger.Warnw("Failed to create binding",
				"scenario", tmpl.Scenario,
				"error", err,
			)
			continue
		}

		created++
		s.logger.Infow("Created department process binding",
			"scenario", tmpl.Scenario,
			"process_key", tmpl.ProcessKey,
		)
	}

	s.logger.Infow("Department defaults initialized",
		"department_id", departmentID,
		"created", created,
	)

	return nil
}

// GetDepartmentInfo returns department information
func (s *DepartmentProcessService) GetDepartmentInfo(
	ctx context.Context,
	tenantID, departmentID int,
) (*ent.Department, error) {
	return s.client.Department.Query().
		Where(
			department.IDEQ(departmentID),
			department.TenantIDEQ(tenantID),
		).
		Only(ctx)
}

// ListDepartmentProcesses lists all process bindings for a department
func (s *DepartmentProcessService) ListDepartmentProcesses(
	ctx context.Context,
	tenantID, departmentID int,
) ([]*ent.ProcessBinding, error) {
	return s.routingService.ListBindings(ctx, tenantID, map[string]interface{}{
		"department_id": departmentID,
	})
}

package service

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/department"
	"itsm-backend/ent/domainconfig"

	"go.uber.org/zap"
)

// ConfigInheritanceService provides hierarchical configuration with inheritance
type ConfigInheritanceService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewConfigInheritanceService creates a new ConfigInheritanceService
func NewConfigInheritanceService(client *ent.Client, logger *zap.SugaredLogger) *ConfigInheritanceService {
	return &ConfigInheritanceService{
		client: client,
		logger: logger,
	}
}

// ResolvedConfig represents a fully resolved configuration
type ResolvedConfig struct {
	Key         string                 `json:"key"`
	Value       map[string]interface{} `json:"value"`
	Source      string                 `json:"source"` // global, tenant, department, team
	InheritMode string                 `json:"inheritMode"`
	Version     int                    `json:"version"`
}

// InheritanceLevel represents a level in the inheritance chain
type InheritanceLevel struct {
	TenantID     int
	DepartmentID int
	TeamID       int
	Source       string
}

// GetEffectiveConfig resolves configuration with inheritance chain
// Chain: Global -> Tenant -> Department(ancestors) -> Team
func (s *ConfigInheritanceService) GetEffectiveConfig(
	ctx context.Context,
	tenantID, departmentID, teamID int,
	configType, configKey string,
) (*ResolvedConfig, error) {
	s.logger.Infow(
		"Resolving effective config",
		"tenant_id", tenantID,
		"department_id", departmentID,
		"team_id", teamID,
		"config_type", configType,
		"config_key", configKey,
	)

	// Build inheritance chain
	chain := s.buildInheritanceChain(ctx, tenantID, departmentID, teamID)

	var baseConfig map[string]interface{}
	var resolvedMode string
	var resolvedSource string
	var resolvedVersion int

	// Walk chain from global to specific
	for _, level := range chain {
		config, err := s.client.DomainConfig.Query().
			Where(
				domainconfig.TenantIDEQ(level.TenantID),
				domainconfig.DepartmentIDEQ(level.DepartmentID),
				domainconfig.TeamIDEQ(level.TeamID),
				domainconfig.ConfigTypeEQ(configType),
				domainconfig.ConfigKeyEQ(configKey),
				domainconfig.IsActiveEQ(true),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				continue // No config at this level, inherit from parent
			}
			return nil, fmt.Errorf("failed to query config: %w", err)
		}

		switch config.InheritMode {
		case "override":
			// Complete replacement at this level, then allow more specific levels to override/extend it.
			s.logger.Infow(
				"Config override found",
				"source", level.Source,
				"key", configKey,
			)
			baseConfig = cloneConfig(config.ConfigValue)
			resolvedMode = "override"
			resolvedSource = level.Source
			resolvedVersion = config.Version

		case "extend":
			// Merge with parent
			if baseConfig == nil {
				baseConfig = make(map[string]interface{})
			}
			s.mergeConfigs(baseConfig, config.ConfigValue)
			resolvedMode = "extend"
			resolvedSource = level.Source
			resolvedVersion = config.Version

		case "inherit":
			// Use parent value (do nothing at this level)
			continue
		}
	}

	if baseConfig == nil {
		return nil, fmt.Errorf("no configuration found for %s/%s", configType, configKey)
	}

	return &ResolvedConfig{
		Key:         configKey,
		Value:       baseConfig,
		Source:      resolvedSource,
		InheritMode: resolvedMode,
		Version:     resolvedVersion,
	}, nil
}

// buildInheritanceChain builds the inheritance chain from global to specific
func (s *ConfigInheritanceService) buildInheritanceChain(
	ctx context.Context,
	tenantID, departmentID, teamID int,
) []InheritanceLevel {
	chain := []InheritanceLevel{
		{TenantID: 0, DepartmentID: 0, TeamID: 0, Source: "global"},        // Global defaults
		{TenantID: tenantID, DepartmentID: 0, TeamID: 0, Source: "tenant"}, // Tenant level
	}

	if departmentID > 0 {
		// Get department ancestors
		deptAncestors := s.getDepartmentAncestors(ctx, tenantID, departmentID)
		for _, deptID := range deptAncestors {
			chain = append(chain, InheritanceLevel{
				TenantID:     tenantID,
				DepartmentID: deptID,
				TeamID:       0,
				Source:       fmt.Sprintf("department:%d", deptID),
			})
		}
	}

	if teamID > 0 {
		chain = append(chain, InheritanceLevel{
			TenantID:     tenantID,
			DepartmentID: departmentID,
			TeamID:       teamID,
			Source:       fmt.Sprintf("team:%d", teamID),
		})
	}

	return chain
}

// getDepartmentAncestors returns department IDs from root to specified department
func (s *ConfigInheritanceService) getDepartmentAncestors(
	ctx context.Context,
	tenantID, departmentID int,
) []int {
	ancestors := []int{}
	currentID := departmentID

	// Traverse up the department tree
	for currentID > 0 {
		ancestors = append([]int{currentID}, ancestors...) // Prepend to get root-first order

		dept, err := s.client.Department.Query().
			Where(
				department.IDEQ(currentID),
				department.TenantIDEQ(tenantID),
			).
			Only(ctx)
		if err != nil {
			break
		}

		currentID = dept.ParentID
	}

	return ancestors
}

// mergeConfigs merges source into target (source overrides target)
func (s *ConfigInheritanceService) mergeConfigs(target, source map[string]interface{}) {
	for k, v := range source {
		target[k] = v
	}
}

func cloneConfig(source map[string]interface{}) map[string]interface{} {
	cloned := make(map[string]interface{}, len(source))
	for k, v := range source {
		cloned[k] = v
	}
	return cloned
}

// SetConfig sets a configuration at a specific level
func (s *ConfigInheritanceService) SetConfig(
	ctx context.Context,
	tenantID, departmentID, teamID int,
	configType, configKey string,
	configValue map[string]interface{},
	inheritMode string,
	description string,
) error {
	// Check if config already exists
	existing, err := s.client.DomainConfig.Query().
		Where(
			domainconfig.TenantIDEQ(tenantID),
			domainconfig.DepartmentIDEQ(departmentID),
			domainconfig.TeamIDEQ(teamID),
			domainconfig.ConfigTypeEQ(configType),
			domainconfig.ConfigKeyEQ(configKey),
		).
		Only(ctx)

	if err != nil && !ent.IsNotFound(err) {
		return fmt.Errorf("failed to query existing config: %w", err)
	}

	if existing != nil {
		// Update existing config
		_, err = s.client.DomainConfig.UpdateOneID(existing.ID).
			SetConfigValue(configValue).
			SetInheritMode(inheritMode).
			SetDescription(description).
			SetVersion(existing.Version + 1).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to update config: %w", err)
		}
	} else {
		// Create new config
		_, err = s.client.DomainConfig.Create().
			SetTenantID(tenantID).
			SetDepartmentID(departmentID).
			SetTeamID(teamID).
			SetConfigType(configType).
			SetConfigKey(configKey).
			SetConfigValue(configValue).
			SetInheritMode(inheritMode).
			SetDescription(description).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}
	}

	return nil
}

// ListConfigs lists all configurations for a tenant
func (s *ConfigInheritanceService) ListConfigs(
	ctx context.Context,
	tenantID int,
	configType string,
) ([]*ent.DomainConfig, error) {
	query := s.client.DomainConfig.Query().
		Where(domainconfig.TenantIDIn(0, tenantID))

	if configType != "" {
		query = query.Where(domainconfig.ConfigTypeEQ(configType))
	}

	return query.Order(ent.Asc(domainconfig.FieldConfigType), ent.Asc(domainconfig.FieldConfigKey)).All(ctx)
}

package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/department"
	"itsm-backend/ent/domainconfig"

	"go.uber.org/zap"
)

// ErrConfigNotFound 表示全局 → 租户 → 部门 → 团队 各层级均未命中配置。
// 这是合法的空结果而非内部错误：调用方必须用 errors.Is 区分，并映射为
// data=null 的成功响应，不能返回 500/5001。
var ErrConfigNotFound = dto.ErrConfigNotFound

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
type ResolvedConfig = dto.ResolvedConfig

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
	chain, err := s.buildInheritanceChain(ctx, tenantID, departmentID, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to build inheritance chain: %w", err)
	}

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
		case dto.InheritModeOverride:
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

		case dto.InheritModeExtend:
			// Merge with parent
			if baseConfig == nil {
				baseConfig = make(map[string]interface{})
			}
			s.mergeConfigs(baseConfig, config.ConfigValue)
			resolvedMode = "extend"
			resolvedSource = level.Source
			resolvedVersion = config.Version

		case dto.InheritModeInherit:
			// Use parent value (do nothing at this level)
			continue
		default:
			// 历史脏数据或绕过写入口直接改库留下的非法值：不能当成 inherit
			// 悄悄跳过，必须可观测；但也不要让一次解析整体 500。
			s.logger.Warnw("Unknown inherit mode, level ignored",
				"mode", config.InheritMode,
				"config_type", configType,
				"config_key", configKey,
				"source", level.Source,
			)
			continue
		}
	}

	if baseConfig == nil {
		return nil, fmt.Errorf("%w: %s/%s", ErrConfigNotFound, configType, configKey)
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
) ([]InheritanceLevel, error) {
	chain := []InheritanceLevel{
		{TenantID: 0, DepartmentID: 0, TeamID: 0, Source: "global"},        // Global defaults
		{TenantID: tenantID, DepartmentID: 0, TeamID: 0, Source: "tenant"}, // Tenant level
	}

	if departmentID > 0 {
		// Get department ancestors
		deptAncestors, err := s.getDepartmentAncestors(ctx, tenantID, departmentID)
		if err != nil {
			return nil, err
		}
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

	return chain, nil
}

// getDepartmentAncestors returns department IDs from root to specified department.
// 只有「部门记录不存在」是合法的链终止（按 ID 存的域配置仍应可解析）；
// 其余 DB 错误必须向上传播，不能吞掉后伪装成一条更短的继承链。
func (s *ConfigInheritanceService) getDepartmentAncestors(
	ctx context.Context,
	tenantID, departmentID int,
) ([]int, error) {
	ancestors := []int{}
	visited := make(map[int]struct{})
	currentID := departmentID

	// Traverse up the department tree
	for currentID > 0 {
		// ParentID 自引用 / 成环会让原实现无限循环。命中已访问节点即停，
		// 保留已解析出的链而不是让请求挂死。
		if _, seen := visited[currentID]; seen {
			s.logger.Warnw("Department ancestor cycle detected",
				"tenant_id", tenantID, "department_id", currentID)
			break
		}
		visited[currentID] = struct{}{}
		ancestors = append([]int{currentID}, ancestors...) // Prepend to get root-first order

		dept, err := s.client.Department.Query().
			Where(
				department.IDEQ(currentID),
				department.TenantIDEQ(tenantID),
				department.DeletedAtIsNil(),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				break
			}
			return nil, fmt.Errorf("failed to query department %d: %w", currentID, err)
		}

		currentID = dept.ParentID
	}

	return ancestors, nil
}

// mergeConfigs merges source into target (source overrides target).
// extend 的语义是「在父级配置上叠加」，嵌套对象必须递归合并，否则子级只改
// 一个字段就会把父级的整个嵌套对象替换掉。
func (s *ConfigInheritanceService) mergeConfigs(target, source map[string]interface{}) {
	for k, v := range source {
		if srcMap, ok := v.(map[string]interface{}); ok {
			if dstMap, ok := target[k].(map[string]interface{}); ok {
				s.mergeConfigs(dstMap, srcMap)
				continue
			}
		}
		target[k] = v
	}
}

// cloneConfig 深拷贝，避免 override 后的 extend 写回父级共享的嵌套 map。
func cloneConfig(source map[string]interface{}) map[string]interface{} {
	cloned := make(map[string]interface{}, len(source))
	for k, v := range source {
		if nested, ok := v.(map[string]interface{}); ok {
			cloned[k] = cloneConfig(nested)
			continue
		}
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

package service

import (
	"context"
	"fmt"
	"strings"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/processbinding"
	"itsm-backend/ent/processdefinition"

	"github.com/pkg/errors"
)

// ProcessBindingService 流程绑定配置服务
type ProcessBindingService struct {
	client *ent.Client
}

// NewProcessBindingService 创建流程绑定服务
func NewProcessBindingService(client *ent.Client) *ProcessBindingService {
	return &ProcessBindingService{client: client}
}

// CreateBinding 创建流程绑定
func (s *ProcessBindingService) CreateBinding(ctx context.Context, binding *dto.ProcessBinding) (*dto.ProcessBinding, error) {
	// 验证流程定义存在
	def, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(binding.ProcessDefinitionKey),
			processdefinition.TenantID(binding.TenantID),
			processdefinition.IsActive(true),
			processdefinition.IsLatest(true),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("流程定义 %s 不存在", binding.ProcessDefinitionKey)
		}
		return nil, errors.Wrap(err, "验证流程定义失败")
	}

	// 如果没有指定版本，使用最新版本
	if binding.ProcessVersion <= 0 {
		// 转换版本字符串为整数
		var version int
		fmt.Sscanf(def.Version, "%d", &version)
		binding.ProcessVersion = version
	}

	// 如果设置为默认，需要取消同类型其他默认绑定
	if binding.IsDefault {
		_, err := s.client.ProcessBinding.Update().
			Where(
				processbinding.BusinessType(string(binding.BusinessType)),
				processbinding.TenantID(binding.TenantID),
				processbinding.IsDefault(true),
			).
			SetIsDefault(false).
			Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "取消其他默认绑定失败")
		}
	}

	entity, err := s.client.ProcessBinding.Create().
		SetBusinessType(string(binding.BusinessType)).
		SetBusinessSubType(binding.BusinessSubType).
		SetProcessDefinitionKey(binding.ProcessDefinitionKey).
		SetProcessVersion(binding.ProcessVersion).
		SetIsDefault(binding.IsDefault).
		SetPriority(binding.Priority).
		SetIsActive(binding.IsActive).
		SetDepartmentID(binding.DepartmentID).
		SetTeamID(binding.TeamID).
		SetScenario(binding.Scenario).
		SetCategory(binding.Category).
		SetConditions(binding.Conditions).
		SetApprovalChainID(binding.ApprovalChainID).
		SetSLAPolicyID(binding.SLAPolicyID).
		SetOverrides(binding.Overrides).
		SetTenantID(binding.TenantID).
		Save(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "创建流程绑定失败")
	}

	return s.toBindingResponse(entity), nil
}

// UpdateBinding 更新流程绑定
func (s *ProcessBindingService) UpdateBinding(ctx context.Context, id int, binding *dto.ProcessBinding) (*dto.ProcessBinding, error) {
	entity, err := s.client.ProcessBinding.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "查询流程绑定失败")
	}

	// 如果设置为默认，需要取消同类型其他默认绑定
	if binding.IsDefault && !entity.IsDefault {
		_, err := s.client.ProcessBinding.Update().
			Where(
				processbinding.BusinessType(entity.BusinessType),
				processbinding.TenantID(entity.TenantID),
				processbinding.IDNEQ(id),
				processbinding.IsDefault(true),
			).
			SetIsDefault(false).
			Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "取消其他默认绑定失败")
		}
	}

	update := entity.Update()
	if binding.ProcessDefinitionKey != "" {
		update.SetProcessDefinitionKey(binding.ProcessDefinitionKey)
	}
	if binding.ProcessVersion > 0 {
		update.SetProcessVersion(binding.ProcessVersion)
	}
	if binding.BusinessSubType != "" {
		update.SetBusinessSubType(binding.BusinessSubType)
	}
	update.SetIsDefault(binding.IsDefault)
	update.SetPriority(binding.Priority)
	update.SetIsActive(binding.IsActive)
	update.SetDepartmentID(binding.DepartmentID)
	update.SetTeamID(binding.TeamID)
	update.SetScenario(binding.Scenario)
	update.SetCategory(binding.Category)
	if binding.Conditions != nil {
		update.SetConditions(binding.Conditions)
	}
	update.SetApprovalChainID(binding.ApprovalChainID)
	update.SetSLAPolicyID(binding.SLAPolicyID)
	if binding.Overrides != nil {
		update.SetOverrides(binding.Overrides)
	}

	entity, err = update.Save(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "更新流程绑定失败")
	}

	return s.toBindingResponse(entity), nil
}

// DeleteBinding 删除流程绑定
func (s *ProcessBindingService) DeleteBinding(ctx context.Context, id int, tenantID int) error {
	entity, err := s.client.ProcessBinding.Get(ctx, id)
	if err != nil {
		return errors.Wrap(err, "查询流程绑定失败")
	}

	if entity.TenantID != tenantID {
		return fmt.Errorf("无权操作此流程绑定")
	}

	return s.client.ProcessBinding.DeleteOne(entity).Exec(ctx)
}

// GetBinding 获取流程绑定
func (s *ProcessBindingService) GetBinding(ctx context.Context, id int, tenantID int) (*dto.ProcessBinding, error) {
	entity, err := s.client.ProcessBinding.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "查询流程绑定失败")
	}

	if entity.TenantID != tenantID {
		return nil, fmt.Errorf("无权访问此流程绑定")
	}

	return s.toBindingResponse(entity), nil
}

// QueryBindings 查询流程绑定列表
func (s *ProcessBindingService) QueryBindings(ctx context.Context, req *dto.ProcessBindingQueryRequest) ([]*dto.ProcessBinding, error) {
	query := s.client.ProcessBinding.Query().
		Where(processbinding.TenantID(req.TenantID))

	if req.BusinessType != "" {
		query = query.Where(processbinding.BusinessType(string(req.BusinessType)))
	}
	if req.BusinessSubType != "" {
		query = query.Where(processbinding.BusinessSubType(req.BusinessSubType))
	}
	if req.DepartmentID > 0 {
		query = query.Where(processbinding.DepartmentID(req.DepartmentID))
	}
	if req.TeamID > 0 {
		query = query.Where(processbinding.TeamID(req.TeamID))
	}
	if req.Scenario != "" {
		query = query.Where(processbinding.Scenario(req.Scenario))
	}
	if req.Category != "" {
		query = query.Where(processbinding.Category(req.Category))
	}
	if req.IsActive != nil {
		query = query.Where(processbinding.IsActive(*req.IsActive))
	}

	entities, err := query.
		Order(ent.Desc("priority"), ent.Desc("created_at")).
		All(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "查询流程绑定失败")
	}

	bindings := make([]*dto.ProcessBinding, 0, len(entities))
	for _, e := range entities {
		bindings = append(bindings, s.toBindingResponse(e))
	}

	return bindings, nil
}

// FindBestBinding 根据业务类型查找最佳流程绑定
func (s *ProcessBindingService) FindBestBinding(ctx context.Context, businessType dto.BusinessType, subType string, tenantID int) (*dto.ProcessBinding, error) {
	query := s.client.ProcessBinding.Query().
		Where(
			processbinding.BusinessType(string(businessType)),
			processbinding.TenantID(tenantID),
			processbinding.IsActive(true),
		)

	// 如果有子类型，优先匹配子类型
	if subType != "" {
		query = query.Where(processbinding.BusinessSubType(subType))
	}

	// 查找匹配的绑定
	bindings, err := query.Order(ent.Desc("priority"), ent.Desc("is_default")).All(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "查询流程绑定失败")
	}

	if len(bindings) == 0 {
		// 查找默认绑定
		defaultBindings, err := s.client.ProcessBinding.Query().
			Where(
				processbinding.BusinessType(string(businessType)),
				processbinding.BusinessSubType(""),
				processbinding.TenantID(tenantID),
				processbinding.IsActive(true),
				processbinding.IsDefault(true),
			).
			All(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "查询默认流程绑定失败")
		}

		if len(defaultBindings) == 0 {
			return nil, nil // 没有找到匹配的绑定
		}

		return s.toBindingResponse(defaultBindings[0]), nil
	}

	// 返回优先级最高的绑定
	return s.toBindingResponse(bindings[0]), nil
}

// BatchCreateBindings 批量创建流程绑定
func (s *ProcessBindingService) BatchCreateBindings(ctx context.Context, req *dto.BatchProcessBindingRequest) error {
	for i, binding := range req.Bindings {
		_, err := s.CreateBinding(ctx, &binding)
		if err != nil {
			return fmt.Errorf("创建流程绑定失败（索引 %d）: %w", i, err)
		}
	}
	return nil
}

// toBindingResponse 转换为响应
func (s *ProcessBindingService) toBindingResponse(entity *ent.ProcessBinding) *dto.ProcessBinding {
	return &dto.ProcessBinding{
		ID:                   entity.ID,
		BusinessType:         dto.BusinessType(entity.BusinessType),
		BusinessSubType:      entity.BusinessSubType,
		ProcessDefinitionKey: entity.ProcessDefinitionKey,
		ProcessVersion:       entity.ProcessVersion,
		IsDefault:            entity.IsDefault,
		Priority:             entity.Priority,
		IsActive:             entity.IsActive,
		DepartmentID:         entity.DepartmentID,
		TeamID:               entity.TeamID,
		Scenario:             entity.Scenario,
		Category:             entity.Category,
		Conditions:           entity.Conditions,
		ApprovalChainID:      entity.ApprovalChainID,
		SLAPolicyID:          entity.SLAPolicyID,
		Overrides:            entity.Overrides,
		TenantID:             entity.TenantID,
		CreatedAt:            entity.CreatedAt,
		UpdatedAt:            entity.UpdatedAt,
	}
}

// InitDefaultBindings 初始化默认流程绑定（系统初始化时调用）
func (s *ProcessBindingService) InitDefaultBindings(ctx context.Context, tenantID int) error {
	defaultBindings := []dto.ProcessBinding{
		// 工单流程
		{
			BusinessType:         dto.BusinessTypeTicket,
			BusinessSubType:      "general",
			ProcessDefinitionKey: "ticket_general_flow",
			Priority:             10,
			IsDefault:            true,
			IsActive:             true,
		},
		// 事件流程
		{
			BusinessType:         dto.BusinessTypeIncident,
			BusinessSubType:      "",
			ProcessDefinitionKey: "incident_emergency_flow",
			Priority:             10,
			IsDefault:            true,
			IsActive:             true,
		},
		// 变更流程
		{
			BusinessType:         dto.BusinessTypeChange,
			BusinessSubType:      "normal",
			ProcessDefinitionKey: "change_normal_flow",
			Priority:             10,
			IsDefault:            true,
			IsActive:             true,
		},
		{
			BusinessType:         dto.BusinessTypeChange,
			BusinessSubType:      "emergency",
			ProcessDefinitionKey: "change_emergency_flow",
			Priority:             20,
			IsDefault:            false,
			IsActive:             true,
		},
		// 服务请求流程
		{
			BusinessType:         dto.BusinessTypeServiceRequest,
			BusinessSubType:      "",
			ProcessDefinitionKey: "service_request_flow",
			Priority:             10,
			IsDefault:            true,
			IsActive:             true,
		},
		// 问题管理流程
		{
			BusinessType:         dto.BusinessTypeProblem,
			BusinessSubType:      "",
			ProcessDefinitionKey: "problem_management_flow",
			Priority:             10,
			IsDefault:            true,
			IsActive:             true,
		},
		// 发布审批流程
		{
			BusinessType:         dto.BusinessTypeRelease,
			BusinessSubType:      "",
			ProcessDefinitionKey: "release_approval_flow",
			Priority:             10,
			IsDefault:            true,
			IsActive:             true,
		},
		// 工单分配流程
		{
			BusinessType:         dto.BusinessTypeTicket,
			BusinessSubType:      "assignment",
			ProcessDefinitionKey: "ticket_assignment_flow",
			Priority:             20,
			IsDefault:            false,
			IsActive:             true,
		},
	}

	for _, binding := range defaultBindings {
		binding.TenantID = tenantID
		binding.ProcessVersion = 1

		// 检查流程定义是否存在（仅检查 IsActive，不过滤 IsLatest 以兼容已有版本）
		exists, err := s.client.ProcessDefinition.Query().
			Where(
				processdefinition.Key(binding.ProcessDefinitionKey),
				processdefinition.TenantID(tenantID),
				processdefinition.IsActive(true),
			).
			Exist(ctx)
		if err != nil || !exists {
			// 流程定义不存在，跳过此绑定
			continue
		}

		_, err = s.CreateBinding(ctx, &binding)
		if err != nil {
			// 忽略已存在的错误
			if !strings.Contains(err.Error(), "already exists") {
				continue // 跳过不关键的错误
			}
		}
	}

	return nil
}

// GetBindingsByBusinessType 根据业务类型获取所有绑定
func (s *ProcessBindingService) GetBindingsByBusinessType(ctx context.Context, businessType dto.BusinessType, tenantID int) ([]*dto.ProcessBinding, error) {
	entities, err := s.client.ProcessBinding.Query().
		Where(
			processbinding.BusinessType(string(businessType)),
			processbinding.TenantID(tenantID),
			processbinding.IsActive(true),
		).
		Order(ent.Desc("priority")).
		All(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "查询流程绑定失败")
	}

	bindings := make([]*dto.ProcessBinding, 0, len(entities))
	for _, e := range entities {
		bindings = append(bindings, s.toBindingResponse(e))
	}

	return bindings, nil
}

// GetDepartmentBindings returns process bindings scoped to one department.
func (s *ProcessBindingService) GetDepartmentBindings(ctx context.Context, tenantID, departmentID int) ([]*dto.ProcessBinding, error) {
	req := &dto.ProcessBindingQueryRequest{
		TenantID:     tenantID,
		DepartmentID: departmentID,
	}
	return s.QueryBindings(ctx, req)
}

// InitDepartmentDefaultBindings initializes scenario-specific bindings for a department.
func (s *ProcessBindingService) InitDepartmentDefaultBindings(ctx context.Context, tenantID, departmentID int, departmentType string) error {
	bindings := getDepartmentDefaultBindings(departmentType)
	if len(bindings) == 0 {
		return fmt.Errorf("未知部门类型: %s", departmentType)
	}

	for _, binding := range bindings {
		binding.TenantID = tenantID
		binding.DepartmentID = departmentID
		binding.IsActive = true

		exists, err := s.client.ProcessBinding.Query().
			Where(
				processbinding.TenantID(tenantID),
				processbinding.DepartmentID(departmentID),
				processbinding.BusinessType(string(binding.BusinessType)),
				processbinding.BusinessSubType(binding.BusinessSubType),
				processbinding.Scenario(binding.Scenario),
			).
			Exist(ctx)
		if err != nil {
			return errors.Wrap(err, "检查部门流程绑定失败")
		}
		if exists {
			continue
		}

		definitionExists, err := s.client.ProcessDefinition.Query().
			Where(
				processdefinition.Key(binding.ProcessDefinitionKey),
				processdefinition.TenantID(tenantID),
				processdefinition.IsActive(true),
			).
			Exist(ctx)
		if err != nil {
			return errors.Wrap(err, "检查流程定义失败")
		}
		if !definitionExists {
			continue
		}

		_, err = s.CreateBinding(ctx, &binding)
		if err != nil {
			return errors.Wrap(err, "初始化部门流程绑定失败")
		}
	}
	return nil
}

func getDepartmentDefaultBindings(departmentType string) []dto.ProcessBinding {
	switch departmentType {
	case "operations":
		return []dto.ProcessBinding{
			{BusinessType: dto.BusinessTypeIncident, BusinessSubType: "alert_p0", ProcessDefinitionKey: "incident_emergency_flow", ProcessVersion: 1, Scenario: "alert_handling", Category: "operations", Priority: 100, Conditions: map[string]interface{}{"severity": "p0"}},
			{BusinessType: dto.BusinessTypeIncident, BusinessSubType: "alert_p1", ProcessDefinitionKey: "incident_emergency_flow", ProcessVersion: 1, Scenario: "alert_handling", Category: "operations", Priority: 90, Conditions: map[string]interface{}{"severity": "p1"}},
			{BusinessType: dto.BusinessTypeChange, BusinessSubType: "normal", ProcessDefinitionKey: "change_normal_flow", ProcessVersion: 1, Scenario: "change_release", Category: "operations", Priority: 70},
			{BusinessType: dto.BusinessTypeChange, BusinessSubType: "emergency", ProcessDefinitionKey: "change_emergency_flow", ProcessVersion: 1, Scenario: "emergency_change", Category: "operations", Priority: 90},
		}
	case "rd":
		return []dto.ProcessBinding{
			{BusinessType: dto.BusinessTypeRelease, BusinessSubType: "production", ProcessDefinitionKey: "release_approval_flow", ProcessVersion: 1, Scenario: "code_release_prod", Category: "rd", Priority: 90, Conditions: map[string]interface{}{"environment": "production"}},
			{BusinessType: dto.BusinessTypeRelease, BusinessSubType: "testing", ProcessDefinitionKey: "release_test_flow", ProcessVersion: 1, Scenario: "code_release_test", Category: "rd", Priority: 70, Conditions: map[string]interface{}{"environment": "testing"}},
			{BusinessType: dto.BusinessTypeChange, BusinessSubType: "requirement", ProcessDefinitionKey: "change_requirement_flow", ProcessVersion: 1, Scenario: "requirement_change", Category: "rd", Priority: 80},
		}
	case "finance":
		return []dto.ProcessBinding{
			{BusinessType: dto.BusinessTypeServiceRequest, BusinessSubType: "expense", ProcessDefinitionKey: "expense_approval_flow", ProcessVersion: 1, Scenario: "expense_approval", Category: "finance", Priority: 80},
			{BusinessType: dto.BusinessTypeServiceRequest, BusinessSubType: "budget", ProcessDefinitionKey: "budget_approval_flow", ProcessVersion: 1, Scenario: "budget_approval", Category: "finance", Priority: 90},
			{BusinessType: dto.BusinessTypeServiceRequest, BusinessSubType: "procurement", ProcessDefinitionKey: "procurement_flow", ProcessVersion: 1, Scenario: "procurement", Category: "finance", Priority: 85},
		}
	case "hr":
		return []dto.ProcessBinding{
			{BusinessType: dto.BusinessTypeServiceRequest, BusinessSubType: "leave", ProcessDefinitionKey: "leave_approval_flow", ProcessVersion: 1, Scenario: "leave_approval", Category: "hr", Priority: 70},
			{BusinessType: dto.BusinessTypeServiceRequest, BusinessSubType: "recruitment", ProcessDefinitionKey: "recruitment_approval_flow", ProcessVersion: 1, Scenario: "recruitment_approval", Category: "hr", Priority: 80},
		}
	default:
		return nil
	}
}

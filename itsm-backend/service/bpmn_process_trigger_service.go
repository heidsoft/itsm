package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
	"itsm-backend/service/bpmn"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// ProcessTriggerService 流程触发服务实现
type ProcessTriggerService struct {
	client            *ent.Client
	processEngine     ProcessEngine
	processBindingSvc *ProcessBindingService
	processRoutingSvc *ProcessRoutingService
}

// NewProcessTriggerService 创建流程触发服务
func NewProcessTriggerService(client *ent.Client, engine ProcessEngine) *ProcessTriggerService {
	svc := &ProcessTriggerService{
		client:        client,
		processEngine: engine,
	}
	// 延迟初始化 binding service，避免循环依赖
	bindingSvc := NewProcessBindingService(client)
	svc.processBindingSvc = bindingSvc
	svc.processRoutingSvc = NewProcessRoutingService(client, zap.NewNop().Sugar())
	return svc
}

// TriggerProcess 触发流程
func (s *ProcessTriggerService) TriggerProcess(ctx context.Context, req *dto.ProcessTriggerRequest) (*dto.ProcessTriggerResponse, error) {
	// 1. 验证请求
	if err := s.validateTriggerRequest(req); err != nil {
		return nil, err
	}

	// 2. 如果没有指定流程定义，则根据业务类型查找
	processDefKey := req.ProcessDefinitionKey
	if processDefKey == "" {
		route, err := s.processRoutingSvc.FindBestRoute(ctx, s.buildRoutingContext(req))
		if err != nil {
			return nil, errors.Wrap(err, "查找流程路由失败")
		}
		if route != nil {
			processDefKey = route.ProcessDefinitionKey
		}
	}
	if processDefKey == "" {
		binding, err := s.processBindingSvc.FindBestBinding(ctx, req.BusinessType, req.BusinessSubType, req.TenantID)
		if err != nil {
			return nil, errors.Wrap(err, "查找默认流程绑定失败")
		}
		if binding == nil {
			return nil, fmt.Errorf("未找到业务类型 %s 对应的流程绑定", req.BusinessType)
		}
		processDefKey = binding.ProcessDefinitionKey
	}

	// 3. 获取流程定义（仅要求 IsActive，不过滤 IsLatest 以支持触发特定版本）
	definition, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(processDefKey),
			processdefinition.TenantID(req.TenantID),
			processdefinition.IsActive(true),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("流程定义 %s 不存在", processDefKey)
		}
		return nil, errors.Wrap(err, "查询流程定义失败")
	}

	// 4. 构建 businessKey
	businessKey := s.buildBusinessKey(req.BusinessType, req.BusinessID)

	// 5. 合并变量
	variables := s.buildProcessVariables(req, definition)

	// 6. 设置租户上下文并启动流程
	triggerCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, req.TenantID)
	instance, err := s.processEngine.StartProcess(triggerCtx, processDefKey, businessKey, variables)
	if err != nil {
		return nil, errors.Wrap(err, "启动流程失败")
	}

	// 7. 返回响应
	return &dto.ProcessTriggerResponse{
		ProcessInstanceID:     instance.ID,
		ProcessDefinitionKey:  processDefKey,
		ProcessDefinitionName: definition.Name,
		BusinessKey:           businessKey,
		Status:                s.mapInstanceStatus(instance.Status),
		CurrentActivityID:     instance.CurrentActivityID,
		CurrentActivityName:   instance.CurrentActivityName,
		StartTime:             instance.StartTime,
		Message:               "流程启动成功",
	}, nil
}

func (s *ProcessTriggerService) buildRoutingContext(req *dto.ProcessTriggerRequest) *RoutingContext {
	return &RoutingContext{
		BusinessType:    string(req.BusinessType),
		BusinessSubType: firstNonEmpty(req.BusinessSubType, stringFromVariables(req.Variables, "business_sub_type")),
		TenantID:        req.TenantID,
		DepartmentID:    firstPositive(req.DepartmentID, intFromVariables(req.Variables, "department_id")),
		TeamID:          firstPositive(req.TeamID, intFromVariables(req.Variables, "team_id")),
		ProjectID:       firstPositive(req.ProjectID, intFromVariables(req.Variables, "project_id")),
		Scenario:        firstNonEmpty(req.Scenario, stringFromVariables(req.Variables, "scenario")),
		Category:        firstNonEmpty(req.Category, stringFromVariables(req.Variables, "category")),
		Variables:       req.Variables,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func stringFromVariables(variables map[string]interface{}, key string) string {
	if variables == nil {
		return ""
	}
	if value, ok := variables[key].(string); ok {
		return value
	}
	return ""
}

func intFromVariables(variables map[string]interface{}, key string) int {
	if variables == nil {
		return 0
	}
	switch value := variables[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return 0
}

// TriggerByBusinessType 根据业务类型触发流程
func (s *ProcessTriggerService) TriggerByBusinessType(
	ctx context.Context,
	businessType dto.BusinessType,
	businessID int,
	variables map[string]interface{},
	triggeredBy string,
	tenantID int,
) (*dto.ProcessTriggerResponse, error) {
	req := &dto.ProcessTriggerRequest{
		BusinessType: businessType,
		BusinessID:   businessID,
		Variables:    variables,
		TriggeredBy:  triggeredBy,
		TriggeredAt:  time.Now(),
		TenantID:     tenantID,
	}
	return s.TriggerProcess(ctx, req)
}

// CancelProcess 取消流程
func (s *ProcessTriggerService) CancelProcess(ctx context.Context, processInstanceID int, reason string, tenantID int) error {
	return s.processEngine.TerminateProcess(ctx, fmt.Sprintf("%d", processInstanceID), reason)
}

// SuspendProcess 暂停流程
func (s *ProcessTriggerService) SuspendProcess(ctx context.Context, processInstanceID int, reason string, tenantID int) error {
	return s.processEngine.SuspendProcess(ctx, fmt.Sprintf("%d", processInstanceID), reason)
}

// ResumeProcess 恢复流程
func (s *ProcessTriggerService) ResumeProcess(ctx context.Context, processInstanceID int, tenantID int) error {
	return s.processEngine.ResumeProcess(ctx, fmt.Sprintf("%d", processInstanceID))
}

// GetProcessStatus 获取流程状态
func (s *ProcessTriggerService) GetProcessStatus(ctx context.Context, processInstanceID int, tenantID int) (*dto.ProcessTriggerResponse, error) {
	instance, err := s.client.ProcessInstance.Query().
		Where(
			processinstance.ID(processInstanceID),
			processinstance.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("流程实例 %d 不存在", processInstanceID)
		}
		return nil, errors.Wrap(err, "查询流程实例失败")
	}

	definition, _ := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(instance.ProcessDefinitionKey),
			processdefinition.TenantID(tenantID),
			processdefinition.IsLatest(true),
		).
		First(ctx)
	processName := instance.ProcessDefinitionKey
	if definition != nil {
		processName = definition.Name
	}

	return &dto.ProcessTriggerResponse{
		ProcessInstanceID:     instance.ID,
		ProcessDefinitionKey:  instance.ProcessDefinitionKey,
		ProcessDefinitionName: processName,
		BusinessKey:           instance.BusinessKey,
		Status:                s.mapInstanceStatus(instance.Status),
		CurrentActivityID:     instance.CurrentActivityID,
		CurrentActivityName:   instance.CurrentActivityName,
		StartTime:             instance.StartTime,
		EndTime:               &instance.EndTime,
	}, nil
}

// validateTriggerRequest 验证触发请求
func (s *ProcessTriggerService) validateTriggerRequest(req *dto.ProcessTriggerRequest) error {
	if req.BusinessType == "" {
		return fmt.Errorf("业务类型不能为空")
	}
	if req.BusinessID <= 0 {
		return fmt.Errorf("业务ID必须大于0")
	}
	if req.TenantID <= 0 {
		return fmt.Errorf("租户ID必须大于0")
	}
	return nil
}

// buildBusinessKey 构建业务键
func (s *ProcessTriggerService) buildBusinessKey(businessType dto.BusinessType, businessID int) string {
	return fmt.Sprintf("%s:%d", strings.ToLower(string(businessType)), businessID)
}

// buildProcessVariables 构建流程变量
func (s *ProcessTriggerService) buildProcessVariables(req *dto.ProcessTriggerRequest, definition *ent.ProcessDefinition) map[string]interface{} {
	variables := make(map[string]interface{})

	// 1. 添加标准业务变量
	variables["business_type"] = string(req.BusinessType)
	variables["business_id"] = req.BusinessID
	variables["business_key"] = s.buildBusinessKey(req.BusinessType, req.BusinessID)

	// 2. 合并用户提供的变量
	if req.Variables != nil {
		for k, v := range req.Variables {
			variables[k] = v
		}
	}

	// 3. 添加触发者信息
	if req.TriggeredBy != "" {
		variables["triggered_by"] = req.TriggeredBy
		variables["triggered_at"] = req.TriggeredAt.Format(time.RFC3339)
	}

	// 4. 添加租户信息
	variables["tenant_id"] = req.TenantID

	// 5. 如果流程定义有默认变量，合并进来
	if definition.ProcessVariables != nil {
		for k, v := range definition.ProcessVariables {
			if _, exists := variables[k]; !exists {
				variables[k] = v
			}
		}
	}

	return variables
}

// mapInstanceStatus 映射实例状态
func (s *ProcessTriggerService) mapInstanceStatus(status string) dto.ProcessStatus {
	switch status {
	case "running", "active":
		return dto.ProcessStatusRunning
	case "completed":
		return dto.ProcessStatusCompleted
	case "suspended":
		return dto.ProcessStatusSuspended
	case "terminated", "cancelled":
		return dto.ProcessStatusTerminated
	default:
		return dto.ProcessStatusPending
	}
}

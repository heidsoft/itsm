package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// ProcessTriggerService 流程触发服务实现
type ProcessTriggerService struct {
	client            *ent.Client
	processEngine     ProcessEngine
	processBindingSvc *ProcessBindingService
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
		binding, err := s.processBindingSvc.FindBestBinding(ctx, req.BusinessType, "", req.TenantID)
		if err != nil {
			return nil, errors.Wrap(err, "查找流程绑定失败")
		}
		if binding == nil {
			return nil, fmt.Errorf("未找到业务类型 %s 对应的流程绑定", req.BusinessType)
		}
		processDefKey = binding.ProcessDefinitionKey
	}

	// 3. 获取流程定义
	definition, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(processDefKey),
			processdefinition.TenantID(req.TenantID),
		).
		Only(ctx)
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

	// 6. 启动流程
	instance, err := s.processEngine.StartProcess(ctx, processDefKey, businessKey, variables)
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
		Where(processdefinition.Key(instance.ProcessDefinitionKey)).
		Only(ctx)

	return &dto.ProcessTriggerResponse{
		ProcessInstanceID:     instance.ID,
		ProcessDefinitionKey:  instance.ProcessDefinitionKey,
		ProcessDefinitionName: definition.Name,
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

package service

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"

	"go.uber.org/zap"
)

// BPMNTenantService BPMN租户服务
type BPMNTenantService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewBPMNTenantService 创建BPMN租户服务
func NewBPMNTenantService(client *ent.Client, logger *zap.SugaredLogger) *BPMNTenantService {
	return &BPMNTenantService{
		client: client,
		logger: logger,
	}
}

// FilterByTenantID 根据租户ID过滤查询
func (s *BPMNTenantService) FilterByTenantID(ctx context.Context, tenantID int) context.Context {
	return context.WithValue(ctx, "bpmn_tenant_id", tenantID)
}

// GetTenantIDFromContext 从上下文获取租户ID
func (s *BPMNTenantService) GetTenantIDFromContext(ctx context.Context) (int, bool) {
	tenantID, ok := ctx.Value("bpmn_tenant_id").(int)
	return tenantID, ok
}

// GetProcessDefinitionsByTenant 获取租户下的流程定义
func (s *BPMNTenantService) GetProcessDefinitionsByTenant(ctx context.Context, tenantID int) ([]*ent.ProcessDefinition, error) {
	return s.client.ProcessDefinition.Query().
		Where(processdefinition.TenantID(tenantID)).
		All(ctx)
}

// GetProcessInstancesByTenant 获取租户下的流程实例
func (s *BPMNTenantService) GetProcessInstancesByTenant(ctx context.Context, tenantID int) ([]*ent.ProcessInstance, error) {
	return s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		All(ctx)
}

// GetTasksByTenant 获取租户下的任务
func (s *BPMNTenantService) GetTasksByTenant(ctx context.Context, tenantID int) ([]*ent.ProcessTask, error) {
	return s.client.ProcessTask.Query().
		Where(processtask.TenantID(tenantID)).
		All(ctx)
}

// CountProcessDefinitionsByTenant 统计租户下的流程定义数量
func (s *BPMNTenantService) CountProcessDefinitionsByTenant(ctx context.Context, tenantID int) (int, error) {
	return s.client.ProcessDefinition.Query().
		Where(processdefinition.TenantID(tenantID)).
		Count(ctx)
}

// CountProcessInstancesByTenant 统计租户下的流程实例数量
func (s *BPMNTenantService) CountProcessInstancesByTenant(ctx context.Context, tenantID int) (int, error) {
	return s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		Count(ctx)
}

// CountTasksByTenant 统计租户下的任务数量
func (s *BPMNTenantService) CountTasksByTenant(ctx context.Context, tenantID int) (int, error) {
	return s.client.ProcessTask.Query().
		Where(processtask.TenantID(tenantID)).
		Count(ctx)
}

// MigrateProcessDefinition 迁移流程定义到另一个租户
func (s *BPMNTenantService) MigrateProcessDefinition(ctx context.Context, definitionID int, newTenantID int) error {
	_, err := s.client.ProcessDefinition.UpdateOneID(definitionID).
		SetTenantID(newTenantID).
		Save(ctx)
	return err
}

// MigrateProcessInstance 迁移流程实例到另一个租户
func (s *BPMNTenantService) MigrateProcessInstance(ctx context.Context, instanceID int, newTenantID int) error {
	_, err := s.client.ProcessInstance.UpdateOneID(instanceID).
		SetTenantID(newTenantID).
		Save(ctx)
	return err
}

// DeleteTenantData 删除租户下的所有BPMN数据
func (s *BPMNTenantService) DeleteTenantData(ctx context.Context, tenantID int) error {
	// 1. 删除流程任务
	_, err := s.client.ProcessTask.Delete().
		Where(processtask.TenantID(tenantID)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除流程任务失败: %w", err)
	}

	// 2. 删除流程实例
	_, err = s.client.ProcessInstance.Delete().
		Where(processinstance.TenantID(tenantID)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除流程实例失败: %w", err)
	}

	// 3. 删除流程定义
	_, err = s.client.ProcessDefinition.Delete().
		Where(processdefinition.TenantID(tenantID)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除流程定义失败: %w", err)
	}

	return nil
}

// GetTenantStatistics 获取租户统计数据
func (s *BPMNTenantService) GetTenantStatistics(ctx context.Context, tenantID int) (*TenantBPMNStats, error) {
	definitionsCount, err := s.CountProcessDefinitionsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	instancesCount, err := s.CountProcessInstancesByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	tasksCount, err := s.CountTasksByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	runningInstances, err := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.Status("running")).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	completedInstances, err := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.Status("completed")).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	openTasks, err := s.client.ProcessTask.Query().
		Where(processtask.TenantID(tenantID)).
		Where(processtask.StatusNEQ("completed")).
		Where(processtask.StatusNEQ("cancelled")).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	return &TenantBPMNStats{
		TotalDefinitions:    definitionsCount,
		TotalInstances:      instancesCount,
		RunningInstances:    runningInstances,
		CompletedInstances:  completedInstances,
		TotalTasks:         tasksCount,
		OpenTasks:          openTasks,
	}, nil
}

// TenantBPMNStats 租户BPMN统计数据
type TenantBPMNStats struct {
	TotalDefinitions   int `json:"total_definitions"`
	TotalInstances     int `json:"total_instances"`
	RunningInstances   int `json:"running_instances"`
	CompletedInstances int `json:"completed_instances"`
	TotalTasks         int `json:"total_tasks"`
	OpenTasks          int `json:"open_tasks"`
}

// ValidateTenantAccess 验证租户访问权限
func (s *BPMNTenantService) ValidateTenantAccess(ctx context.Context, resourceType string, resourceID int, tenantID int) error {
	switch resourceType {
	case "process_definition":
		definition, err := s.client.ProcessDefinition.Get(ctx, resourceID)
		if err != nil {
			return fmt.Errorf("获取流程定义失败: %w", err)
		}
		if definition.TenantID != tenantID {
			return fmt.Errorf("无权访问该流程定义")
		}
	case "process_instance":
		instance, err := s.client.ProcessInstance.Get(ctx, resourceID)
		if err != nil {
			return fmt.Errorf("获取流程实例失败: %w", err)
		}
		if instance.TenantID != tenantID {
			return fmt.Errorf("无权访问该流程实例")
		}
	case "task":
		task, err := s.client.ProcessTask.Get(ctx, resourceID)
		if err != nil {
			return fmt.Errorf("获取任务失败: %w", err)
		}
		if task.TenantID != tenantID {
			return fmt.Errorf("无权访问该任务")
		}
	}
	return nil
}

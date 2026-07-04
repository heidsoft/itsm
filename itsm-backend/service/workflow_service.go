package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/workflow"
	"itsm-backend/ent/workflowinstance"
)

// WorkflowService 工作流服务
type WorkflowService struct {
	client *ent.Client
}

// NewWorkflowService 创建工作流服务实例
func NewWorkflowService(client *ent.Client) *WorkflowService {
	return &WorkflowService{client: client}
}

// CreateWorkflow 创建工作流
func (s *WorkflowService) CreateWorkflow(ctx context.Context, req *CreateWorkflowRequest) (*ent.Workflow, error) {
	// 验证工作流定义
	if err := s.validateWorkflowDefinition(req.Definition); err != nil {
		return nil, err
	}

	// 序列化工作流定义
	definitionBytes, err := json.Marshal(req.Definition)
	if err != nil {
		return nil, err
	}

	workflow, err := s.client.Workflow.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetType(req.Type).
		SetDefinition(definitionBytes).
		SetVersion(req.Version).
		SetIsActive(req.IsActive).
		SetTenantID(req.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return workflow, nil
}

// GetWorkflow 获取工作流
func (s *WorkflowService) GetWorkflow(ctx context.Context, id int, tenantID int) (*ent.Workflow, error) {
	return s.client.Workflow.Query().
		Where(
			workflow.IDEQ(id),
			workflow.TenantIDEQ(tenantID),
		).
		Only(ctx)
}

// ListWorkflows 获取工作流列表
func (s *WorkflowService) ListWorkflows(ctx context.Context, req *ListWorkflowsRequest) ([]*ent.Workflow, int, error) {
	query := s.client.Workflow.Query()

	// 应用过滤条件
	if req.Type != "" {
		query = query.Where(workflow.Type(req.Type))
	}
	if req.IsActive != nil {
		query = query.Where(workflow.IsActive(*req.IsActive))
	}
	if req.TenantID > 0 {
		query = query.Where(workflow.TenantID(req.TenantID))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 应用分页和排序
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	if req.SortBy != "" {
		switch req.SortBy {
		case "name":
			if req.SortOrder == "desc" {
				query = query.Order(ent.Desc(workflow.FieldName))
			} else {
				query = query.Order(ent.Asc(workflow.FieldName))
			}
		case "created_at":
			if req.SortOrder == "desc" {
				query = query.Order(ent.Desc(workflow.FieldCreatedAt))
			} else {
				query = query.Order(ent.Asc(workflow.FieldCreatedAt))
			}
		}
	}

	workflows, err := query.All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return workflows, total, nil
}

// UpdateWorkflow 更新工作流
func (s *WorkflowService) UpdateWorkflow(ctx context.Context, id int, req *UpdateWorkflowRequest, tenantID int) (*ent.Workflow, error) {
	update := s.client.Workflow.UpdateOneID(id).
		Where(workflow.TenantIDEQ(tenantID))

	if req.Name != "" {
		update.SetName(req.Name)
	}
	if req.Description != "" {
		update.SetDescription(req.Description)
	}
	if req.Type != "" {
		update.SetType(req.Type)
	}
	if req.Definition != nil {
		if err := s.validateWorkflowDefinition(req.Definition); err != nil {
			return nil, err
		}
		definitionBytes, err := json.Marshal(req.Definition)
		if err != nil {
			return nil, err
		}
		update.SetDefinition(definitionBytes)
	}
	if req.Version != "" {
		update.SetVersion(req.Version)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}

	update.SetUpdatedAt(time.Now())

	return update.Save(ctx)
}

// DeleteWorkflow 删除工作流
func (s *WorkflowService) DeleteWorkflow(ctx context.Context, id int, tenantID int) error {
	exists, err := s.client.Workflow.Query().
		Where(
			workflow.IDEQ(id),
			workflow.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("工作流不存在")
	}

	// 检查是否有工作流实例使用此工作流
	count, err := s.client.WorkflowInstance.Query().
		Where(
			workflowinstance.WorkflowIDEQ(id),
			workflowinstance.TenantIDEQ(tenantID),
		).
		Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("无法删除正在使用的工作流")
	}

	return s.client.Workflow.DeleteOneID(id).
		Where(workflow.TenantIDEQ(tenantID)).
		Exec(ctx)
}

// StartWorkflow 启动工作流实例
func (s *WorkflowService) StartWorkflow(ctx context.Context, req *StartWorkflowRequest) (*ent.WorkflowInstance, error) {
	// 获取工作流定义
	workflow, err := s.client.Workflow.Query().
		Where(
			workflow.IDEQ(req.WorkflowID),
			workflow.TenantIDEQ(req.TenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	if !workflow.IsActive {
		return nil, errors.New("工作流未启用")
	}

	// 解析工作流定义
	var definition map[string]interface{}
	if err := json.Unmarshal(workflow.Definition, &definition); err != nil {
		return nil, err
	}

	// 序列化上下文
	contextBytes, err := json.Marshal(req.Context)
	if err != nil {
		return nil, err
	}

	// 创建工作流实例
	instance, err := s.client.WorkflowInstance.Create().
		SetStatus("running").
		SetCurrentStep("start").
		SetContext(contextBytes).
		SetWorkflowID(req.WorkflowID).
		SetEntityID(req.EntityID).
		SetEntityType(req.EntityType).
		SetTenantID(req.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

func (s *WorkflowService) GetWorkflowInstance(ctx context.Context, instanceID int, tenantID int) (*ent.WorkflowInstance, error) {
	return s.client.WorkflowInstance.Query().
		Where(
			workflowinstance.IDEQ(instanceID),
			workflowinstance.TenantIDEQ(tenantID),
		).
		Only(ctx)
}

// ExecuteWorkflowStep 执行工作流步骤
func (s *WorkflowService) ExecuteWorkflowStep(ctx context.Context, req *ExecuteWorkflowStepRequest, tenantID int) error {
	// 获取工作流实例
	instance, err := s.GetWorkflowInstance(ctx, req.InstanceID, tenantID)
	if err != nil {
		return err
	}

	// 获取工作流定义
	workflow, err := s.client.Workflow.Query().
		Where(
			workflow.IDEQ(instance.WorkflowID),
			workflow.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		return err
	}

	// 解析工作流定义
	var definition map[string]interface{}
	if err := json.Unmarshal(workflow.Definition, &definition); err != nil {
		return err
	}

	// 执行步骤逻辑
	// 这里应该实现具体的步骤执行逻辑
	// 例如：发送通知、更新状态、分配任务等

	// 更新实例状态
	update := s.client.WorkflowInstance.UpdateOneID(req.InstanceID).
		Where(workflowinstance.TenantIDEQ(tenantID))
	if req.NextStep != "" {
		update.SetCurrentStep(req.NextStep)
	}
	if req.Status != "" {
		update.SetStatus(req.Status)
	}
	if req.Status == "completed" {
		update.SetCompletedAt(time.Now())
	}

	// 更新上下文
	if req.Context != nil {
		contextBytes, err := json.Marshal(req.Context)
		if err != nil {
			return err
		}
		update.SetContext(contextBytes)
	}

	_, err = update.Save(ctx)
	return err
}

// validateWorkflowDefinition 验证工作流定义
func (s *WorkflowService) validateWorkflowDefinition(definition map[string]interface{}) error {
	// 这里应该实现工作流定义的验证逻辑
	// 例如：检查必需的字段、验证步骤定义等
	return nil
}

// CreateWorkflowRequest 创建工作流请求
type CreateWorkflowRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Type        string                 `json:"type" binding:"required"`
	Definition  map[string]interface{} `json:"definition" binding:"required"`
	Version     string                 `json:"version"`
	IsActive    bool                   `json:"isActive"`
	TenantID    int                    `json:"tenantId" binding:"required"`
}

// UpdateWorkflowRequest 更新工作流请求
type UpdateWorkflowRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Definition  map[string]interface{} `json:"definition"`
	Version     string                 `json:"version"`
	IsActive    *bool                  `json:"isActive"`
}

// ListWorkflowsRequest 获取工作流列表请求
type ListWorkflowsRequest struct {
	Page      int    `json:"page" form:"page"`
	PageSize  int    `json:"pageSize" form:"page_size"`
	Type      string `json:"type" form:"type"`
	IsActive  *bool  `json:"isActive" form:"is_active"`
	TenantID  int    `json:"tenantId" form:"tenant_id"`
	SortBy    string `json:"sortBy" form:"sort_by"`
	SortOrder string `json:"sortOrder" form:"sort_order"`
}

// StartWorkflowRequest 启动工作流请求
type StartWorkflowRequest struct {
	WorkflowID int                    `json:"workflowId" binding:"required"`
	EntityID   int                    `json:"entityId" binding:"required"`
	EntityType string                 `json:"entityType" binding:"required"`
	Context    map[string]interface{} `json:"context"`
	TenantID   int                    `json:"tenantId" binding:"required"`
}

// ExecuteWorkflowStepRequest 执行工作流步骤请求
type ExecuteWorkflowStepRequest struct {
	InstanceID int                    `json:"instanceId" binding:"required"`
	Action     string                 `json:"action" binding:"required"`
	NextStep   string                 `json:"nextStep"`
	Status     string                 `json:"status"`
	Context    map[string]interface{} `json:"context"`
}

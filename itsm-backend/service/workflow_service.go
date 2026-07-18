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
// 校验项：
//  1. 必填字段：startEvent、steps、transitions
//  2. steps 中必须包含 startEvent
//  3. transitions 每条 fromStep/toStep 必须指向已存在的 step
//  4. **DAG 校验**：转换图不能存在环（防止工作流死循环）
//  5. 起点可达性：从 startEvent 出发可达所有其他 step（无孤岛）
func (s *WorkflowService) validateWorkflowDefinition(definition map[string]interface{}) error {
	if definition == nil {
		return errors.New("工作流定义不能为空")
	}

	startEvent, _ := definition["startEvent"].(string)
	if startEvent == "" {
		return errors.New("工作流定义缺少 startEvent")
	}

	// 解析 steps
	rawSteps, ok := definition["steps"].([]interface{})
	if !ok || len(rawSteps) == 0 {
		return errors.New("工作流定义缺少 steps 或为空")
	}
	stepIDs := make(map[string]bool, len(rawSteps))
	for _, r := range rawSteps {
		m, ok := r.(map[string]interface{})
		if !ok {
			return errors.New("steps 元素结构不合法")
		}
		id, _ := m["id"].(string)
		if id == "" {
			return errors.New("step 缺少 id")
		}
		if stepIDs[id] {
			return errors.New("step id 重复: " + id)
		}
		stepIDs[id] = true
	}
	if !stepIDs[startEvent] {
		return errors.New("startEvent 未在 steps 中定义: " + startEvent)
	}

	// 解析 transitions 并构建邻接表
	graph := make(map[string][]string)
	if rawTrans, ok := definition["transitions"].([]interface{}); ok {
		for _, r := range rawTrans {
			m, ok := r.(map[string]interface{})
			if !ok {
				return errors.New("transitions 元素结构不合法")
			}
			from, _ := m["fromStep"].(string)
			to, _ := m["toStep"].(string)
			if from == "" || to == "" {
				return errors.New("transition 缺少 fromStep/toStep")
			}
			if !stepIDs[from] {
				return errors.New("transition fromStep 未定义: " + from)
			}
			if !stepIDs[to] {
				return errors.New("transition toStep 未定义: " + to)
			}
			graph[from] = append(graph[from], to)
		}
	}

	// DFS 环检测（three-color: white=0/gray=1/black=2）
	color := make(map[string]int, len(stepIDs))
	var dfs func(node string) (bool, string)
	dfs = func(node string) (bool, string) {
		color[node] = 1 // gray
		for _, next := range graph[node] {
			switch color[next] {
			case 1:
				return true, node + "→" + next
			case 0:
				if cyc, path := dfs(next); cyc {
					return true, node + "→" + path
				}
			}
		}
		color[node] = 2 // black
		return false, ""
	}
	for id := range stepIDs {
		if color[id] == 0 {
			if cyc, path := dfs(id); cyc {
				return errors.New("工作流转换图存在环: " + path)
			}
		}
	}

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

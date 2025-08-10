package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
)

// ProcessEngine BPMN流程引擎核心接口
type ProcessEngine interface {
	// 流程定义管理
	ProcessDefinitionService() ProcessDefinitionService
	// 流程实例管理
	ProcessInstanceService() ProcessInstanceService
	// 任务管理
	TaskService() TaskService
	// 流程执行
	StartProcess(ctx context.Context, processDefinitionKey string, businessKey string, variables map[string]interface{}) (*ent.ProcessInstance, error)
	CompleteTask(ctx context.Context, taskID string, variables map[string]interface{}) error
	SuspendProcess(ctx context.Context, processInstanceID string, reason string) error
	ResumeProcess(ctx context.Context, processInstanceID string) error
	TerminateProcess(ctx context.Context, processInstanceID string, reason string) error
}

// ProcessDefinitionService 流程定义服务接口
type ProcessDefinitionService interface {
	// 创建流程定义
	CreateProcessDefinition(ctx context.Context, req *CreateProcessDefinitionRequest) (*ent.ProcessDefinition, error)
	// 获取流程定义
	GetProcessDefinition(ctx context.Context, key string, version string) (*ent.ProcessDefinition, error)
	// 获取最新版本的流程定义
	GetLatestProcessDefinition(ctx context.Context, key string) (*ent.ProcessDefinition, error)
	// 更新流程定义
	UpdateProcessDefinition(ctx context.Context, key string, version string, req *UpdateProcessDefinitionRequest) (*ent.ProcessDefinition, error)
	// 删除流程定义
	DeleteProcessDefinition(ctx context.Context, key string, version string) error
	// 获取流程定义列表
	ListProcessDefinitions(ctx context.Context, req *ListProcessDefinitionsRequest) ([]*ent.ProcessDefinition, int, error)
	// 激活/停用流程定义
	SetProcessDefinitionActive(ctx context.Context, key string, version string, active bool) error
}

// ProcessInstanceService 流程实例服务接口
type ProcessInstanceService interface {
	// 获取流程实例
	GetProcessInstance(ctx context.Context, processInstanceID string) (*ent.ProcessInstance, error)
	// 获取流程实例列表
	ListProcessInstances(ctx context.Context, req *ListProcessInstancesRequest) ([]*ent.ProcessInstance, int, error)
	// 获取流程实例变量
	GetProcessInstanceVariables(ctx context.Context, processInstanceID string) (map[string]interface{}, error)
	// 设置流程实例变量
	SetProcessInstanceVariables(ctx context.Context, processInstanceID string, variables map[string]interface{}) error
	// 获取流程实例历史
	GetProcessInstanceHistory(ctx context.Context, processInstanceID string) ([]*ent.ProcessExecutionHistory, error)
}

// TaskService 任务管理服务接口
type TaskService interface {
	// 获取任务
	GetTask(ctx context.Context, taskID string) (*ent.ProcessTask, error)
	// 获取用户任务列表
	ListUserTasks(ctx context.Context, req *ListUserTasksRequest) ([]*ent.ProcessTask, int, error)
	// 分配任务
	AssignTask(ctx context.Context, taskID string, assignee string) error
	// 完成任务
	CompleteTask(ctx context.Context, taskID string, variables map[string]interface{}) error
	// 取消任务
	CancelTask(ctx context.Context, taskID string, reason string) error
	// 获取任务变量
	GetTaskVariables(ctx context.Context, taskID string) (map[string]interface{}, error)
	// 设置任务变量
	SetTaskVariables(ctx context.Context, taskID string, variables map[string]interface{}) error
	// 任务超时处理
	HandleTaskTimeout(ctx context.Context, taskID string) error
	// 任务重试
	RetryTask(ctx context.Context, taskID string, maxRetries int) error
	// 任务委托
	DelegateTask(ctx context.Context, taskID string, newAssignee string) error
	// 任务升级
	EscalateTask(ctx context.Context, taskID string, reason string) error
	// 批量分配任务
	BatchAssignTasks(ctx context.Context, taskIDs []string, assignee string) error
	// 获取任务统计信息
	GetTaskStatistics(ctx context.Context, req *TaskStatisticsRequest) (*TaskStatistics, error)
}

// BPMNProcessEngine BPMN流程引擎实现
type BPMNProcessEngine struct {
	client *ent.Client
}

// NewBPMNProcessEngine 创建BPMN流程引擎实例
func NewBPMNProcessEngine(client *ent.Client) ProcessEngine {
	return &BPMNProcessEngine{client: client}
}

// ProcessDefinitionService 返回流程定义服务
func (e *BPMNProcessEngine) ProcessDefinitionService() ProcessDefinitionService {
	return &bpmnProcessDefinitionService{client: e.client}
}

// ProcessInstanceService 返回流程实例服务
func (e *BPMNProcessEngine) ProcessInstanceService() ProcessInstanceService {
	return &bpmnProcessInstanceService{client: e.client}
}

// TaskService 返回任务服务
func (e *BPMNProcessEngine) TaskService() TaskService {
	return &bpmnTaskService{client: e.client}
}

// StartProcess 启动流程实例
func (e *BPMNProcessEngine) StartProcess(ctx context.Context, processDefinitionKey string, businessKey string, variables map[string]interface{}) (*ent.ProcessInstance, error) {
	// 获取流程定义
	definition, err := e.client.ProcessDefinition.Query().
		Where(processdefinition.Key(processDefinitionKey)).
		Where(processdefinition.IsActive(true)).
		Where(processdefinition.IsLatest(true)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	// 序列化变量（用于存储）
	_, err = json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("序列化变量失败: %w", err)
	}

	// 创建流程实例
	instance, err := e.client.ProcessInstance.Create().
		SetProcessInstanceID(fmt.Sprintf("PI-%s-%d", processDefinitionKey, time.Now().Unix())).
		SetBusinessKey(businessKey).
		SetProcessDefinitionKey(processDefinitionKey).
		SetProcessDefinitionID(fmt.Sprintf("%d", definition.ID)).
		SetStatus("running").
		SetVariables(variables).
		SetStartTime(time.Now()).
		SetTenantID(definition.TenantID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建流程实例失败: %w", err)
	}

	return instance, nil
}

// CompleteTask 完成任务
func (e *BPMNProcessEngine) CompleteTask(ctx context.Context, taskID string, variables map[string]interface{}) error {
	// 获取任务
	task, err := e.client.ProcessTask.Query().
		Where(processtask.TaskID(taskID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 更新任务状态
	_, err = e.client.ProcessTask.UpdateOne(task).
		SetStatus("completed").
		SetCompletedTime(time.Now()).
		SetTaskVariables(variables).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// TODO: 实现流程继续逻辑
	// 这里需要根据BPMN定义确定下一步活动

	return nil
}

// SuspendProcess 暂停流程实例
func (e *BPMNProcessEngine) SuspendProcess(ctx context.Context, processInstanceID string, reason string) error {
	_, err := e.client.ProcessInstance.Update().
		Where(processinstance.ProcessInstanceID(processInstanceID)).
		SetStatus("suspended").
		SetSuspendedTime(time.Now()).
		SetSuspendedReason(reason).
		Save(ctx)
	return err
}

// ResumeProcess 恢复流程实例
func (e *BPMNProcessEngine) ResumeProcess(ctx context.Context, processInstanceID string) error {
	_, err := e.client.ProcessInstance.Update().
		Where(processinstance.ProcessInstanceID(processInstanceID)).
		SetStatus("running").
		SetSuspendedTime(time.Time{}).
		SetSuspendedReason("").
		Save(ctx)
	return err
}

// TerminateProcess 终止流程实例
func (e *BPMNProcessEngine) TerminateProcess(ctx context.Context, processInstanceID string, reason string) error {
	_, err := e.client.ProcessInstance.Update().
		Where(processinstance.ProcessInstanceID(processInstanceID)).
		SetStatus("terminated").
		SetEndTime(time.Now()).
		Save(ctx)
	return err
}

// 请求和响应结构体
type CreateProcessDefinitionRequest struct {
	Key              string                 `json:"key" binding:"required"`
	Name             string                 `json:"name" binding:"required"`
	Description      string                 `json:"description"`
	Category         string                 `json:"category"`
	BPMNXML          []byte                 `json:"bpmn_xml" binding:"required"`
	ProcessVariables map[string]interface{} `json:"process_variables"`
	TenantID         int                    `json:"tenant_id" binding:"required"`
}

type UpdateProcessDefinitionRequest struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Category         string                 `json:"category"`
	BPMNXML          []byte                 `json:"bpmn_xml"`
	ProcessVariables map[string]interface{} `json:"process_variables"`
	IsActive         *bool                  `json:"is_active"`
}

type ListProcessDefinitionsRequest struct {
	Key      string `json:"key"`
	Category string `json:"category"`
	IsActive *bool  `json:"is_active"`
	TenantID int    `json:"tenant_id"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type ListProcessInstancesRequest struct {
	ProcessDefinitionKey string `json:"process_definition_key"`
	Status               string `json:"status"`
	BusinessKey          string `json:"business_key"`
	TenantID             int    `json:"tenant_id"`
	Page                 int    `json:"page"`
	PageSize             int    `json:"page_size"`
}

type ListUserTasksRequest struct {
	Assignee             string `json:"assignee"`
	CandidateUsers       string `json:"candidate_users"`
	CandidateGroups      string `json:"candidate_groups"`
	Status               string `json:"status"`
	ProcessDefinitionKey string `json:"process_definition_key"`
	TenantID             int    `json:"tenant_id"`
	Page                 int    `json:"page"`
	PageSize             int    `json:"page_size"`
}

// bpmnProcessDefinitionService 流程定义服务实现
type bpmnProcessDefinitionService struct {
	client *ent.Client
}

// CreateProcessDefinition 创建流程定义
func (s *bpmnProcessDefinitionService) CreateProcessDefinition(ctx context.Context, req *CreateProcessDefinitionRequest) (*ent.ProcessDefinition, error) {
	// 检查key是否已存在
	existing, err := s.client.ProcessDefinition.Query().
		Where(processdefinition.Key(req.Key)).
		Where(processdefinition.IsLatest(true)).
		First(ctx)

	if err == nil && existing != nil {
		// 如果存在，将旧版本标记为非最新
		_, err = s.client.ProcessDefinition.UpdateOne(existing).
			SetIsLatest(false).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("更新旧版本失败: %w", err)
		}
	}

	// 创建新版本
	definition, err := s.client.ProcessDefinition.Create().
		SetKey(req.Key).
		SetName(req.Name).
		SetDescription(req.Description).
		SetCategory(req.Category).
		SetBpmnXML(req.BPMNXML).
		SetProcessVariables(req.ProcessVariables).
		SetVersion("1.0.0"). // TODO: 实现版本管理
		SetIsActive(true).
		SetIsLatest(true).
		SetTenantID(req.TenantID).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("创建流程定义失败: %w", err)
	}

	return definition, nil
}

// GetProcessDefinition 获取流程定义
func (s *bpmnProcessDefinitionService) GetProcessDefinition(ctx context.Context, key string, version string) (*ent.ProcessDefinition, error) {
	definition, err := s.client.ProcessDefinition.Query().
		Where(processdefinition.Key(key)).
		Where(processdefinition.Version(version)).
		First(ctx)

	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	return definition, nil
}

// GetLatestProcessDefinition 获取最新版本的流程定义
func (s *bpmnProcessDefinitionService) GetLatestProcessDefinition(ctx context.Context, key string) (*ent.ProcessDefinition, error) {
	definition, err := s.client.ProcessDefinition.Query().
		Where(processdefinition.Key(key)).
		Where(processdefinition.IsLatest(true)).
		First(ctx)

	if err != nil {
		return nil, fmt.Errorf("获取最新流程定义失败: %w", err)
	}

	return definition, nil
}

// UpdateProcessDefinition 更新流程定义
func (s *bpmnProcessDefinitionService) UpdateProcessDefinition(ctx context.Context, key string, version string, req *UpdateProcessDefinitionRequest) (*ent.ProcessDefinition, error) {
	definition, err := s.GetProcessDefinition(ctx, key, version)
	if err != nil {
		return nil, err
	}

	update := s.client.ProcessDefinition.UpdateOne(definition)

	if req.Name != "" {
		update.SetName(req.Name)
	}
	if req.Description != "" {
		update.SetDescription(req.Description)
	}
	if req.Category != "" {
		update.SetCategory(req.Category)
	}
	if req.BPMNXML != nil {
		update.SetBpmnXML(req.BPMNXML)
	}
	if req.ProcessVariables != nil {
		update.SetProcessVariables(req.ProcessVariables)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新流程定义失败: %w", err)
	}

	return updated, nil
}

// DeleteProcessDefinition 删除流程定义
func (s *bpmnProcessDefinitionService) DeleteProcessDefinition(ctx context.Context, key string, version string) error {
	definition, err := s.GetProcessDefinition(ctx, key, version)
	if err != nil {
		return err
	}

	return s.client.ProcessDefinition.DeleteOne(definition).Exec(ctx)
}

// ListProcessDefinitions 获取流程定义列表
func (s *bpmnProcessDefinitionService) ListProcessDefinitions(ctx context.Context, req *ListProcessDefinitionsRequest) ([]*ent.ProcessDefinition, int, error) {
	query := s.client.ProcessDefinition.Query()

	if req.Key != "" {
		query = query.Where(processdefinition.Key(req.Key))
	}
	if req.Category != "" {
		query = query.Where(processdefinition.Category(req.Category))
	}
	if req.IsActive != nil {
		query = query.Where(processdefinition.IsActive(*req.IsActive))
	}
	if req.TenantID > 0 {
		query = query.Where(processdefinition.TenantID(req.TenantID))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取流程定义总数失败: %w", err)
	}

	// 分页
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	definitions, err := query.Order(ent.Desc(processdefinition.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取流程定义列表失败: %w", err)
	}

	return definitions, total, nil
}

// SetProcessDefinitionActive 激活/停用流程定义
func (s *bpmnProcessDefinitionService) SetProcessDefinitionActive(ctx context.Context, key string, version string, active bool) error {
	definition, err := s.GetProcessDefinition(ctx, key, version)
	if err != nil {
		return err
	}

	_, err = s.client.ProcessDefinition.UpdateOne(definition).
		SetIsActive(active).
		Save(ctx)

	return err
}

// bpmnProcessInstanceService 流程实例服务实现
type bpmnProcessInstanceService struct {
	client *ent.Client
}

// GetProcessInstance 获取流程实例
func (s *bpmnProcessInstanceService) GetProcessInstance(ctx context.Context, processInstanceID string) (*ent.ProcessInstance, error) {
	instance, err := s.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(processInstanceID)).
		First(ctx)

	if err != nil {
		return nil, fmt.Errorf("获取流程实例失败: %w", err)
	}

	return instance, nil
}

// ListProcessInstances 获取流程实例列表
func (s *bpmnProcessInstanceService) ListProcessInstances(ctx context.Context, req *ListProcessInstancesRequest) ([]*ent.ProcessInstance, int, error) {
	query := s.client.ProcessInstance.Query()

	if req.ProcessDefinitionKey != "" {
		query = query.Where(processinstance.ProcessDefinitionKey(req.ProcessDefinitionKey))
	}
	if req.Status != "" {
		query = query.Where(processinstance.Status(req.Status))
	}
	if req.BusinessKey != "" {
		query = query.Where(processinstance.BusinessKey(req.BusinessKey))
	}
	if req.TenantID > 0 {
		query = query.Where(processinstance.TenantID(req.TenantID))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取流程实例总数失败: %w", err)
	}

	// 分页
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	instances, err := query.Order(ent.Desc(processinstance.FieldStartTime)).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取流程实例列表失败: %w", err)
	}

	return instances, total, nil
}

// GetProcessInstanceVariables 获取流程实例变量
func (s *bpmnProcessInstanceService) GetProcessInstanceVariables(ctx context.Context, processInstanceID string) (map[string]interface{}, error) {
	instance, err := s.GetProcessInstance(ctx, processInstanceID)
	if err != nil {
		return nil, err
	}

	return instance.Variables, nil
}

// SetProcessInstanceVariables 设置流程实例变量
func (s *bpmnProcessInstanceService) SetProcessInstanceVariables(ctx context.Context, processInstanceID string, variables map[string]interface{}) error {
	instance, err := s.GetProcessInstance(ctx, processInstanceID)
	if err != nil {
		return err
	}

	_, err = s.client.ProcessInstance.UpdateOne(instance).
		SetVariables(variables).
		Save(ctx)

	return err
}

// GetProcessInstanceHistory 获取流程实例历史
func (s *bpmnProcessInstanceService) GetProcessInstanceHistory(ctx context.Context, processInstanceID string) ([]*ent.ProcessExecutionHistory, error) {
	// TODO: 实现从ProcessExecutionHistory表获取历史记录
	// 这里需要根据实际的表结构来实现
	return []*ent.ProcessExecutionHistory{}, nil
}

// bpmnTaskService 任务服务实现
type bpmnTaskService struct {
	client *ent.Client
}

// GetTask 获取任务
func (s *bpmnTaskService) GetTask(ctx context.Context, taskID string) (*ent.ProcessTask, error) {
	task, err := s.client.ProcessTask.Query().
		Where(processtask.TaskID(taskID)).
		First(ctx)

	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %w", err)
	}

	return task, nil
}

// ListUserTasks 获取用户任务列表
func (s *bpmnTaskService) ListUserTasks(ctx context.Context, req *ListUserTasksRequest) ([]*ent.ProcessTask, int, error) {
	query := s.client.ProcessTask.Query()

	if req.Assignee != "" {
		query = query.Where(processtask.Assignee(req.Assignee))
	}
	if req.CandidateUsers != "" {
		query = query.Where(processtask.CandidateUsersContains(req.CandidateUsers))
	}
	if req.CandidateGroups != "" {
		query = query.Where(processtask.CandidateGroupsContains(req.CandidateGroups))
	}
	if req.Status != "" {
		query = query.Where(processtask.Status(req.Status))
	}
	if req.ProcessDefinitionKey != "" {
		query = query.Where(processtask.ProcessDefinitionKey(req.ProcessDefinitionKey))
	}
	if req.TenantID > 0 {
		query = query.Where(processtask.TenantID(req.TenantID))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取任务总数失败: %w", err)
	}

	// 分页
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	tasks, err := query.Order(ent.Desc(processtask.FieldCreatedTime)).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取任务列表失败: %w", err)
	}

	return tasks, total, nil
}

// AssignTask 分配任务
func (s *bpmnTaskService) AssignTask(ctx context.Context, taskID string, assignee string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	_, err = s.client.ProcessTask.UpdateOne(task).
		SetAssignee(assignee).
		SetStatus("assigned").
		SetAssignedTime(time.Now()).
		Save(ctx)

	return err
}

// CompleteTask 完成任务
func (s *bpmnTaskService) CompleteTask(ctx context.Context, taskID string, variables map[string]interface{}) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	_, err = s.client.ProcessTask.UpdateOne(task).
		SetStatus("completed").
		SetCompletedTime(time.Now()).
		SetTaskVariables(variables).
		Save(ctx)

	return err
}

// CancelTask 取消任务
func (s *bpmnTaskService) CancelTask(ctx context.Context, taskID string, reason string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	_, err = s.client.ProcessTask.UpdateOne(task).
		SetStatus("cancelled").
		Save(ctx)

	return err
}

// GetTaskVariables 获取任务变量
func (s *bpmnTaskService) GetTaskVariables(ctx context.Context, taskID string) (map[string]interface{}, error) {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return task.TaskVariables, nil
}

// SetTaskVariables 设置任务变量
func (s *bpmnTaskService) SetTaskVariables(ctx context.Context, taskID string, variables map[string]interface{}) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	_, err = s.client.ProcessTask.UpdateOne(task).
		SetTaskVariables(variables).
		Save(ctx)

	return err
}

// HandleTaskTimeout 处理任务超时
func (s *bpmnTaskService) HandleTaskTimeout(ctx context.Context, taskID string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	// 检查任务是否真的超时（使用due_date字段）
	if !task.DueDate.IsZero() && time.Now().After(task.DueDate) {
		_, err = s.client.ProcessTask.UpdateOne(task).
			SetStatus("timeout").
			Save(ctx)
		return err
	}

	return fmt.Errorf("任务未超时")
}

// RetryTask 重试任务
func (s *bpmnTaskService) RetryTask(ctx context.Context, taskID string, maxRetries int) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	// 由于ProcessTask没有RetryCount字段，我们使用task_variables来存储重试次数
	retryCount := 0
	if task.TaskVariables != nil {
		if count, exists := task.TaskVariables["retry_count"]; exists {
			if countInt, ok := count.(float64); ok {
				retryCount = int(countInt)
			}
		}
	}

	if retryCount >= maxRetries {
		return fmt.Errorf("任务重试次数已达上限: %d", maxRetries)
	}

	// 更新重试次数到变量中
	if task.TaskVariables == nil {
		task.TaskVariables = make(map[string]interface{})
	}
	task.TaskVariables["retry_count"] = retryCount + 1
	task.TaskVariables["last_retry_time"] = time.Now().Format(time.RFC3339)

	_, err = s.client.ProcessTask.UpdateOne(task).
		SetStatus("pending").
		SetTaskVariables(task.TaskVariables).
		Save(ctx)

	return err
}

// DelegateTask 委托任务
func (s *bpmnTaskService) DelegateTask(ctx context.Context, taskID string, newAssignee string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	// 记录原负责人到变量中
	if task.TaskVariables == nil {
		task.TaskVariables = make(map[string]interface{})
	}
	task.TaskVariables["delegated_from"] = task.Assignee
	task.TaskVariables["delegated_time"] = time.Now().Format(time.RFC3339)

	_, err = s.client.ProcessTask.UpdateOne(task).
		SetAssignee(newAssignee).
		SetStatus("delegated").
		SetTaskVariables(task.TaskVariables).
		Save(ctx)

	return err
}

// EscalateTask 升级任务
func (s *bpmnTaskService) EscalateTask(ctx context.Context, taskID string, reason string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	// 记录升级信息到变量中
	if task.TaskVariables == nil {
		task.TaskVariables = make(map[string]interface{})
	}
	task.TaskVariables["escalation_reason"] = reason
	task.TaskVariables["escalated_time"] = time.Now().Format(time.RFC3339)

	_, err = s.client.ProcessTask.UpdateOne(task).
		SetStatus("escalated").
		SetTaskVariables(task.TaskVariables).
		Save(ctx)

	return err
}

// BatchAssignTasks 批量分配任务
func (s *bpmnTaskService) BatchAssignTasks(ctx context.Context, taskIDs []string, assignee string) error {
	if len(taskIDs) == 0 {
		return fmt.Errorf("任务ID列表为空")
	}

	// 批量更新任务
	_, err := s.client.ProcessTask.Update().
		Where(processtask.TaskIDIn(taskIDs...)).
		SetAssignee(assignee).
		SetStatus("assigned").
		SetAssignedTime(time.Now()).
		Save(ctx)

	return err
}

// GetTaskStatistics 获取任务统计信息
func (s *bpmnTaskService) GetTaskStatistics(ctx context.Context, req *TaskStatisticsRequest) (*TaskStatistics, error) {
	query := s.client.ProcessTask.Query()

	// 应用过滤条件
	if req.ProcessDefinitionKey != "" {
		query = query.Where(processtask.ProcessDefinitionKey(req.ProcessDefinitionKey))
	}
	if req.Assignee != "" {
		query = query.Where(processtask.Assignee(req.Assignee))
	}
	if req.Status != "" {
		query = query.Where(processtask.Status(req.Status))
	}
	if req.TenantID > 0 {
		query = query.Where(processtask.TenantID(req.TenantID))
	}
	if req.StartDate != nil {
		query = query.Where(processtask.CreatedTimeGTE(*req.StartDate))
	}
	if req.EndDate != nil {
		query = query.Where(processtask.CreatedTimeLTE(*req.EndDate))
	}

	// 获取所有任务
	tasks, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取任务统计信息失败: %w", err)
	}

	// 计算统计信息
	stats := &TaskStatistics{
		TotalTasks:        len(tasks),
		StatusBreakdown:   make(map[string]int),
		AssigneeBreakdown: make(map[string]int),
		TimeDistribution:  make(map[string]interface{}),
	}

	var totalCompletionTime time.Duration
	completedCount := 0

	for _, task := range tasks {
		// 状态统计
		stats.StatusBreakdown[task.Status]++

		// 负责人统计
		if task.Assignee != "" {
			stats.AssigneeBreakdown[task.Assignee]++
		}

		// 完成时间统计
		if task.Status == "completed" && !task.CompletedTime.IsZero() && !task.AssignedTime.IsZero() {
			completionTime := task.CompletedTime.Sub(task.AssignedTime)
			totalCompletionTime += completionTime
			completedCount++
		}

		// 超时任务统计
		if !task.DueDate.IsZero() && time.Now().After(task.DueDate) && task.Status != "completed" {
			stats.OverdueTasks++
		}
	}

	// 计算平均完成时间
	if completedCount > 0 {
		stats.AverageCompletion = float64(totalCompletionTime.Milliseconds()) / float64(completedCount)
	}

	// 计算各状态任务数量
	stats.CompletedTasks = stats.StatusBreakdown["completed"]
	stats.PendingTasks = stats.StatusBreakdown["pending"] + stats.StatusBreakdown["assigned"]

	return stats, nil
}

// TaskStatisticsRequest 任务统计请求
type TaskStatisticsRequest struct {
	ProcessDefinitionKey string     `json:"process_definition_key"`
	Assignee             string     `json:"assignee"`
	Status               string     `json:"status"`
	TenantID             int        `json:"tenant_id"`
	StartDate            *time.Time `json:"start_date"`
	EndDate              *time.Time `json:"end_date"`
}

// TaskStatistics 任务统计信息
type TaskStatistics struct {
	TotalTasks        int                    `json:"total_tasks"`
	CompletedTasks    int                    `json:"completed_tasks"`
	PendingTasks      int                    `json:"pending_tasks"`
	OverdueTasks      int                    `json:"overdue_tasks"`
	AverageCompletion float64                `json:"average_completion"`
	StatusBreakdown   map[string]int         `json:"status_breakdown"`
	AssigneeBreakdown map[string]int         `json:"assignee_breakdown"`
	TimeDistribution  map[string]interface{} `json:"time_distribution"`
}

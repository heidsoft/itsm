package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
)

// WorkflowTaskService 工作流任务管理服务
type WorkflowTaskService struct {
	client *ent.Client
	engine *WorkflowEngine
}

// NewWorkflowTaskService 创建工作流任务管理服务实例
func NewWorkflowTaskService(client *ent.Client, engine *WorkflowEngine) *WorkflowTaskService {
	return &WorkflowTaskService{
		client: client,
		engine: engine,
	}
}

// WorkflowTask 工作流任务
type WorkflowTask struct {
	ID           int                    `json:"id"`
	InstanceID   int                    `json:"instance_id"`
	StepID       string                 `json:"step_id"`
	StepName     string                 `json:"step_name"`
	TaskType     string                 `json:"task_type"` // manual, automated, approval, notification
	AssigneeID   int                    `json:"assignee_id"`
	AssigneeName string                 `json:"assignee_name"`
	Status       string                 `json:"status"`   // pending, in_progress, completed, failed, cancelled
	Priority     string                 `json:"priority"` // low, medium, high, urgent
	DueDate      *time.Time             `json:"due_date"`
	StartedAt    *time.Time             `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Description  string                 `json:"description"`
	Instructions string                 `json:"instructions"`
	Data         map[string]interface{} `json:"data"`
	Result       map[string]interface{} `json:"result"`
	Error        string                 `json:"error"`
}

// CreateTask 创建任务
func (s *WorkflowTaskService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*WorkflowTask, error) {
	// 创建任务记录
	task := &WorkflowTask{
		InstanceID:   req.InstanceID,
		StepID:       req.StepID,
		StepName:     req.StepName,
		TaskType:     req.TaskType,
		AssigneeID:   req.AssigneeID,
		AssigneeName: req.AssigneeName,
		Status:       "pending",
		Priority:     req.Priority,
		DueDate:      req.DueDate,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Description:  req.Description,
		Instructions: req.Instructions,
		Data:         req.Data,
	}

	// 这里应该将任务保存到数据库
	// 暂时返回内存中的任务对象
	return task, nil
}

// GetTasks 获取任务列表
func (s *WorkflowTaskService) GetTasks(ctx context.Context, req *GetTasksRequest) ([]*WorkflowTask, int, error) {
	// 这里应该从数据库查询任务
	// 暂时返回空列表
	return []*WorkflowTask{}, 0, nil
}

// GetTask 获取任务详情
func (s *WorkflowTaskService) GetTask(ctx context.Context, taskID int) (*WorkflowTask, error) {
	// 这里应该从数据库查询任务
	// 暂时返回nil
	return nil, fmt.Errorf("任务不存在")
}

// StartTask 开始执行任务
func (s *WorkflowTaskService) StartTask(ctx context.Context, req *StartTaskRequest) error {
	// 获取任务
	task, err := s.GetTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 检查任务状态
	if task.Status != "pending" {
		return fmt.Errorf("任务状态不正确: %s", task.Status)
	}

	// 检查权限
	if task.AssigneeID != req.UserID {
		return fmt.Errorf("无权限执行此任务")
	}

	// 更新任务状态
	task.Status = "in_progress"
	task.StartedAt = &time.Time{}
	*task.StartedAt = time.Now()
	task.UpdatedAt = time.Now()

	// 这里应该更新数据库中的任务状态
	return nil
}

// CompleteTask 完成任务
func (s *WorkflowTaskService) CompleteTask(ctx context.Context, req *CompleteTaskRequest) error {
	// 获取任务
	task, err := s.GetTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 检查任务状态
	if task.Status != "in_progress" {
		return fmt.Errorf("任务状态不正确: %s", task.Status)
	}

	// 检查权限
	if task.AssigneeID != req.UserID {
		return fmt.Errorf("无权限完成此任务")
	}

	// 更新任务状态
	task.Status = "completed"
	task.CompletedAt = &time.Time{}
	*task.CompletedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.Result = req.Result

	// 完成工作流步骤
	completeReq := &CompleteWorkflowStepRequest{
		InstanceID: task.InstanceID,
		Action:     "complete",
		Variables:  req.Variables,
		Data: map[string]interface{}{
			"task_id": task.ID,
			"step_id": task.StepID,
			"result":  req.Result,
		},
		Comment: req.Comment,
		UserID:  req.UserID,
	}

	return s.engine.CompleteWorkflowStep(ctx, completeReq)
}

// FailTask 任务执行失败
func (s *WorkflowTaskService) FailTask(ctx context.Context, req *FailTaskRequest) error {
	// 获取任务
	task, err := s.GetTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 检查任务状态
	if task.Status != "in_progress" {
		return fmt.Errorf("任务状态不正确: %s", task.Status)
	}

	// 检查权限
	if task.AssigneeID != req.UserID {
		return fmt.Errorf("无权限操作此任务")
	}

	// 更新任务状态
	task.Status = "failed"
	task.CompletedAt = &time.Time{}
	*task.CompletedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.Error = req.Error

	// 这里应该更新数据库中的任务状态
	return nil
}

// CancelTask 取消任务
func (s *WorkflowTaskService) CancelTask(ctx context.Context, req *CancelWorkflowTaskRequest) error {
	// 获取任务
	task, err := s.GetTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 检查任务状态
	if task.Status != "pending" && task.Status != "in_progress" {
		return fmt.Errorf("任务状态不正确: %s", task.Status)
	}

	// 检查权限（只有任务分配者或管理员可以取消）
	if task.AssigneeID != req.UserID && !req.IsAdmin {
		return fmt.Errorf("无权限取消此任务")
	}

	// 更新任务状态
	task.Status = "cancelled"
	task.UpdatedAt = time.Now()

	// 这里应该更新数据库中的任务状态
	return nil
}

// ReassignTask 重新分配任务
func (s *WorkflowTaskService) ReassignTask(ctx context.Context, req *ReassignWorkflowTaskRequest) error {
	// 获取任务
	task, err := s.GetTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 检查任务状态
	if task.Status != "pending" && task.Status != "in_progress" {
		return fmt.Errorf("任务状态不正确: %s", task.Status)
	}

	// 检查权限（只有任务分配者或管理员可以重新分配）
	if task.AssigneeID != req.UserID && !req.IsAdmin {
		return fmt.Errorf("无权限重新分配此任务")
	}

	// 更新任务分配
	task.AssigneeID = req.NewAssigneeID
	task.AssigneeName = req.NewAssigneeName
	task.UpdatedAt = time.Now()

	// 这里应该更新数据库中的任务状态
	return nil
}

// GetTaskMetrics 获取任务指标
func (s *WorkflowTaskService) GetTaskMetrics(ctx context.Context, req *GetTaskMetricsRequest) (*WorkflowTaskMetrics, error) {
	// 这里应该从数据库查询任务指标
	// 暂时返回空的指标对象
	return &WorkflowTaskMetrics{}, nil
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	InstanceID   int                    `json:"instance_id" binding:"required"`
	StepID       string                 `json:"step_id" binding:"required"`
	StepName     string                 `json:"step_name" binding:"required"`
	TaskType     string                 `json:"task_type" binding:"required"`
	AssigneeID   int                    `json:"assignee_id" binding:"required"`
	AssigneeName string                 `json:"assignee_name" binding:"required"`
	Priority     string                 `json:"priority"`
	DueDate      *time.Time             `json:"due_date"`
	Description  string                 `json:"description"`
	Instructions string                 `json:"instructions"`
	Data         map[string]interface{} `json:"data"`
}

// GetTasksRequest 获取任务请求
type GetTasksRequest struct {
	InstanceID int    `json:"instance_id"`
	AssigneeID int    `json:"assignee_id"`
	Status     string `json:"status"`
	Priority   string `json:"priority"`
	TaskType   string `json:"task_type"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
}

// StartTaskRequest 开始任务请求
type StartTaskRequest struct {
	TaskID int `json:"task_id" binding:"required"`
	UserID int `json:"user_id" binding:"required"`
}

// CompleteTaskRequest 完成任务请求
type CompleteTaskRequest struct {
	TaskID    int                    `json:"task_id" binding:"required"`
	UserID    int                    `json:"user_id" binding:"required"`
	Result    map[string]interface{} `json:"result"`
	Variables map[string]interface{} `json:"variables"`
	Comment   string                 `json:"comment"`
}

// FailTaskRequest 任务失败请求
type FailTaskRequest struct {
	TaskID int    `json:"task_id" binding:"required"`
	UserID int    `json:"user_id" binding:"required"`
	Error  string `json:"error"`
}

// CancelWorkflowTaskRequest 取消工作流任务请求
type CancelWorkflowTaskRequest struct {
	TaskID  int    `json:"task_id" binding:"required"`
	UserID  int    `json:"user_id" binding:"required"`
	IsAdmin bool   `json:"is_admin"`
	Comment string `json:"comment"`
}

// ReassignWorkflowTaskRequest 重新分配工作流任务请求
type ReassignWorkflowTaskRequest struct {
	TaskID          int    `json:"task_id" binding:"required"`
	UserID          int    `json:"user_id" binding:"required"`
	IsAdmin         bool   `json:"is_admin"`
	NewAssigneeID   int    `json:"new_assignee_id" binding:"required"`
	NewAssigneeName string `json:"new_assignee_name" binding:"required"`
	Comment         string `json:"comment"`
}

// GetTaskMetricsRequest 获取任务指标请求
type GetTaskMetricsRequest struct {
	InstanceID int       `json:"instance_id"`
	AssigneeID int       `json:"assignee_id"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
}

// WorkflowTaskMetrics 工作流任务指标
type WorkflowTaskMetrics struct {
	TotalTasks      int     `json:"total_tasks"`
	CompletedTasks  int     `json:"completed_tasks"`
	FailedTasks     int     `json:"failed_tasks"`
	CancelledTasks  int     `json:"cancelled_tasks"`
	CompletionRate  float64 `json:"completion_rate"`
	AverageDuration float64 `json:"average_duration"`
	OnTimeTasks     int     `json:"on_time_tasks"`
	OverdueTasks    int     `json:"overdue_tasks"`
}

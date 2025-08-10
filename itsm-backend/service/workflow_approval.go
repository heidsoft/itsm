package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
)

// WorkflowApprovalService 工作流审批服务
type WorkflowApprovalService struct {
	client *ent.Client
	engine *WorkflowEngine
}

// NewWorkflowApprovalService 创建工作流审批服务实例
func NewWorkflowApprovalService(client *ent.Client, engine *WorkflowEngine) *WorkflowApprovalService {
	return &WorkflowApprovalService{
		client: client,
		engine: engine,
	}
}

// ApprovalTask 审批任务
type ApprovalTask struct {
	ID           int                    `json:"id"`
	InstanceID   int                    `json:"instance_id"`
	StepID       string                 `json:"step_id"`
	StepName     string                 `json:"step_name"`
	AssigneeID   int                    `json:"assignee_id"`
	AssigneeName string                 `json:"assignee_name"`
	Status       string                 `json:"status"` // pending, approved, rejected, cancelled
	Priority     string                 `json:"priority"`
	DueDate      *time.Time             `json:"due_date"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	CompletedAt  *time.Time             `json:"completed_at"`
	Comment      string                 `json:"comment"`
	Data         map[string]interface{} `json:"data"`
}

// CreateApprovalTask 创建审批任务
func (s *WorkflowApprovalService) CreateApprovalTask(ctx context.Context, req *CreateApprovalTaskRequest) (*ApprovalTask, error) {
	// 创建工作流任务记录
	task := &ApprovalTask{
		InstanceID:   req.InstanceID,
		StepID:       req.StepID,
		StepName:     req.StepName,
		AssigneeID:   req.AssigneeID,
		AssigneeName: req.AssigneeName,
		Status:       "pending",
		Priority:     req.Priority,
		DueDate:      req.DueDate,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Data:         req.Data,
	}

	// 这里应该将任务保存到数据库
	// 暂时返回内存中的任务对象
	return task, nil
}

// GetApprovalTasks 获取审批任务列表
func (s *WorkflowApprovalService) GetApprovalTasks(ctx context.Context, req *GetApprovalTasksRequest) ([]*ApprovalTask, int, error) {
	// 这里应该从数据库查询审批任务
	// 暂时返回空列表
	return []*ApprovalTask{}, 0, nil
}

// GetApprovalTask 获取审批任务详情
func (s *WorkflowApprovalService) GetApprovalTask(ctx context.Context, taskID int) (*ApprovalTask, error) {
	// 这里应该从数据库查询审批任务
	// 暂时返回nil
	return nil, fmt.Errorf("审批任务不存在")
}

// ApproveTask 审批通过
func (s *WorkflowApprovalService) ApproveTask(ctx context.Context, req *ApproveTaskRequest) error {
	// 获取审批任务
	task, err := s.GetApprovalTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("获取审批任务失败: %w", err)
	}

	// 检查任务状态
	if task.Status != "pending" {
		return fmt.Errorf("任务状态不正确: %s", task.Status)
	}

	// 检查权限
	if task.AssigneeID != req.UserID {
		return fmt.Errorf("无权限审批此任务")
	}

	// 更新任务状态
	task.Status = "approved"
	task.Comment = req.Comment
	task.CompletedAt = &time.Time{}
	*task.CompletedAt = time.Now()
	task.UpdatedAt = time.Now()

	// 更新工作流实例变量
	variables := map[string]interface{}{
		"approval_status": "approved",
		"approval_comment": req.Comment,
		"approval_user_id": req.UserID,
		"approval_time": time.Now(),
	}

	// 完成工作流步骤
	completeReq := &CompleteWorkflowStepRequest{
		InstanceID: task.InstanceID,
		Action:     "approve",
		Variables:  variables,
		Data: map[string]interface{}{
			"task_id": task.ID,
			"step_id": task.StepID,
		},
		Comment: req.Comment,
		UserID:  req.UserID,
	}

	return s.engine.CompleteWorkflowStep(ctx, completeReq)
}

// RejectTask 审批拒绝
func (s *WorkflowApprovalService) RejectTask(ctx context.Context, req *RejectTaskRequest) error {
	// 获取审批任务
	task, err := s.GetApprovalTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("获取审批任务失败: %w", err)
	}

	// 检查任务状态
	if task.Status != "pending" {
		return fmt.Errorf("任务状态不正确: %s", task.Status)
	}

	// 检查权限
	if task.AssigneeID != req.UserID {
		return fmt.Errorf("无权限审批此任务")
	}

	// 更新任务状态
	task.Status = "rejected"
	task.Comment = req.Comment
	task.CompletedAt = &time.Time{}
	*task.CompletedAt = time.Now()
	task.UpdatedAt = time.Now()

	// 更新工作流实例变量
	variables := map[string]interface{}{
		"approval_status": "rejected",
		"approval_comment": req.Comment,
		"approval_user_id": req.UserID,
		"approval_time": time.Now(),
	}

	// 完成工作流步骤
	completeReq := &CompleteWorkflowStepRequest{
		InstanceID: task.InstanceID,
		Action:     "reject",
		Variables:  variables,
		Data: map[string]interface{}{
			"task_id": task.ID,
			"step_id": task.StepID,
		},
		Comment: req.Comment,
		UserID:  req.UserID,
	}

	return s.engine.CompleteWorkflowStep(ctx, completeReq)
}

// CancelTask 取消审批任务
func (s *WorkflowApprovalService) CancelTask(ctx context.Context, req *CancelTaskRequest) error {
	// 获取审批任务
	task, err := s.GetApprovalTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("获取审批任务失败: %w", err)
	}

	// 检查任务状态
	if task.Status != "pending" {
		return fmt.Errorf("任务状态不正确: %s", task.Status)
	}

	// 检查权限（只有任务分配者或管理员可以取消）
	if task.AssigneeID != req.UserID && !req.IsAdmin {
		return fmt.Errorf("无权限取消此任务")
	}

	// 更新任务状态
	task.Status = "cancelled"
	task.Comment = req.Comment
	task.UpdatedAt = time.Now()

	// 这里应该更新数据库中的任务状态
	return nil
}

// ReassignTask 重新分配审批任务
func (s *WorkflowApprovalService) ReassignTask(ctx context.Context, req *ReassignTaskRequest) error {
	// 获取审批任务
	task, err := s.GetApprovalTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("获取审批任务失败: %w", err)
	}

	// 检查任务状态
	if task.Status != "pending" {
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

// GetApprovalHistory 获取审批历史
func (s *WorkflowApprovalService) GetApprovalHistory(ctx context.Context, instanceID int) ([]*ApprovalTask, error) {
	// 这里应该从数据库查询审批历史
	// 暂时返回空列表
	return []*ApprovalTask{}, nil
}

// CreateApprovalTaskRequest 创建审批任务请求
type CreateApprovalTaskRequest struct {
	InstanceID   int                    `json:"instance_id" binding:"required"`
	StepID       string                 `json:"step_id" binding:"required"`
	StepName     string                 `json:"step_name" binding:"required"`
	AssigneeID   int                    `json:"assignee_id" binding:"required"`
	AssigneeName string                 `json:"assignee_name" binding:"required"`
	Priority     string                 `json:"priority"`
	DueDate      *time.Time             `json:"due_date"`
	Data         map[string]interface{} `json:"data"`
}

// GetApprovalTasksRequest 获取审批任务请求
type GetApprovalTasksRequest struct {
	AssigneeID int    `json:"assignee_id"`
	Status     string `json:"status"`
	Priority   string `json:"priority"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
}

// ApproveTaskRequest 审批通过请求
type ApproveTaskRequest struct {
	TaskID  int    `json:"task_id" binding:"required"`
	UserID  int    `json:"user_id" binding:"required"`
	Comment string `json:"comment"`
}

// RejectTaskRequest 审批拒绝请求
type RejectTaskRequest struct {
	TaskID  int    `json:"task_id" binding:"required"`
	UserID  int    `json:"user_id" binding:"required"`
	Comment string `json:"comment"`
}

// CancelTaskRequest 取消审批任务请求
type CancelTaskRequest struct {
	TaskID  int    `json:"task_id" binding:"required"`
	UserID  int    `json:"user_id" binding:"required"`
	IsAdmin bool   `json:"is_admin"`
	Comment string `json:"comment"`
}

// ReassignTaskRequest 重新分配审批任务请求
type ReassignTaskRequest struct {
	TaskID          int    `json:"task_id" binding:"required"`
	UserID          int    `json:"user_id" binding:"required"`
	IsAdmin         bool   `json:"is_admin"`
	NewAssigneeID   int    `json:"new_assignee_id" binding:"required"`
	NewAssigneeName string `json:"new_assignee_name" binding:"required"`
	Comment         string `json:"comment"`
}

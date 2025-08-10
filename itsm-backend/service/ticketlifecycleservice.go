package service

import (
	"context"
	"fmt"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/workflowinstance"
	"time"
)

// TicketLifecycleService 工单生命周期服务
type TicketLifecycleService struct {
	client *ent.Client
}

// NewTicketLifecycleService 创建工单生命周期服务实例
func NewTicketLifecycleService(client *ent.Client) *TicketLifecycleService {
	return &TicketLifecycleService{
		client: client,
	}
}

// TicketStatus 工单状态
const (
	StatusOpen       = "open"        // 已开启
	StatusInProgress = "in_progress" // 处理中
	StatusPending    = "pending"     // 待处理
	StatusResolved   = "resolved"    // 已解决
	StatusClosed     = "closed"      // 已关闭
	StatusCancelled  = "cancelled"   // 已取消
	StatusReopened   = "reopened"    // 重新开启
)

// TicketPriority 工单优先级
const (
	PriorityLow      = "low"      // 低
	PriorityMedium   = "medium"   // 中
	PriorityHigh     = "high"     // 高
	PriorityCritical = "critical" // 紧急
)

// StatusTransitionRequest 状态转换请求
type StatusTransitionRequest struct {
	TicketID     int    `json:"ticket_id"`
	NewStatus    string `json:"new_status"`
	Comment      string `json:"comment,omitempty"`
	AssignedTo   *int   `json:"assigned_to,omitempty"`
	Priority     string `json:"priority,omitempty"`
	TenantID     int    `json:"tenant_id"`
	UserID       int    `json:"user_id"`
	WorkflowStep string `json:"workflow_step,omitempty"`
}

// StatusTransitionResponse 状态转换响应
type StatusTransitionResponse struct {
	TicketID     int       `json:"ticket_id"`
	OldStatus    string    `json:"old_status"`
	NewStatus    string    `json:"new_status"`
	TransitionAt time.Time `json:"transition_at"`
	Comment      string    `json:"comment,omitempty"`
	AssignedTo   *int      `json:"assigned_to,omitempty"`
	Priority     string    `json:"priority,omitempty"`
	WorkflowStep string    `json:"workflow_step,omitempty"`
	IsValid      bool      `json:"is_valid"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// TicketLifecycleInfo 工单生命周期信息
type TicketLifecycleInfo struct {
	TicketID        int                `json:"ticket_id"`
	CurrentStatus   string             `json:"current_status"`
	CurrentPriority string             `json:"current_priority"`
	AssignedTo      *int               `json:"assigned_to,omitempty"`
	StatusHistory   []StatusTransition `json:"status_history"`
	WorkflowStatus  *WorkflowStatus    `json:"workflow_status,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	TotalDuration   time.Duration      `json:"total_duration"`
	SLAMetrics      *TicketSLAMetrics  `json:"sla_metrics,omitempty"`
}

// StatusTransition 状态转换记录
type StatusTransition struct {
	FromStatus   string        `json:"from_status"`
	ToStatus     string        `json:"to_status"`
	TransitionAt time.Time     `json:"transition_at"`
	Comment      string        `json:"comment,omitempty"`
	UserID       int           `json:"user_id"`
	WorkflowStep string        `json:"workflow_step,omitempty"`
	Duration     time.Duration `json:"duration,omitempty"`
}

// WorkflowStatus 工作流状态
type WorkflowStatus struct {
	WorkflowID    int           `json:"workflow_id"`
	CurrentStep   string        `json:"current_step"`
	Status        string        `json:"status"`
	Progress      int           `json:"progress"` // 0-100
	NextStep      string        `json:"next_step,omitempty"`
	EstimatedTime time.Duration `json:"estimated_time,omitempty"`
}

// TicketSLAMetrics 工单SLA指标
type TicketSLAMetrics struct {
	ResponseTime   time.Duration `json:"response_time,omitempty"`
	ResolutionTime time.Duration `json:"resolution_time,omitempty"`
	IsWithinSLA    bool          `json:"is_within_sla"`
	SLABreachAt    *time.Time    `json:"sla_breach_at,omitempty"`
	RemainingTime  time.Duration `json:"remaining_time,omitempty"`
}

// TransitionRules 状态转换规则
var TransitionRules = map[string][]string{
	StatusOpen: {
		StatusInProgress,
		StatusPending,
		StatusCancelled,
	},
	StatusInProgress: {
		StatusPending,
		StatusResolved,
		StatusCancelled,
	},
	StatusPending: {
		StatusInProgress,
		StatusResolved,
		StatusCancelled,
	},
	StatusResolved: {
		StatusClosed,
		StatusReopened,
	},
	StatusClosed: {
		StatusReopened,
	},
	StatusReopened: {
		StatusInProgress,
		StatusPending,
	},
	StatusCancelled: {
		StatusOpen,
	},
}

// ValidateStatusTransition 验证状态转换是否有效
func (s *TicketLifecycleService) ValidateStatusTransition(fromStatus, toStatus string) bool {
	allowedTransitions, exists := TransitionRules[fromStatus]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == toStatus {
			return true
		}
	}
	return false
}

// TransitionTicketStatus 转换工单状态
func (s *TicketLifecycleService) TransitionTicketStatus(ctx context.Context, req *StatusTransitionRequest) (*StatusTransitionResponse, error) {
	// 获取当前工单
	ticketEntity, err := s.client.Ticket.Query().
		Where(ticket.ID(req.TicketID), ticket.TenantID(req.TenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	oldStatus := ticketEntity.Status

	// 验证状态转换
	if !s.ValidateStatusTransition(oldStatus, req.NewStatus) {
		return &StatusTransitionResponse{
			TicketID:     req.TicketID,
			OldStatus:    oldStatus,
			NewStatus:    req.NewStatus,
			IsValid:      false,
			ErrorMessage: fmt.Sprintf("不允许从状态 %s 转换到 %s", oldStatus, req.NewStatus),
		}, nil
	}

	// 更新工单状态
	update := s.client.Ticket.UpdateOneID(req.TicketID).
		SetStatus(req.NewStatus).
		SetUpdatedAt(time.Now())

	if req.Priority != "" {
		update.SetPriority(req.Priority)
	}
	if req.AssignedTo != nil {
		update.SetAssigneeID(*req.AssignedTo)
	}

	_, err = update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新工单状态失败: %w", err)
	}

	// 记录状态转换历史
	err = s.recordStatusTransition(ctx, req.TicketID, oldStatus, req.NewStatus, req.Comment, req.UserID, req.WorkflowStep)
	if err != nil {
		// 记录失败不影响状态转换
		fmt.Printf("记录状态转换历史失败: %v\n", err)
	}

	// 更新工作流实例状态
	if req.WorkflowStep != "" {
		err = s.updateWorkflowInstance(ctx, req.TicketID, req.NewStatus, req.WorkflowStep)
		if err != nil {
			fmt.Printf("更新工作流实例失败: %v\n", err)
		}
	}

	// 检查是否需要自动分配
	if req.NewStatus == StatusOpen && req.AssignedTo == nil {
		err = s.autoAssignTicket(ctx, req.TicketID, req.TenantID)
		if err != nil {
			fmt.Printf("自动分配工单失败: %v\n", err)
		}
	}

	return &StatusTransitionResponse{
		TicketID:     req.TicketID,
		OldStatus:    oldStatus,
		NewStatus:    req.NewStatus,
		TransitionAt: time.Now(),
		Comment:      req.Comment,
		AssignedTo:   req.AssignedTo,
		Priority:     req.Priority,
		WorkflowStep: req.WorkflowStep,
		IsValid:      true,
	}, nil
}

// GetTicketLifecycleInfo 获取工单生命周期信息
func (s *TicketLifecycleService) GetTicketLifecycleInfo(ctx context.Context, ticketID, tenantID int) (*TicketLifecycleInfo, error) {
	// 获取工单基本信息
	ticketEntity, err := s.client.Ticket.Query().
		Where(ticket.ID(ticketID), ticket.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	// 获取状态转换历史
	statusHistory, err := s.getStatusTransitionHistory(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("获取状态转换历史失败: %w", err)
	}

	// 获取工作流状态
	workflowStatus, err := s.getWorkflowStatus(ctx, ticketID)
	if err != nil {
		// 工作流状态获取失败不影响主要功能
		fmt.Printf("获取工作流状态失败: %v\n", err)
	}

	// 计算总持续时间
	totalDuration := time.Since(ticketEntity.CreatedAt)

	// 计算SLA指标
	slaMetrics, err := s.calculateSLAMetrics(ctx, ticketEntity)
	if err != nil {
		fmt.Printf("计算SLA指标失败: %v\n", err)
	}

	// 处理AssigneeID，如果是0则设为nil
	var assignedTo *int
	if ticketEntity.AssigneeID != 0 {
		assignedTo = &ticketEntity.AssigneeID
	}

	return &TicketLifecycleInfo{
		TicketID:        ticketID,
		CurrentStatus:   ticketEntity.Status,
		CurrentPriority: ticketEntity.Priority,
		AssignedTo:      assignedTo,
		StatusHistory:   statusHistory,
		WorkflowStatus:  workflowStatus,
		CreatedAt:       ticketEntity.CreatedAt,
		UpdatedAt:       ticketEntity.UpdatedAt,
		TotalDuration:   totalDuration,
		SLAMetrics:      slaMetrics,
	}, nil
}

// BulkStatusTransition 批量状态转换
func (s *TicketLifecycleService) BulkStatusTransition(ctx context.Context, reqs []StatusTransitionRequest) ([]StatusTransitionResponse, error) {
	var responses []StatusTransitionResponse

	for _, req := range reqs {
		response, err := s.TransitionTicketStatus(ctx, &req)
		if err != nil {
			response = &StatusTransitionResponse{
				TicketID:     req.TicketID,
				NewStatus:    req.NewStatus,
				IsValid:      false,
				ErrorMessage: err.Error(),
			}
		}
		responses = append(responses, *response)
	}

	return responses, nil
}

// GetTicketsByStatus 根据状态获取工单列表
func (s *TicketLifecycleService) GetTicketsByStatus(ctx context.Context, status string, tenantID int, limit, offset int) ([]*ent.Ticket, error) {
	query := s.client.Ticket.Query().
		Where(ticket.Status(status), ticket.TenantID(tenantID)).
		Order(ent.Desc(ticket.FieldCreatedAt)).
		Limit(limit).
		Offset(offset)

	return query.All(ctx)
}

// GetTicketsByPriority 根据优先级获取工单列表
func (s *TicketLifecycleService) GetTicketsByPriority(ctx context.Context, priority string, tenantID int, limit, offset int) ([]*ent.Ticket, error) {
	query := s.client.Ticket.Query().
		Where(ticket.Priority(priority), ticket.TenantID(tenantID)).
		Order(ent.Desc(ticket.FieldCreatedAt)).
		Limit(limit).
		Offset(offset)

	return query.All(ctx)
}

// GetTicketsByAssignee 根据处理人获取工单列表
func (s *TicketLifecycleService) GetTicketsByAssignee(ctx context.Context, assigneeID, tenantID int, limit, offset int) ([]*ent.Ticket, error) {
	query := s.client.Ticket.Query().
		Where(ticket.AssigneeID(assigneeID), ticket.TenantID(tenantID)).
		Order(ent.Desc(ticket.FieldCreatedAt)).
		Limit(limit).
		Offset(offset)

	return query.All(ctx)
}

// recordStatusTransition 记录状态转换历史
func (s *TicketLifecycleService) recordStatusTransition(ctx context.Context, ticketID int, fromStatus, toStatus, comment string, userID int, workflowStep string) error {
	// 这里可以记录到专门的审计日志表或状态转换历史表
	// 暂时使用简单的日志记录
	fmt.Printf("工单 %d 状态转换: %s -> %s, 用户: %d, 工作流步骤: %s, 备注: %s\n",
		ticketID, fromStatus, toStatus, userID, workflowStep, comment)
	return nil
}

// updateWorkflowInstance 更新工作流实例状态
func (s *TicketLifecycleService) updateWorkflowInstance(ctx context.Context, ticketID int, status, workflowStep string) error {
	// 查找相关的工作流实例
	instances, err := s.client.WorkflowInstance.Query().
		Where(workflowinstance.EntityID(ticketID), workflowinstance.EntityType("ticket")).
		All(ctx)
	if err != nil {
		return err
	}

	// 更新工作流实例状态
	for _, instance := range instances {
		_, err = instance.Update().
			SetStatus(status).
			SetCurrentStep(workflowStep).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// autoAssignTicket 自动分配工单
func (s *TicketLifecycleService) autoAssignTicket(ctx context.Context, ticketID, tenantID int) error {
	// 这里可以实现自动分配逻辑
	// 例如：根据工单类型、优先级、处理人工作负载等
	// 暂时返回nil，表示不进行自动分配
	return nil
}

// getStatusTransitionHistory 获取状态转换历史
func (s *TicketLifecycleService) getStatusTransitionHistory(ctx context.Context, ticketID int) ([]StatusTransition, error) {
	// 这里应该从专门的审计日志表或状态转换历史表获取
	// 暂时返回空列表
	return []StatusTransition{}, nil
}

// getWorkflowStatus 获取工作流状态
func (s *TicketLifecycleService) getWorkflowStatus(ctx context.Context, ticketID int) (*WorkflowStatus, error) {
	// 查找相关的工作流实例
	instances, err := s.client.WorkflowInstance.Query().
		Where(workflowinstance.EntityID(ticketID), workflowinstance.EntityType("ticket")).
		All(ctx)
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return nil, nil
	}

	// 使用第一个实例（通常一个工单只有一个工作流实例）
	instance := instances[0]

	return &WorkflowStatus{
		WorkflowID:    instance.WorkflowID,
		CurrentStep:   instance.CurrentStep,
		Status:        instance.Status,
		Progress:      0,  // 需要根据工作流定义计算
		NextStep:      "", // 需要根据工作流定义获取
		EstimatedTime: 0,  // 需要根据工作流定义计算
	}, nil
}

// calculateSLAMetrics 计算SLA指标
func (s *TicketLifecycleService) calculateSLAMetrics(ctx context.Context, ticketEntity *ent.Ticket) (*TicketSLAMetrics, error) {
	// 这里应该根据SLA定义计算指标
	// 暂时返回基本的指标
	now := time.Now()
	responseTime := now.Sub(ticketEntity.CreatedAt)

	var resolutionTime time.Duration
	if ticketEntity.Status == StatusResolved || ticketEntity.Status == StatusClosed {
		// 需要从状态转换历史中找到解决时间
		resolutionTime = responseTime
	}

	return &TicketSLAMetrics{
		ResponseTime:   responseTime,
		ResolutionTime: resolutionTime,
		IsWithinSLA:    true, // 需要根据SLA定义判断
		RemainingTime:  0,    // 需要根据SLA定义计算
	}, nil
}

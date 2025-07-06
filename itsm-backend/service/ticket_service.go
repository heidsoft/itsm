package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/approvallog"

	"go.uber.org/zap"
)

type TicketService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewTicketService(client *ent.Client, logger *zap.SugaredLogger) *TicketService {
	return &TicketService{
		client: client,
		logger: logger,
	}
}

// CreateTicket 创建工单
func (s *TicketService) CreateTicket(ctx context.Context, req *dto.CreateTicketRequest, requesterID int) (*ent.Ticket, error) {
	// 生成工单编号
	ticketNumber := s.generateTicketNumber()

	// 创建工单
	create := s.client.Ticket.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetPriority(ticket.Priority(req.Priority)).
		SetTicketNumber(ticketNumber).
		SetRequesterID(requesterID)

	if req.FormFields != nil {
		create = create.SetFormFields(req.FormFields)
	}

	if req.AssigneeID != nil {
		create = create.SetAssigneeID(*req.AssigneeID)
	}

	ticketEntity, err := create.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create ticket", "error", err)
		return nil, fmt.Errorf("创建工单失败: %w", err)
	}

	s.logger.Infow("Ticket created successfully", "ticket_id", ticketEntity.ID, "requester_id", requesterID)
	return ticketEntity, nil
}

// GetTicketByID 根据ID获取工单详情
func (s *TicketService) GetTicketByID(ctx context.Context, id int) (*ent.Ticket, error) {
	ticketEntity, err := s.client.Ticket.Query().
		Where(ticket.ID(id)).
		WithRequester().
		WithAssignee().
		WithApprovalLogs(func(q *ent.ApprovalLogQuery) {
			q.WithApprover().Order(ent.Asc(approvallog.FieldStepOrder))
		}).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("工单不存在")
		}
		s.logger.Errorw("Failed to get ticket", "ticket_id", id, "error", err)
		return nil, fmt.Errorf("获取工单失败: %w", err)
	}

	return ticketEntity, nil
}

// UpdateTicket 更新工单
func (s *TicketService) UpdateTicket(ctx context.Context, id int, req *dto.UpdateTicketRequest, userID int) (*ent.Ticket, error) {
	// 检查工单是否存在
	ticketEntity, err := s.GetTicketByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 检查权限（只有申请人或处理人可以更新）
	if ticketEntity.RequesterID != userID && (ticketEntity.AssigneeID == nil || *ticketEntity.AssigneeID != userID) {
		return nil, fmt.Errorf("无权限更新此工单")
	}

	update := s.client.Ticket.UpdateOneID(id)

	if req.Status != nil {
		update = update.SetStatus(ticket.Status(*req.Status))
	}

	if req.Priority != nil {
		update = update.SetPriority(ticket.Priority(*req.Priority))
	}

	if req.FormFields != nil {
		update = update.SetFormFields(req.FormFields)
	}

	if req.AssigneeID != nil {
		update = update.SetAssigneeID(*req.AssigneeID)
	}

	updatedTicket, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update ticket", "ticket_id", id, "error", err)
		return nil, fmt.Errorf("更新工单失败: %w", err)
	}

	s.logger.Infow("Ticket updated successfully", "ticket_id", id, "user_id", userID)
	return updatedTicket, nil
}

// ApproveTicket 审批工单
func (s *TicketService) ApproveTicket(ctx context.Context, ticketID int, req *dto.ApprovalRequest, approverID int) error {
	// 检查工单是否存在
	ticketEntity, err := s.GetTicketByID(ctx, ticketID)
	if err != nil {
		return err
	}

	// 检查工单状态是否可以审批
	if ticketEntity.Status != ticket.StatusPending {
		return fmt.Errorf("工单状态不允许审批")
	}

	// 获取当前审批步骤
	currentStep, err := s.getCurrentApprovalStep(ctx, ticketID)
	if err != nil {
		return err
	}

	// 创建审批记录
	approvalStatus := approvallog.StatusApproved
	if req.Action == "reject" {
		approvalStatus = approvallog.StatusRejected
	}

	now := time.Now()
	_, err = s.client.ApprovalLog.Create().
		SetApproverID(approverID).
		SetTicketID(ticketID).
		SetComment(req.Comment).
		SetStatus(approvalStatus).
		SetStepOrder(currentStep).
		SetStepName(req.StepName).
		SetApprovedAt(now).
		Save(ctx)

	if err != nil {
		s.logger.Errorw("Failed to create approval log", "ticket_id", ticketID, "error", err)
		return fmt.Errorf("创建审批记录失败: %w", err)
	}

	// 更新工单状态
	newStatus := ticket.StatusApproved
	if req.Action == "reject" {
		newStatus = ticket.StatusRejected
	}

	_, err = s.client.Ticket.UpdateOneID(ticketID).
		SetStatus(newStatus).
		Save(ctx)

	if err != nil {
		s.logger.Errorw("Failed to update ticket status", "ticket_id", ticketID, "error", err)
		return fmt.Errorf("更新工单状态失败: %w", err)
	}

	s.logger.Infow("Ticket approval completed", "ticket_id", ticketID, "action", req.Action, "approver_id", approverID)
	return nil
}

// AddComment 添加评论（通过审批记录实现）
func (s *TicketService) AddComment(ctx context.Context, ticketID int, req *dto.CommentRequest, userID int) error {
	// 检查工单是否存在
	_, err := s.GetTicketByID(ctx, ticketID)
	if err != nil {
		return err
	}

	// 获取下一个步骤序号
	nextStep, err := s.getNextCommentStep(ctx, ticketID)
	if err != nil {
		return err
	}

	// 创建评论记录（使用审批记录表）
	_, err = s.client.ApprovalLog.Create().
		SetApproverID(userID).
		SetTicketID(ticketID).
		SetComment(req.Content).
		SetStatus(approvallog.StatusSkipped). // 评论使用skipped状态
		SetStepOrder(nextStep).
		SetStepName("评论").
		Save(ctx)

	if err != nil {
		s.logger.Errorw("Failed to add comment", "ticket_id", ticketID, "error", err)
		return fmt.Errorf("添加评论失败: %w", err)
	}

	s.logger.Infow("Comment added successfully", "ticket_id", ticketID, "user_id", userID)
	return nil
}

// generateTicketNumber 生成工单编号
func (s *TicketService) generateTicketNumber() string {
	now := time.Now()
	return fmt.Sprintf("TK%s%06d", now.Format("20060102"), now.Unix()%1000000)
}

// getCurrentApprovalStep 获取当前审批步骤
func (s *TicketService) getCurrentApprovalStep(ctx context.Context, ticketID int) (int, error) {
	lastLog, err := s.client.ApprovalLog.Query().
		Where(approvallog.TicketID(ticketID)).
		Order(ent.Desc(approvallog.FieldStepOrder)).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return 1, nil // 第一步
		}
		return 0, err
	}

	return lastLog.StepOrder + 1, nil
}

// getNextCommentStep 获取下一个评论步骤序号
func (s *TicketService) getNextCommentStep(ctx context.Context, ticketID int) (int, error) {
	lastLog, err := s.client.ApprovalLog.Query().
		Where(approvallog.TicketID(ticketID)).
		Order(ent.Desc(approvallog.FieldStepOrder)).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return 1, nil
		}
		return 0, err
	}

	return lastLog.StepOrder + 1, nil
}
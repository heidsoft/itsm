package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticketcomment"
	"itsm-backend/ent/ticket"

	"go.uber.org/zap"
)

type TicketCommentService struct {
	client              *ent.Client
	logger              *zap.SugaredLogger
	notificationService *TicketNotificationService // 可选的通知服务
}

func NewTicketCommentService(client *ent.Client, logger *zap.SugaredLogger) *TicketCommentService {
	return &TicketCommentService{
		client: client,
		logger: logger,
	}
}

// SetNotificationService 设置通知服务（用于依赖注入）
func (s *TicketCommentService) SetNotificationService(notificationService *TicketNotificationService) {
	s.notificationService = notificationService
}

// CreateTicketComment 创建工单评论
func (s *TicketCommentService) CreateTicketComment(ctx context.Context, ticketID int, req *dto.CreateTicketCommentRequest, userID, tenantID int) (*dto.TicketCommentResponse, error) {
	s.logger.Infow("Creating ticket comment", "ticket_id", ticketID, "user_id", userID)

	// 验证工单是否存在且属于当前租户
	ticketExists, err := s.client.Ticket.Query().
		Where(
			ticket.ID(ticketID),
			ticket.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check ticket existence", "error", err)
		return nil, fmt.Errorf("failed to check ticket existence: %w", err)
	}
	if !ticketExists {
		return nil, fmt.Errorf("ticket not found")
	}

	// 创建评论
	comment, err := s.client.TicketComment.Create().
		SetTicketID(ticketID).
		SetUserID(userID).
		SetContent(req.Content).
		SetIsInternal(req.IsInternal).
		SetTenantID(tenantID).
		SetMentions(req.Mentions).
		SetAttachments(req.Attachments).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create ticket comment", "error", err)
		return nil, fmt.Errorf("failed to create ticket comment: %w", err)
	}

	// 查询用户信息
	user, err := s.client.User.Get(ctx, userID)
	if err != nil {
		s.logger.Warnw("Failed to get user", "error", err, "user_id", userID)
		// 用户信息获取失败不影响评论创建
		user = nil
	}

	// 发送评论通知
	if s.notificationService != nil {
		mentionedUserIDs := req.Mentions
		if err := s.notificationService.NotifyTicketCommented(ctx, ticketID, userID, mentionedUserIDs, tenantID); err != nil {
			s.logger.Warnw("Failed to send comment notification", "error", err)
		}
	}

	return dto.ToTicketCommentResponse(comment, user), nil
}

// ListTicketComments 获取工单评论列表
func (s *TicketCommentService) ListTicketComments(ctx context.Context, ticketID, tenantID int, currentUserID int) ([]*dto.TicketCommentResponse, error) {
	s.logger.Infow("Listing ticket comments", "ticket_id", ticketID)

	// 验证工单是否存在且属于当前租户
	ticketExists, err := s.client.Ticket.Query().
		Where(
			ticket.ID(ticketID),
			ticket.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check ticket existence", "error", err)
		return nil, fmt.Errorf("failed to check ticket existence: %w", err)
	}
	if !ticketExists {
		return nil, fmt.Errorf("ticket not found")
	}

	// 查询工单信息，判断当前用户是否有权限查看内部备注
	ticketInfo, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		s.logger.Errorw("Failed to get ticket", "error", err)
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	// 查询评论（按时间倒序）
	comments, err := s.client.TicketComment.Query().
		Where(
			ticketcomment.TicketID(ticketID),
			ticketcomment.TenantID(tenantID),
		).
		Order(ent.Desc(ticketcomment.FieldCreatedAt)).
		WithUser().
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list ticket comments", "error", err)
		return nil, fmt.Errorf("failed to list ticket comments: %w", err)
	}

	// 判断当前用户是否有权限查看内部备注
	// 只有工单处理人、申请人或管理员可以查看内部备注
	canViewInternal := currentUserID == ticketInfo.RequesterID || 
		(currentUserID == ticketInfo.AssigneeID && ticketInfo.AssigneeID > 0)

	// 转换为 DTO
	responses := make([]*dto.TicketCommentResponse, 0, len(comments))
	for _, comment := range comments {
		// 如果是内部备注，检查权限
		if comment.IsInternal && !canViewInternal {
			continue
		}

		var userEntity *ent.User
		if comment.Edges.User != nil {
			userEntity = comment.Edges.User
		} else {
			// 如果没有加载用户信息，单独查询
			userEntity, _ = s.client.User.Get(ctx, comment.UserID)
		}

		responses = append(responses, dto.ToTicketCommentResponse(comment, userEntity))
	}

	return responses, nil
}

// UpdateTicketComment 更新工单评论
func (s *TicketCommentService) UpdateTicketComment(ctx context.Context, ticketID, commentID int, req *dto.UpdateTicketCommentRequest, userID, tenantID int) (*dto.TicketCommentResponse, error) {
	s.logger.Infow("Updating ticket comment", "ticket_id", ticketID, "comment_id", commentID, "user_id", userID)

	// 查询评论
	comment, err := s.client.TicketComment.Query().
		Where(
			ticketcomment.ID(commentID),
			ticketcomment.TicketID(ticketID),
			ticketcomment.TenantID(tenantID),
		).
		WithUser().
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get ticket comment", "error", err)
		return nil, fmt.Errorf("ticket comment not found: %w", err)
	}

	// 权限检查：只有评论作者或工单处理人可以编辑
	ticketInfo, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		s.logger.Errorw("Failed to get ticket", "error", err)
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	canEdit := comment.UserID == userID || 
		(ticketInfo.AssigneeID > 0 && ticketInfo.AssigneeID == userID)
	if !canEdit {
		return nil, fmt.Errorf("permission denied: only comment author or ticket assignee can edit")
	}

	// 更新评论
	update := s.client.TicketComment.UpdateOneID(commentID)
	if req.Content != "" {
		update = update.SetContent(req.Content)
	}
	if req.IsInternal != nil {
		update = update.SetIsInternal(*req.IsInternal)
	}
	if req.Mentions != nil {
		update = update.SetMentions(req.Mentions)
	}

	updatedComment, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update ticket comment", "error", err)
		return nil, fmt.Errorf("failed to update ticket comment: %w", err)
	}

	// 重新加载用户信息
	var userEntity *ent.User
	if comment.Edges.User != nil {
		userEntity = comment.Edges.User
	} else {
		userEntity, _ = s.client.User.Get(ctx, updatedComment.UserID)
	}

	return dto.ToTicketCommentResponse(updatedComment, userEntity), nil
}

// DeleteTicketComment 删除工单评论
func (s *TicketCommentService) DeleteTicketComment(ctx context.Context, ticketID, commentID int, userID, tenantID int) error {
	s.logger.Infow("Deleting ticket comment", "ticket_id", ticketID, "comment_id", commentID, "user_id", userID)

	// 查询评论
	comment, err := s.client.TicketComment.Query().
		Where(
			ticketcomment.ID(commentID),
			ticketcomment.TicketID(ticketID),
			ticketcomment.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get ticket comment", "error", err)
		return fmt.Errorf("ticket comment not found: %w", err)
	}

	// 权限检查：只有评论作者或工单处理人可以删除
	ticketInfo, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		s.logger.Errorw("Failed to get ticket", "error", err)
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	canDelete := comment.UserID == userID || 
		(ticketInfo.AssigneeID > 0 && ticketInfo.AssigneeID == userID)
	if !canDelete {
		return fmt.Errorf("permission denied: only comment author or ticket assignee can delete")
	}

	// 删除评论
	err = s.client.TicketComment.DeleteOneID(commentID).Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete ticket comment", "error", err)
		return fmt.Errorf("failed to delete ticket comment: %w", err)
	}

	return nil
}


package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/ticketnotification"
	"itsm-backend/ent/user"
	"time"

	"go.uber.org/zap"
)

type TicketNotificationService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewTicketNotificationService(client *ent.Client, logger *zap.SugaredLogger) *TicketNotificationService {
	return &TicketNotificationService{
		client: client,
		logger: logger,
	}
}

// SendNotification 发送工单通知
func (s *TicketNotificationService) SendNotification(
	ctx context.Context,
	ticketID int,
	req *dto.SendTicketNotificationRequest,
	tenantID int,
) error {
	s.logger.Infow("Sending ticket notification", "ticket_id", ticketID, "type", req.Type, "channel", req.Channel)

	// 验证工单是否存在
	ticketExists, err := s.client.Ticket.Query().
		Where(
			ticket.ID(ticketID),
			ticket.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("failed to check ticket existence: %w", err)
	}
	if !ticketExists {
		return fmt.Errorf("ticket not found")
	}

	// 为每个用户创建通知记录
	now := time.Now()
	for _, userID := range req.UserIDs {
		// 验证用户是否存在
		userExists, err := s.client.User.Query().
			Where(user.ID(userID)).
			Exist(ctx)
		if err != nil || !userExists {
			s.logger.Warnw("User not found, skipping notification", "user_id", userID)
			continue
		}

		// 检查用户通知偏好
		preferences, err := s.getUserNotificationPreferences(ctx, userID)
		if err != nil {
			s.logger.Warnw("Failed to get user preferences, using defaults", "user_id", userID, "error", err)
		}

		// 根据渠道和用户偏好决定是否发送
		shouldSend := false
		switch req.Channel {
		case "email":
			shouldSend = preferences.EmailEnabled
		case "in_app":
			shouldSend = preferences.InAppEnabled
		case "sms":
			shouldSend = preferences.SmsEnabled
		}

		// 站内消息总是创建记录（即使其他渠道被禁用）
		if req.Channel == "in_app" || shouldSend {
			notification, err := s.client.TicketNotification.Create().
				SetTicketID(ticketID).
				SetUserID(userID).
				SetType(req.Type).
				SetChannel(req.Channel).
				SetContent(req.Content).
				SetTenantID(tenantID).
				SetStatus("pending").
				Save(ctx)
			if err != nil {
				s.logger.Errorw("Failed to create notification", "error", err, "user_id", userID)
				continue
			}

			// 如果是站内消息，立即标记为已发送
			if req.Channel == "in_app" {
				_, err = s.client.TicketNotification.UpdateOneID(notification.ID).
					SetStatus("sent").
					SetNillableSentAt(&now).
					Save(ctx)
				if err != nil {
					s.logger.Warnw("Failed to update notification status", "error", err)
				}
			} else if shouldSend {
				// 异步发送邮件或短信（这里只是标记，实际发送需要后台任务）
				// TODO: 集成邮件服务和短信服务
				_, err = s.client.TicketNotification.UpdateOneID(notification.ID).
					SetStatus("sent").
					SetNillableSentAt(&now).
					Save(ctx)
				if err != nil {
					s.logger.Warnw("Failed to update notification status", "error", err)
				}
			}
		}
	}

	return nil
}

// NotifyTicketCreated 工单创建时发送通知
func (s *TicketNotificationService) NotifyTicketCreated(ctx context.Context, ticket *ent.Ticket) error {
	if ticket.AssigneeID > 0 {
		content := fmt.Sprintf("您被分配了一个新工单：%s (#%s)", ticket.Title, ticket.TicketNumber)
		return s.SendNotification(ctx, ticket.ID, &dto.SendTicketNotificationRequest{
			UserIDs: []int{ticket.AssigneeID},
			Type:    "created",
			Channel: "in_app",
			Content: content,
		}, ticket.TenantID)
	}
	return nil
}

// NotifyTicketAssigned 工单分配时发送通知
func (s *TicketNotificationService) NotifyTicketAssigned(ctx context.Context, ticketID, assigneeID, tenantID int) error {
	ticket, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	content := fmt.Sprintf("您被分配了工单：%s (#%s)", ticket.Title, ticket.TicketNumber)
	return s.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: []int{assigneeID},
		Type:    "assigned",
		Channel: "in_app",
		Content: content,
	}, tenantID)
}

// NotifyTicketStatusChanged 工单状态变更时发送通知
func (s *TicketNotificationService) NotifyTicketStatusChanged(
	ctx context.Context,
	ticketID int,
	oldStatus, newStatus string,
	tenantID int,
) error {
	ticket, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	content := fmt.Sprintf("工单 #%s 状态已从 %s 变更为 %s", ticket.TicketNumber, oldStatus, newStatus)
	userIDs := []int{ticket.RequesterID}
	if ticket.AssigneeID > 0 {
		userIDs = append(userIDs, ticket.AssigneeID)
	}

	return s.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: userIDs,
		Type:    "status_changed",
		Channel: "in_app",
		Content: content,
	}, tenantID)
}

// NotifyTicketCommented 工单评论时发送通知
func (s *TicketNotificationService) NotifyTicketCommented(
	ctx context.Context,
	ticketID int,
	commenterID int,
	mentionedUserIDs []int,
	tenantID int,
) error {
	ticket, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	// 通知工单相关人员和被@的用户
	userIDs := []int{ticket.RequesterID}
	if ticket.AssigneeID > 0 && ticket.AssigneeID != commenterID {
		userIDs = append(userIDs, ticket.AssigneeID)
	}
	
	// 添加被@的用户（排除评论者自己）
	for _, userID := range mentionedUserIDs {
		if userID != commenterID {
			exists := false
			for _, id := range userIDs {
				if id == userID {
					exists = true
					break
				}
			}
			if !exists {
				userIDs = append(userIDs, userID)
			}
		}
	}

	if len(userIDs) == 0 {
		return nil
	}

	content := fmt.Sprintf("工单 #%s 有新的评论", ticket.TicketNumber)
	return s.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: userIDs,
		Type:    "commented",
		Channel: "in_app",
		Content: content,
	}, tenantID)
}

// NotifySLAWarning SLA即将到期时发送提醒
func (s *TicketNotificationService) NotifySLAWarning(
	ctx context.Context,
	ticketID int,
	warningType string, // response_deadline, resolution_deadline
	deadline time.Time,
	tenantID int,
) error {
	ticket, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	content := fmt.Sprintf("工单 #%s 的SLA %s 即将在 %s 到期", 
		ticket.TicketNumber, 
		map[string]string{
			"response_deadline":   "响应时间",
			"resolution_deadline": "解决时间",
		}[warningType],
		deadline.Format("2006-01-02 15:04:05"))

	userIDs := []int{ticket.RequesterID}
	if ticket.AssigneeID > 0 {
		userIDs = append(userIDs, ticket.AssigneeID)
	}

	return s.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: userIDs,
		Type:    "sla_warning",
		Channel: "in_app",
		Content: content,
	}, tenantID)
}

// ListTicketNotifications 获取工单通知列表
func (s *TicketNotificationService) ListTicketNotifications(
	ctx context.Context,
	ticketID, tenantID int,
) ([]*dto.TicketNotificationResponse, error) {
	notifications, err := s.client.TicketNotification.Query().
		Where(
			ticketnotification.TicketID(ticketID),
			ticketnotification.TenantID(tenantID),
		).
		Order(ent.Desc(ticketnotification.FieldCreatedAt)).
		WithUser().
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list ticket notifications", "error", err)
		return nil, fmt.Errorf("failed to list ticket notifications: %w", err)
	}

	responses := make([]*dto.TicketNotificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		var userEntity *ent.User
		if notification.Edges.User != nil {
			userEntity = notification.Edges.User
		} else {
			userEntity, _ = s.client.User.Get(ctx, notification.UserID)
		}
		responses = append(responses, dto.ToTicketNotificationResponse(notification, userEntity))
	}

	return responses, nil
}

// ListUserNotifications 获取用户通知列表
func (s *TicketNotificationService) ListUserNotifications(
	ctx context.Context,
	userID, tenantID int,
	page, pageSize int,
	read *bool,
) ([]*dto.TicketNotificationResponse, int, error) {
	query := s.client.TicketNotification.Query().
		Where(
			ticketnotification.UserID(userID),
			ticketnotification.TenantID(tenantID),
		)

	if read != nil {
		if *read {
			query = query.Where(ticketnotification.ReadAtNotNil())
		} else {
			query = query.Where(ticketnotification.ReadAtIsNil())
		}
	}

	// 获取总数
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// 分页查询
	notifications, err := query.
		Order(ent.Desc(ticketnotification.FieldCreatedAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		WithUser().
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list user notifications", "error", err)
		return nil, 0, fmt.Errorf("failed to list user notifications: %w", err)
	}

	responses := make([]*dto.TicketNotificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		var userEntity *ent.User
		if notification.Edges.User != nil {
			userEntity = notification.Edges.User
		} else {
			userEntity, _ = s.client.User.Get(ctx, notification.UserID)
		}
		responses = append(responses, dto.ToTicketNotificationResponse(notification, userEntity))
	}

	return responses, total, nil
}

// MarkNotificationRead 标记通知为已读
func (s *TicketNotificationService) MarkNotificationRead(
	ctx context.Context,
	notificationID, userID, tenantID int,
) error {
	_, err := s.client.TicketNotification.Query().
		Where(
			ticketnotification.ID(notificationID),
			ticketnotification.UserID(userID),
			ticketnotification.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("notification not found: %w", err)
	}

	now := time.Now()
	_, err = s.client.TicketNotification.UpdateOneID(notificationID).
		SetStatus("read").
		SetNillableReadAt(&now).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to mark notification as read", "error", err)
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return nil
}

// MarkAllNotificationsRead 标记所有通知为已读
func (s *TicketNotificationService) MarkAllNotificationsRead(
	ctx context.Context,
	userID, tenantID int,
) error {
	now := time.Now()
	_, err := s.client.TicketNotification.Update().
		Where(
			ticketnotification.UserID(userID),
			ticketnotification.TenantID(tenantID),
			ticketnotification.ReadAtIsNil(),
		).
		SetStatus("read").
		SetNillableReadAt(&now).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to mark all notifications as read", "error", err)
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	return nil
}

// GetUserNotificationPreferences 获取用户通知偏好
func (s *TicketNotificationService) GetUserNotificationPreferences(
	ctx context.Context,
	userID int,
) (*dto.NotificationPreferencesResponse, error) {
	// TODO: 创建用户通知偏好表，这里先返回默认值
	// 可以从用户表的扩展字段或单独的偏好表中获取
	return &dto.NotificationPreferencesResponse{
		UserID:         userID,
		EmailEnabled:   true,
		InAppEnabled:   true,
		SmsEnabled:     false,
		SlaWarningTime: 30, // 默认30分钟
	}, nil
}

// UpdateUserNotificationPreferences 更新用户通知偏好
func (s *TicketNotificationService) UpdateUserNotificationPreferences(
	ctx context.Context,
	userID int,
	req *dto.UpdateNotificationPreferencesRequest,
) (*dto.NotificationPreferencesResponse, error) {
	// TODO: 保存到用户通知偏好表
	// 这里先返回更新后的值
	return &dto.NotificationPreferencesResponse{
		UserID:         userID,
		EmailEnabled:   req.EmailEnabled,
		InAppEnabled:   req.InAppEnabled,
		SmsEnabled:     req.SmsEnabled,
		SlaWarningTime: req.SlaWarningTime,
	}, nil
}

// getUserNotificationPreferences 内部方法：获取用户通知偏好（带默认值）
func (s *TicketNotificationService) getUserNotificationPreferences(
	ctx context.Context,
	userID int,
) (*dto.NotificationPreferencesResponse, error) {
	prefs, err := s.GetUserNotificationPreferences(ctx, userID)
	if err != nil {
		// 返回默认值
		return &dto.NotificationPreferencesResponse{
			UserID:         userID,
			EmailEnabled:   true,
			InAppEnabled:   true,
			SmsEnabled:     false,
			SlaWarningTime: 30,
		}, nil
	}
	return prefs, nil
}


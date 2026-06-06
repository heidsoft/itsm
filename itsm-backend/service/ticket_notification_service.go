package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/ticketnotification"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

func ticketNotificationStringPtr(s string) *string {
	return &s
}

type TicketNotificationService struct {
	client       *ent.Client
	logger       *zap.SugaredLogger
	emailService *EmailService
	smsService   *SMSService
}

// NewTicketNotificationService 创建通知服务
func NewTicketNotificationService(client *ent.Client, logger *zap.SugaredLogger) *TicketNotificationService {
	return &TicketNotificationService{
		client: client,
		logger: logger,
	}
}

// SetEmailService 设置邮件服务
func (s *TicketNotificationService) SetEmailService(emailService *EmailService) {
	s.emailService = emailService
}

// SetSMSService 设置短信服务
func (s *TicketNotificationService) SetSMSService(smsService *SMSService) {
	s.smsService = smsService
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

			// 同步创建到通用notifications表(供前端统一查询)
			_, _ = s.client.Notification.Create().
				SetTitle(req.Type).
				SetMessage(req.Content).
				SetType(req.Type).
				SetUserID(userID).
				SetTenantID(tenantID).
				SetNillableActionURL(ticketNotificationStringPtr(fmt.Sprintf("/tickets/%d", ticketID))).
				SetNillableActionText(ticketNotificationStringPtr("查看工单")).
				Save(ctx)

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
				// 实际发送邮件或短信
				var sendErr error
				for _, userID := range req.UserIDs {
					// 获取用户邮箱和手机号
					userEntity, _ := s.client.User.Get(ctx, userID)
					if userEntity == nil {
						continue
					}

					// 获取工单信息
					ticketEntity, _ := s.client.Ticket.Get(ctx, ticketID)
					if ticketEntity == nil {
						continue
					}

					switch req.Channel {
					case "email":
						if s.emailService != nil && userEntity.Email != "" {
							sendErr = s.emailService.SendTicketNotification(
								ctx,
								[]string{userEntity.Email},
								ticketEntity.TicketNumber,
								ticketEntity.Title,
								req.Type,
								req.Content,
							)
							if sendErr != nil {
								s.logger.Errorw("Failed to send email notification", "error", sendErr, "user_id", userID)
							}
						}
					case "sms":
						if s.smsService != nil && userEntity.Phone != "" {
							sendErr = s.smsService.SendTicketNotification(
								ctx,
								[]string{userEntity.Phone},
								ticketEntity.TicketNumber,
								req.Type,
							)
							if sendErr != nil {
								s.logger.Errorw("Failed to send SMS notification", "error", sendErr, "user_id", userID)
							}
						}
					}
				}

				// 更新通知状态
				status := "sent"
				if sendErr != nil {
					status = "failed"
				}
				_, err = s.client.TicketNotification.UpdateOneID(notification.ID).
					SetStatus(status).
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
// 通知目标:
//  1. 工单处理人(AssigneeID),如果有
//  2. 工单创建人(ReporterID),如果是普通用户
//  3. 所有租户内管理员(如果没有处理人)
func (s *TicketNotificationService) NotifyTicketCreated(ctx context.Context, ticket *ent.Ticket) error {
	userIDs := []int{}

	// 1. 处理人
	if ticket.AssigneeID > 0 {
		userIDs = append(userIDs, ticket.AssigneeID)
	}

	// 2. 创建人(去重)
	if ticket.RequesterID > 0 {
		dup := false
		for _, id := range userIDs {
			if id == ticket.RequesterID {
				dup = true
				break
			}
		}
		if !dup {
			userIDs = append(userIDs, ticket.RequesterID)
		}
	}

	// 3. 如果只有创建人(没有处理人),广播给所有admin
	if len(userIDs) <= 1 && ticket.RequesterID > 0 {
		admins, err := s.client.User.Query().
			Where(user.TenantID(ticket.TenantID)).
			Where(user.IDNEQ(ticket.RequesterID)).
			All(ctx)
		if err == nil {
			for _, admin := range admins {
				dup := false
				for _, id := range userIDs {
					if id == admin.ID {
						dup = true
						break
					}
				}
				if !dup {
					userIDs = append(userIDs, admin.ID)
				}
			}
		}
	}

	if len(userIDs) == 0 {
		return nil
	}

	content := fmt.Sprintf("新工单已创建：%s (#%s)", ticket.Title, ticket.TicketNumber)
	return s.SendNotification(ctx, ticket.ID, &dto.SendTicketNotificationRequest{
		UserIDs: userIDs,
		Type:    "created",
		Channel: "in_app",
		Content: content,
	}, ticket.TenantID)
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

// NotifySLABreached SLA违规时发送通知
func (s *TicketNotificationService) NotifySLABreached(
	ctx context.Context,
	ticketID int,
	violationType string, // response_time, resolution_time
	exceededMinutes float64,
	tenantID int,
) error {
	ticket, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	slaType := map[string]string{
		"response_time":   "响应时间",
		"resolution_time": "解决时间",
	}[violationType]

	content := fmt.Sprintf("【SLA违规】工单 #%s 的%s已违反SLA，超时 %.1f 分钟",
		ticket.TicketNumber, slaType, exceededMinutes)

	// 获取需要通知的用户列表（创建人、处理人、相关经理）
	userIDs := []int{ticket.RequesterID}
	if ticket.AssigneeID > 0 {
		userIDs = append(userIDs, ticket.AssigneeID)
	}

	// 根据配置的通知渠道发送
	// 1. 站内消息
	if err := s.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: userIDs,
		Type:    "sla_breached",
		Channel: "in_app",
		Content: content,
	}, tenantID); err != nil {
		s.logger.Errorw("Failed to send in-app SLA breach notification", "error", err)
	}

	// 2. 邮件通知
	if s.emailService != nil {
		// 获取所有需要通知的用户邮箱
		var emails []string
		for _, userID := range userIDs {
			userEntity, _ := s.client.User.Get(ctx, userID)
			if userEntity != nil && userEntity.Email != "" {
				emails = append(emails, userEntity.Email)
			}
		}
		if len(emails) > 0 {
			if err := s.emailService.SendTicketNotification(ctx, emails, ticket.TicketNumber, ticket.Title, "sla_breached", content); err != nil {
				s.logger.Warnw("failed to send SLA breach email notification", "error", err, "ticket_id", ticketID)
			}
		}
	}

	// 3. 短信通知（严重级别时）
	if exceededMinutes > 60 && s.smsService != nil {
		var phones []string
		for _, userID := range userIDs {
			userEntity, _ := s.client.User.Get(ctx, userID)
			if userEntity != nil && userEntity.Phone != "" {
				phones = append(phones, userEntity.Phone)
			}
		}
		if len(phones) > 0 {
			smsContent := fmt.Sprintf("【ITSM系统】SLA告警：工单 %s 的%s已超时 %.1f 分钟，请立即处理！",
				ticket.TicketNumber, slaType, exceededMinutes)
			if err := s.smsService.Send(ctx, &SMSMessage{
				PhoneNumbers: phones,
				Content:      smsContent,
			}); err != nil {
				s.logger.Warnw("failed to send SLA breach SMS notification", "error", err, "ticket_id", ticketID)
			}
		}
	}

	return nil
}

// NotifySLAAlertLevelChanged SLA预警级别变更时发送通知
func (s *TicketNotificationService) NotifySLAAlertLevelChanged(
	ctx context.Context,
	ticketID int,
	alertLevel string, // warning, critical
	percentage float64,
	tenantID int,
) error {
	ticket, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	levelText := map[string]string{
		"warning":  "警告",
		"critical": "严重",
	}[alertLevel]

	content := fmt.Sprintf("【SLA%s】工单 #%s 剩余时间不足 %.1f%%，请及时处理！",
		levelText, ticket.TicketNumber, percentage)

	userIDs := []int{ticket.RequesterID}
	if ticket.AssigneeID > 0 {
		userIDs = append(userIDs, ticket.AssigneeID)
	}

	return s.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: userIDs,
		Type:    "sla_alert",
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
			userEntity, err = s.client.User.Get(ctx, notification.UserID)
			if err != nil {
				s.logger.Warnw("failed to get user for notification response", "error", err, "user_id", notification.UserID)
			}
			responses = append(responses, dto.ToTicketNotificationResponse(notification, userEntity))
		}
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
			userEntity, err = s.client.User.Get(ctx, notification.UserID)
			if err != nil {
				s.logger.Warnw("failed to get user for notification response", "error", err, "user_id", notification.UserID)
			}
			responses = append(responses, dto.ToTicketNotificationResponse(notification, userEntity))
		}
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
	// 注意：用户通知偏好存储在用户表的 preferences JSON 字段中
	// 如果需要单独的 preference 表，可以在未来版本中实现
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
	// 注意：偏好应该保存到用户表的 preferences 字段
	// 当前实现仅返回更新后的值
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

// SendAssignmentNotification 发送工单分配通知
func (s *TicketNotificationService) SendAssignmentNotification(ticketID, assigneeID, assignedBy int) {
	if s == nil {
		return
	}
	ctx := context.Background()
	content := fmt.Sprintf("您被分配了工单 #%d", ticketID)

	tenantID := s.resolveTenantID(ctx, ticketID)
	if err := s.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: []int{assigneeID},
		Type:    "assigned",
		Channel: "in_app",
		Content: content,
	}, tenantID); err != nil {
		s.logger.Warnw("failed to send assignment notification", "error", err, "ticket_id", ticketID)
	}
}

// SendEscalationNotification 发送工单升级通知
func (s *TicketNotificationService) SendEscalationNotification(ticketID, newAssignee, escalatedBy int, reason string) {
	if s == nil {
		return
	}
	ctx := context.Background()
	content := fmt.Sprintf("工单 #%d 已被升级，新处理人: %d", ticketID, newAssignee)

	tenantID := s.resolveTenantID(ctx, ticketID)
	if err := s.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: []int{newAssignee},
		Type:    "escalated",
		Channel: "in_app",
		Content: content,
	}, tenantID); err != nil {
		s.logger.Warnw("failed to send escalation notification", "error", err, "ticket_id", ticketID)
	}
}

// SendResolutionNotification 发送工单解决通知
func (s *TicketNotificationService) SendResolutionNotification(ticketID, requesterID, resolvedBy int) {
	if s == nil {
		return
	}
	ctx := context.Background()
	content := fmt.Sprintf("工单 #%d 已被解决", ticketID)

	tenantID := s.resolveTenantID(ctx, ticketID)
	if err := s.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: []int{requesterID},
		Type:    "resolved",
		Channel: "in_app",
		Content: content,
	}, tenantID); err != nil {
		s.logger.Warnw("failed to send resolution notification", "error", err, "ticket_id", ticketID)
	}
}

// resolveTenantID resolves the tenant ID for a ticket from the database.
// Returns 0 if the ticket cannot be found (callers should handle this appropriately).
func (s *TicketNotificationService) resolveTenantID(ctx context.Context, ticketID int) int {
	ticketEntity, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		s.logger.Warnw("failed to resolve tenant ID for ticket", "error", err, "ticket_id", ticketID)
		return 0
	}
	return ticketEntity.TenantID
}

package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/slaalerthistory"
	"itsm-backend/ent/slaalertrule"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

type EscalationService struct {
	client            *ent.Client
	logger            *zap.SugaredLogger
	notificationSvc   *TicketNotificationService
}

func NewEscalationService(client *ent.Client, logger *zap.SugaredLogger) *EscalationService {
	return &EscalationService{
		client: client,
		logger: logger,
	}
}

// SetNotificationService 设置通知服务
func (e *EscalationService) SetNotificationService(notificationSvc *TicketNotificationService) {
	e.notificationSvc = notificationSvc
}

// ProcessEscalations 处理所有升级任务
func (e *EscalationService) ProcessEscalations(ctx context.Context, tenantID int) error {
	e.logger.Infow("Processing escalations", "tenant_id", tenantID)

	// 1. 检查SLA预警规则中的升级配置
	if err := e.processSLAEscalations(ctx, tenantID); err != nil {
		e.logger.Errorw("Failed to process SLA escalations", "error", err)
	}

	// 2. 检查长时间未解决的工单
	if err := e.processLongPendingTickets(ctx, tenantID); err != nil {
		e.logger.Errorw("Failed to process long pending tickets", "error", err)
	}

	// 3. 检查未分配工单
	if err := e.processUnassignedTickets(ctx, tenantID); err != nil {
		e.logger.Errorw("Failed to process unassigned tickets", "error", err)
	}

	e.logger.Infow("Escalation processing completed", "tenant_id", tenantID)
	return nil
}

// processSLAEscalations 处理SLA预警规则中的升级
func (e *EscalationService) processSLAEscalations(ctx context.Context, tenantID int) error {
	// 获取所有启用升级的预警规则
	alertRules, err := e.client.SLAAlertRule.Query().
		Where(
			slaalertrule.TenantIDEQ(tenantID),
			slaalertrule.IsActiveEQ(true),
			slaalertrule.EscalationEnabledEQ(true),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query alert rules: %w", err)
	}

	for _, rule := range alertRules {
		// 检查是否有未处理的预警
		pendingAlerts, err := e.client.SLAAlertHistory.Query().
			Where(
				slaalerthistory.AlertRuleIDEQ(rule.ID),
				slaalerthistory.ResolvedAtIsNil(),
			).
			All(ctx)
		if err != nil {
			continue
		}

		for _, alert := range pendingAlerts {
			// 检查是否达到升级阈值
			escalationLevels := rule.EscalationLevels
			if len(escalationLevels) == 0 {
				continue
			}

			// 计算已过去的时间
			timeSinceAlert := time.Since(alert.CreatedAt).Minutes()

			// 检查每个升级级别
			for _, level := range escalationLevels {
				threshold, ok := level["threshold"].(float64)
				if !ok {
					continue
				}

				// 如果达到升级时间且尚未升级到该级别
				if timeSinceAlert >= threshold && alert.EscalationLevel < int(threshold) {
					if err := e.escalateToLevel(ctx, alert, rule, int(threshold), tenantID); err != nil {
						e.logger.Errorw("Failed to escalate alert", "alert_id", alert.ID, "error", err)
					}
					break
				}
			}
		}
	}

	return nil
}

// escalateToLevel 升级到指定级别
func (e *EscalationService) escalateToLevel(ctx context.Context, alert *ent.SLAAlertHistory, rule *ent.SLAAlertRule, level int, tenantID int) error {
	// 更新升级级别
	_, err := e.client.SLAAlertHistory.UpdateOneID(alert.ID).
		SetEscalationLevel(level).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update escalation level: %w", err)
	}

	// 获取通知用户列表
	notifyUsers := e.getEscalationNotifyUsers(ctx, level, rule)
	if len(notifyUsers) == 0 {
		return nil
	}

	// 发送升级通知
	if e.notificationSvc != nil {
		content := fmt.Sprintf("【SLA升级告警】工单 #%s (%s) 已升级至级别 %d，请关注处理！",
			alert.TicketNumber, alert.TicketTitle, level)

		for _, userID := range notifyUsers {
			err := e.notificationSvc.SendNotification(ctx, alert.TicketID, &dto.SendTicketNotificationRequest{
				UserIDs: []int{userID},
				Type:    "sla_escalated",
				Channel: "in_app",
				Content: content,
			}, tenantID)
			
			if err != nil {
				e.logger.Errorw("Failed to send SLA escalation notification", "user_id", userID, "error", err)
			}
		}
	}

	e.logger.Infow("Alert escalated", "alert_id", alert.ID, "ticket_number", alert.TicketNumber, "escalation_level", level)
	return nil
}

// getEscalationNotifyUsers 获取指定升级级别的通知用户
func (e *EscalationService) getEscalationNotifyUsers(ctx context.Context, level int, rule *ent.SLAAlertRule) []int {
	var userIDs []int

	// 从预警规则获取通知用户
	if rule.EscalationLevels != nil && len(rule.EscalationLevels) > 0 {
		for _, levelConfig := range rule.EscalationLevels {
			if lvl, ok := levelConfig["level"].(float64); ok && int(lvl) == level {
				if users, ok := levelConfig["notify_users"].([]interface{}); ok {
					for _, u := range users {
						if uid, ok := u.(float64); ok {
							userIDs = append(userIDs, int(uid))
						}
					}
				}
			}
		}
	}

	// 如果没有配置，查找该SLA定义的管理员
	if len(userIDs) == 0 {
		// 获取所有管理员用户
		admins, _ := e.client.User.Query().
			Where(user.RoleEQ("admin")).
			IDs(ctx)
		userIDs = append(userIDs, admins...)
	}

	return userIDs
}

// processLongPendingTickets 处理长时间未解决的工单
func (e *EscalationService) processLongPendingTickets(ctx context.Context, tenantID int) error {
	// 查找超过24小时未解决的工单
	threshold := time.Now().Add(-24 * time.Hour)

	tickets, err := e.client.Ticket.Query().
		Where(
			ticket.TenantIDEQ(tenantID),
			ticket.ResolvedAtIsNil(),
			ticket.CreatedAtLTE(threshold),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query long pending tickets: %w", err)
	}

	for _, t := range tickets {
		// 检查是否已发送过长时间未解决通知
		exists, _ := e.client.SLAAlertHistory.Query().
			Where(
				slaalerthistory.TicketIDEQ(t.ID),
				slaalerthistory.AlertLevelEQ("long_pending"),
				slaalerthistory.ResolvedAtIsNil(),
			).
			Exist(ctx)

		if !exists && e.notificationSvc != nil {
			// 获取该工单的处理人或创建人
			userIDs := []int{t.RequesterID}
			if t.AssigneeID > 0 {
				userIDs = append(userIDs, t.AssigneeID)
			}

			content := fmt.Sprintf("【超时提醒】工单 #%s (%s) 已超过24小时未解决，请及时处理！",
				t.TicketNumber, t.Title)

			for _, userID := range userIDs {
				if err := e.notificationSvc.SendNotification(ctx, t.ID, &dto.SendTicketNotificationRequest{
					UserIDs: []int{userID},
					Type:    "long_pending",
					Channel: "in_app",
					Content: content,
				}, tenantID); err != nil {
					e.logger.Errorw("Failed to send long pending notification", "ticket_id", t.ID, "user_id", userID, "error", err)
				}
			}

			// 通知管理员
			admins, _ := e.client.User.Query().
				Where(user.RoleEQ("admin")).
				IDs(ctx)

			for _, adminID := range admins {
				if err := e.notificationSvc.SendNotification(ctx, t.ID, &dto.SendTicketNotificationRequest{
					UserIDs: []int{adminID},
					Type:    "long_pending_admin",
					Channel: "in_app",
					Content: content,
				}, tenantID); err != nil {
					e.logger.Errorw("Failed to send long pending admin notification", "ticket_id", t.ID, "admin_id", adminID, "error", err)
				}
			}

			e.logger.Infow("Long pending ticket notification sent", "ticket_id", t.ID, "ticket_number", t.TicketNumber)
		}
	}

	return nil
}

// processUnassignedTickets 处理未分配的工单
func (e *EscalationService) processUnassignedTickets(ctx context.Context, tenantID int) error {
	// 查找超过2小时未分配的工单
	threshold := time.Now().Add(-2 * time.Hour)

	tickets, err := e.client.Ticket.Query().
		Where(
			ticket.TenantIDEQ(tenantID),
			ticket.AssigneeIDIsNil(),
			ticket.StatusNEQ("closed"),
			ticket.CreatedAtLTE(threshold),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query unassigned tickets: %w", err)
	}

	for _, t := range tickets {
		// 检查是否已发送过未分配通知
		exists, _ := e.client.SLAAlertHistory.Query().
			Where(
				slaalerthistory.TicketIDEQ(t.ID),
				slaalerthistory.AlertLevelEQ("unassigned"),
				slaalerthistory.ResolvedAtIsNil(),
			).
			Exist(ctx)

		if !exists && e.notificationSvc != nil {
			// 通知管理员
			admins, _ := e.client.User.Query().
				Where(user.RoleEQ("admin")).
				IDs(ctx)

			if len(admins) > 0 {
				content := fmt.Sprintf("【未分配提醒】工单 #%s (%s) 已超过2小时未分配，请及时处理！",
					t.TicketNumber, t.Title)

				for _, adminID := range admins {
					if err := e.notificationSvc.SendNotification(ctx, t.ID, &dto.SendTicketNotificationRequest{
						UserIDs: []int{adminID},
						Type:    "unassigned",
						Channel: "in_app",
						Content: content,
					}, tenantID); err != nil {
						e.logger.Errorw("Failed to send unassigned notification", "ticket_id", t.ID, "admin_id", adminID, "error", err)
					}
				}

				e.logger.Infow("Unassigned ticket notification sent", "ticket_id", t.ID, "ticket_number", t.TicketNumber)
			}
		}
	}

	return nil
}

// EscalateTicket 手动升级工单
func (e *EscalationService) EscalateTicket(ctx context.Context, ticketID int, reason string, notifyUsers []int, tenantID int) error {
	ticket, err := e.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	// 创建升级记录
	_, err = e.client.SLAAlertHistory.Create().
		SetTicketID(ticketID).
		SetTicketNumber(ticket.TicketNumber).
		SetTicketTitle(ticket.Title).
		SetAlertRuleID(0).
		SetAlertRuleName("Manual Escalation").
		SetAlertLevel("manual").
		SetThresholdPercentage(0).
		SetActualPercentage(0).
		SetNotificationSent(true).
		SetEscalationLevel(1).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create escalation record: %w", err)
	}

	// 发送通知
	if e.notificationSvc != nil && len(notifyUsers) > 0 {
		content := fmt.Sprintf("【工单升级】#%s (%s) 已被手动升级，原因：%s",
			ticket.TicketNumber, ticket.Title, reason)

		for _, userID := range notifyUsers {
			if err := e.notificationSvc.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
				UserIDs: []int{userID},
				Type:    "manual_escalation",
				Channel: "in_app",
				Content: content,
			}, tenantID); err != nil {
				e.logger.Errorw("Failed to send manual escalation notification", "ticket_id", ticketID, "user_id", userID, "error", err)
			}
		}
	}

	e.logger.Infow("Ticket manually escalated", "ticket_id", ticketID, "ticket_number", ticket.TicketNumber, "reason", reason)
	return nil
}

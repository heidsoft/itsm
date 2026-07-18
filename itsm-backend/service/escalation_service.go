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
	client          *ent.Client
	logger          *zap.SugaredLogger
	notificationSvc *TicketNotificationService
	matrixSvc       *EscalationMatrixService
}

func NewEscalationService(client *ent.Client, logger *zap.SugaredLogger) *EscalationService {
	return &EscalationService{
		client:    client,
		logger:    logger,
		matrixSvc: NewEscalationMatrixService(logger),
	}
}

// SetNotificationService 设置通知服务
func (e *EscalationService) SetNotificationService(notificationSvc *TicketNotificationService) {
	e.notificationSvc = notificationSvc
}

// SetMatrixService 设置升级矩阵服务（供注入自定义矩阵/测试）
func (e *EscalationService) SetMatrixService(matrixSvc *EscalationMatrixService) {
	if matrixSvc != nil {
		e.matrixSvc = matrixSvc
	}
}

// MatrixService 返回内部升级矩阵服务（用于测试与调用方访问）
func (e *EscalationService) MatrixService() *EscalationMatrixService {
	return e.matrixSvc
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
//
// 升级策略：
//  1. 取工单的 Priority 字段
//  2. 调用 EscalationMatrixService.FindNextEscalationLevel 获取下一个应触发的升级级别
//  3. 若存在则升级（多级连续升级：若 elapsed 同时跨多级，每级都会记录一次）
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
			// 计算已过去的时间（分钟）
			elapsedMinutes := int(time.Since(alert.CreatedAt).Minutes())

			// 取工单优先级
			priority := e.resolveTicketPriority(ctx, alert.TicketID)
			if priority == "" {
				priority = "medium" // 默认值
			}

			// 使用升级矩阵查下一个升级级别
			// 多级连续升级：while 循环，逐级推进
			currentMax := alert.EscalationLevel
			for {
				nextLevel := e.matrixSvc.FindNextEscalationLevel(tenantID, priority, elapsedMinutes, currentMax)
				if nextLevel == nil {
					break
				}
				if err := e.escalateToLevelByMatrix(ctx, alert, rule, nextLevel, tenantID, priority); err != nil {
					e.logger.Errorw("Failed to escalate alert",
						"alert_id", alert.ID,
						"level", nextLevel.Level,
						"error", err)
					break
				}
				currentMax = nextLevel.Level
				e.logger.Infow(
					"Alert escalated via matrix",
					"alert_id", alert.ID,
					"ticket_number", alert.TicketNumber,
					"priority", priority,
					"level", nextLevel.Level,
					"elapsed_minutes", elapsedMinutes,
					"notify_roles", nextLevel.NotifyRoles,
				)
			}
		}
	}

	return nil
}

// resolveTicketPriority 从工单查询优先级
func (e *EscalationService) resolveTicketPriority(ctx context.Context, ticketID int) string {
	if ticketID <= 0 || e.client == nil {
		return ""
	}
	t, err := e.client.Ticket.Get(ctx, ticketID)
	if err != nil || t == nil {
		return ""
	}
	return string(t.Priority)
}

// escalateToLevelByMatrix 按矩阵级别升级
func (e *EscalationService) escalateToLevelByMatrix(
	ctx context.Context,
	alert *ent.SLAAlertHistory,
	rule *ent.SLAAlertRule,
	level *EscalationLevel,
	tenantID int,
	priority string,
) error {
	// 1. 更新告警的 escalation level
	_, err := e.client.SLAAlertHistory.UpdateOneID(alert.ID).
		SetEscalationLevel(level.Level).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update escalation level: %w", err)
	}

	// 2. 解析 notify_user_ids（按角色名→具体用户）
	notifyUsers := e.resolveNotifyUsers(ctx, level, tenantID)

	// 3. 发送通知
	if e.notificationSvc != nil && len(notifyUsers) > 0 {
		content := fmt.Sprintf("【SLA矩阵升级】工单 #%s (%s) [优先级 %s] 已升级至 L%d：%s",
			alert.TicketNumber, alert.TicketTitle, priority, level.Level, level.Description)
		for _, userID := range notifyUsers {
			err := e.notificationSvc.SendNotification(ctx, alert.TicketID, &dto.SendTicketNotificationRequest{
				UserIDs: []int{userID},
				Type:    "sla_escalated",
				Channel: "in_app",
				Content: content,
			}, tenantID)
			if err != nil {
				e.logger.Errorw("Failed to send SLA escalation notification",
					"user_id", userID, "error", err)
			}
		}
	}

	e.logger.Infow(
		"Matrix-based escalation completed",
		"alert_id", alert.ID,
		"level", level.Level,
		"notify_users_count", len(notifyUsers),
	)
	return nil
}

// resolveNotifyUsers 解析升级级别的通知用户
//
// 输入：
//   - level: 矩阵级别配置（含 NotifyRoles 和 NotifyUserIDs）
//   - tenantID: 租户 ID
//
// 输出：合并后的 userIDs（先去 NotifyUserIDs，再按 NotifyRoles 查询）
func (e *EscalationService) resolveNotifyUsers(ctx context.Context, level *EscalationLevel, tenantID int) []int {
	if level == nil {
		return nil
	}
	seen := make(map[int]bool)
	var out []int
	for _, uid := range level.NotifyUserIDs {
		if !seen[uid] {
			out = append(out, uid)
			seen[uid] = true
		}
	}
	// 按角色名查 user（简化：以 role 字段精确匹配角色名）
	if e.client != nil {
		for _, role := range level.NotifyRoles {
			ids, _ := e.client.User.Query().
				Where(user.TenantIDEQ(tenantID)).
				Where(user.RoleEQ(user.Role(role))).
				IDs(ctx)
			for _, uid := range ids {
				if !seen[uid] {
					out = append(out, uid)
					seen[uid] = true
				}
			}
		}
	}
	return out
}

// getEscalationNotifyUsers 获取指定升级级别的通知用户
func (e *EscalationService) getEscalationNotifyUsers(ctx context.Context, level int, rule *ent.SLAAlertRule) []int {
	var userIDs []int

	// 从预警规则获取通知用户
	if len(rule.EscalationLevels) > 0 {
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

	if len(tickets) == 0 {
		return nil
	}

	// 批量加载已有的长等待预警记录，避免N+1查询
	ticketIDs := make([]int, len(tickets))
	for i, t := range tickets {
		ticketIDs[i] = t.ID
	}
	existingAlerts, _ := e.client.SLAAlertHistory.Query().
		Where(
			slaalerthistory.TicketIDIn(ticketIDs...),
			slaalerthistory.AlertLevelEQ("long_pending"),
			slaalerthistory.ResolvedAtIsNil(),
		).
		All(ctx)

	// 构建已发送预警的工单集合
	alertedTickets := make(map[int]bool)
	for _, alert := range existingAlerts {
		alertedTickets[alert.TicketID] = true
	}

	for _, t := range tickets {
		// 检查是否已发送过长时间未解决通知
		if alertedTickets[t.ID] {
			continue
		}

		if e.notificationSvc != nil {
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

			// H3 修复：发送通知后创建alert history记录，避免重复发送
			if _, err := e.client.SLAAlertHistory.Create().
				SetTicketID(t.ID).
				SetTicketNumber(t.TicketNumber).
				SetTicketTitle(t.Title).
				SetAlertLevel("long_pending").
				SetTenantID(tenantID).
				SetAlertRuleName(fmt.Sprintf("工单 #%s 已超过24小时未解决", t.TicketNumber)).
				Save(ctx); err != nil {
				e.logger.Errorw("Failed to create alert history", "ticket_id", t.ID, "error", err)
			}
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

	if len(tickets) == 0 {
		return nil
	}

	// 批量加载已有的未分配预警记录，避免N+1查询
	ticketIDs := make([]int, len(tickets))
	for i, t := range tickets {
		ticketIDs[i] = t.ID
	}
	existingAlerts, _ := e.client.SLAAlertHistory.Query().
		Where(
			slaalerthistory.TicketIDIn(ticketIDs...),
			slaalerthistory.AlertLevelEQ("unassigned"),
			slaalerthistory.ResolvedAtIsNil(),
		).
		All(ctx)

	// 构建已发送预警的工单集合
	alertedTickets := make(map[int]bool)
	for _, alert := range existingAlerts {
		alertedTickets[alert.TicketID] = true
	}

	// 通知管理员
	admins, _ := e.client.User.Query().
		Where(user.RoleEQ("admin")).
		IDs(ctx)

	for _, t := range tickets {
		// 检查是否已发送过未分配通知
		if alertedTickets[t.ID] {
			continue
		}

		if e.notificationSvc != nil && len(admins) > 0 {
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

			// H3 修复：发送通知后创建alert history记录，避免重复发送
			if _, err := e.client.SLAAlertHistory.Create().
				SetTicketID(t.ID).
				SetTicketNumber(t.TicketNumber).
				SetTicketTitle(t.Title).
				SetAlertLevel("unassigned").
				SetTenantID(tenantID).
				SetAlertRuleName(fmt.Sprintf("工单 #%s 已超过2小时未分配", t.TicketNumber)).
				Save(ctx); err != nil {
				e.logger.Errorw("Failed to create alert history for unassigned", "ticket_id", t.ID, "error", err)
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

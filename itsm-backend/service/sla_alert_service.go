package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/slaalerthistory"
	"itsm-backend/ent/slaalertrule"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/ticket"

	"go.uber.org/zap"
)

// defaultAlertCooldownMinutes 默认告警抑制间隔（分钟）
//
// 为避免高频扫描（默认 5 分钟一轮）在同一工单同一规则上重复触发告警，
// 同一 (ticket_id, alert_rule_id) 在 cooldown 窗口内只保留一条未解决的
// alert history。抑制间隔可通过 alert_rule.EscalationLevels JSON 中
// 第一个元素的 cooldown_minutes 字段覆盖（0 或负数表示禁用）。
const defaultAlertCooldownMinutes = 15

type SLAAlertService struct {
	client          *ent.Client
	logger          *zap.SugaredLogger
	notificationSvc *TicketNotificationService
}

func NewSLAAlertService(client *ent.Client, logger *zap.SugaredLogger) *SLAAlertService {
	return &SLAAlertService{
		client: client,
		logger: logger,
	}
}

// SetNotificationService 设置通知服务
func (s *SLAAlertService) SetNotificationService(notificationSvc *TicketNotificationService) {
	s.notificationSvc = notificationSvc
}

// CreateAlertRule 创建SLA预警规则
func (s *SLAAlertService) CreateAlertRule(ctx context.Context, req *dto.CreateSLAAlertRuleRequest, tenantID int) (*dto.SLAAlertRuleResponse, error) {
	s.logger.Infow("Creating SLA alert rule", "name", req.Name, "tenant_id", tenantID)

	// 验证SLA定义是否存在
	_, err := s.client.SLADefinition.Query().
		Where(sladefinition.IDEQ(req.SLADefinitionID), sladefinition.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("SLA definition not found: %w", err)
	}

	// 转换EscalationLevels为map格式
	escalationLevelsMap := make([]map[string]interface{}, len(req.EscalationLevels))
	for i, level := range req.EscalationLevels {
		escalationLevelsMap[i] = map[string]interface{}{
			"level":        level.Level,
			"threshold":    level.Threshold,
			"notify_users": level.NotifyUsers,
		}
	}

	alertRule, err := s.client.SLAAlertRule.Create().
		SetName(req.Name).
		SetSLADefinitionID(req.SLADefinitionID).
		SetAlertLevel(req.AlertLevel).
		SetThresholdPercentage(req.ThresholdPercentage).
		SetNotificationChannels(req.NotificationChannels).
		SetEscalationEnabled(req.EscalationEnabled).
		SetEscalationLevels(escalationLevelsMap).
		SetIsActive(req.IsActive).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create SLA alert rule", "error", err)
		return nil, fmt.Errorf("failed to create SLA alert rule: %w", err)
	}

	return s.toAlertRuleResponse(alertRule), nil
}

// UpdateAlertRule 更新SLA预警规则
func (s *SLAAlertService) UpdateAlertRule(ctx context.Context, id int, req *dto.UpdateSLAAlertRuleRequest, tenantID int) (*dto.SLAAlertRuleResponse, error) {
	s.logger.Infow("Updating SLA alert rule", "id", id, "tenant_id", tenantID)

	update := s.client.SLAAlertRule.Update().
		Where(
			slaalertrule.IDEQ(id),
			slaalertrule.TenantIDEQ(tenantID),
		).
		SetUpdatedAt(time.Now())

	if req.Name != nil {
		update = update.SetName(*req.Name)
	}
	if req.AlertLevel != nil {
		update = update.SetAlertLevel(*req.AlertLevel)
	}
	if req.ThresholdPercentage != nil {
		update = update.SetThresholdPercentage(*req.ThresholdPercentage)
	}
	if req.NotificationChannels != nil {
		update = update.SetNotificationChannels(*req.NotificationChannels)
	}
	if req.EscalationEnabled != nil {
		update = update.SetEscalationEnabled(*req.EscalationEnabled)
	}
	if req.EscalationLevels != nil {
		// 转换EscalationLevels为map格式
		escalationLevelsMap := make([]map[string]interface{}, len(*req.EscalationLevels))
		for i, level := range *req.EscalationLevels {
			escalationLevelsMap[i] = map[string]interface{}{
				"level":        level.Level,
				"threshold":    level.Threshold,
				"notify_users": level.NotifyUsers,
			}
		}
		update = update.SetEscalationLevels(escalationLevelsMap)
	}
	if req.IsActive != nil {
		update = update.SetIsActive(*req.IsActive)
	}

	_, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update SLA alert rule", "error", err)
		return nil, fmt.Errorf("failed to update SLA alert rule: %w", err)
	}

	// 重新获取更新后的规则
	alertRule, err := s.client.SLAAlertRule.Query().
		Where(
			slaalertrule.IDEQ(id),
			slaalertrule.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated alert rule: %w", err)
	}

	return s.toAlertRuleResponse(alertRule), nil
}

// DeleteAlertRule 删除SLA预警规则
func (s *SLAAlertService) DeleteAlertRule(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting SLA alert rule", "id", id, "tenant_id", tenantID)

	_, err := s.client.SLAAlertRule.Delete().
		Where(
			slaalertrule.IDEQ(id),
			slaalertrule.TenantIDEQ(tenantID),
		).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete SLA alert rule", "error", err)
		return fmt.Errorf("failed to delete SLA alert rule: %w", err)
	}

	return nil
}

// ListAlertRules 获取SLA预警规则列表
func (s *SLAAlertService) ListAlertRules(ctx context.Context, filters map[string]interface{}, tenantID int) ([]*dto.SLAAlertRuleResponse, error) {
	s.logger.Infow("Listing SLA alert rules", "tenant_id", tenantID)

	query := s.client.SLAAlertRule.Query().
		Where(slaalertrule.TenantIDEQ(tenantID))

	if slaDefinitionID, ok := filters["sla_definition_id"].(int); ok {
		query = query.Where(slaalertrule.SLADefinitionIDEQ(slaDefinitionID))
	}
	if isActive, ok := filters["is_active"].(bool); ok {
		query = query.Where(slaalertrule.IsActiveEQ(isActive))
	}
	if alertLevel, ok := filters["alert_level"].(string); ok && alertLevel != "" {
		query = query.Where(slaalertrule.AlertLevelEQ(alertLevel))
	}

	alertRules, err := query.Order(ent.Desc(slaalertrule.FieldCreatedAt)).All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list SLA alert rules", "error", err)
		return nil, fmt.Errorf("failed to list SLA alert rules: %w", err)
	}

	responses := make([]*dto.SLAAlertRuleResponse, len(alertRules))
	for i, rule := range alertRules {
		responses[i] = s.toAlertRuleResponse(rule)
	}

	return responses, nil
}

// GetAlertRule 获取SLA预警规则详情
func (s *SLAAlertService) GetAlertRule(ctx context.Context, id int, tenantID int) (*dto.SLAAlertRuleResponse, error) {
	s.logger.Infow("Getting SLA alert rule", "id", id, "tenant_id", tenantID)

	alertRule, err := s.client.SLAAlertRule.Query().
		Where(
			slaalertrule.IDEQ(id),
			slaalertrule.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get SLA alert rule", "error", err)
		return nil, fmt.Errorf("failed to get SLA alert rule: %w", err)
	}

	return s.toAlertRuleResponse(alertRule), nil
}

// GetAlertHistory 获取SLA预警历史
func (s *SLAAlertService) GetAlertHistory(ctx context.Context, req *dto.GetSLAAlertHistoryRequest, tenantID int) ([]*dto.SLAAlertHistoryResponse, int, error) {
	s.logger.Infow("Getting SLA alert history", "tenant_id", tenantID)

	query := s.client.SLAAlertHistory.Query().
		Where(slaalerthistory.TenantIDEQ(tenantID))

	if req.SLADefinitionID != nil {
		// 需要通过alert_rule关联查询
		alertRuleIDs, err := s.client.SLAAlertRule.Query().
			Where(
				slaalertrule.SLADefinitionIDEQ(*req.SLADefinitionID),
				slaalertrule.TenantIDEQ(tenantID),
			).
			IDs(ctx)
		if err == nil && len(alertRuleIDs) > 0 {
			query = query.Where(slaalerthistory.AlertRuleIDIn(alertRuleIDs...))
		}
	}
	if req.AlertRuleID != nil {
		query = query.Where(slaalerthistory.AlertRuleIDEQ(*req.AlertRuleID))
	}
	if req.TicketID != nil {
		query = query.Where(slaalerthistory.TicketIDEQ(*req.TicketID))
	}
	if req.AlertLevel != nil && *req.AlertLevel != "" {
		query = query.Where(slaalerthistory.AlertLevelEQ(*req.AlertLevel))
	}

	// 解析时间范围
	if req.StartTime != "" && req.EndTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err == nil {
			endTime, err := time.Parse(time.RFC3339, req.EndTime)
			if err == nil {
				query = query.Where(
					slaalerthistory.CreatedAtGTE(startTime),
					slaalerthistory.CreatedAtLTE(endTime),
				)
			}
		}
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count SLA alert history", "error", err)
		return nil, 0, fmt.Errorf("failed to count SLA alert history: %w", err)
	}

	// 分页查询
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	histories, err := query.
		Order(ent.Desc(slaalerthistory.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get SLA alert history", "error", err)
		return nil, 0, fmt.Errorf("failed to get SLA alert history: %w", err)
	}

	responses := make([]*dto.SLAAlertHistoryResponse, len(histories))
	for i, history := range histories {
		responses[i] = s.toAlertHistoryResponse(history)
	}

	return responses, total, nil
}

// CheckAndTriggerAlerts 检查并触发预警，返回是否触发了告警
func (s *SLAAlertService) CheckAndTriggerAlerts(ctx context.Context, ticketID int, tenantID int) (bool, error) {
	s.logger.Infow("Checking and triggering alerts", "ticket_id", ticketID, "tenant_id", tenantID)

	// 获取工单信息
	ticketEntity, err := s.client.Ticket.Query().
		Where(
			ticket.IDEQ(ticketID),
			ticket.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get ticket: %w", err)
	}

	if ticketEntity.SLADefinitionID == 0 {
		return false, nil // 没有SLA定义，无需检查
	}

	// 获取该SLA定义的所有活跃预警规则
	alertRules, err := s.client.SLAAlertRule.Query().
		Where(
			slaalertrule.SLADefinitionIDEQ(ticketEntity.SLADefinitionID),
			slaalertrule.TenantIDEQ(tenantID),
			slaalertrule.IsActiveEQ(true),
		).
		All(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get alert rules: %w", err)
	}

	if len(alertRules) == 0 {
		return false, nil
	}

	now := time.Now()
	var slaDeadline time.Time
	var timeRemaining float64
	alertTriggered := false

	// 检查响应时间预警
	if !ticketEntity.SLAResponseDeadline.IsZero() && ticketEntity.FirstResponseAt.IsZero() {
		slaDeadline = ticketEntity.SLAResponseDeadline
		timeRemaining = slaDeadline.Sub(now).Minutes()
		if timeRemaining > 0 {
			// 计算剩余时间百分比
			totalTime := slaDeadline.Sub(ticketEntity.CreatedAt).Minutes()
			if totalTime > 0 {
				percentage := (timeRemaining / totalTime) * 100
				triggered := s.checkAndCreateAlert(ctx, ticketEntity, alertRules, "response_time", percentage, tenantID)
				if triggered {
					alertTriggered = true
				}
			}
		}
	}

	// 检查解决时间预警
	if !ticketEntity.SLAResolutionDeadline.IsZero() && ticketEntity.ResolvedAt.IsZero() {
		slaDeadline = ticketEntity.SLAResolutionDeadline
		timeRemaining = slaDeadline.Sub(now).Minutes()
		if timeRemaining > 0 {
			// 计算剩余时间百分比
			totalTime := slaDeadline.Sub(ticketEntity.CreatedAt).Minutes()
			if totalTime > 0 {
				percentage := (timeRemaining / totalTime) * 100
				triggered := s.checkAndCreateAlert(ctx, ticketEntity, alertRules, "resolution_time", percentage, tenantID)
				if triggered {
					alertTriggered = true
				}
			}
		}
	}

	return alertTriggered, nil
}

// TriggerSLAWarning 触发SLA预警（基于默认阈值）
// 返回是否成功触发预警
func (s *SLAAlertService) TriggerSLAWarning(ctx context.Context, ticketID int, warningType string, tenantID int) (bool, error) {
	// 获取工单信息
	ticketEntity, err := s.client.Ticket.Query().
		Where(
			ticket.IDEQ(ticketID),
			ticket.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get ticket: %w", err)
	}

	if ticketEntity.SLADefinitionID == 0 {
		return false, nil
	}

	// 获取该SLA定义的所有活跃预警规则
	alertRules, err := s.client.SLAAlertRule.Query().
		Where(
			slaalertrule.SLADefinitionIDEQ(ticketEntity.SLADefinitionID),
			slaalertrule.TenantIDEQ(tenantID),
			slaalertrule.IsActiveEQ(true),
		).
		All(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get alert rules: %w", err)
	}

	if len(alertRules) == 0 {
		return false, nil
	}

	now := time.Now()
	var percentage float64
	var slaDeadline time.Time

	if warningType == "response_time" && !ticketEntity.SLAResponseDeadline.IsZero() && ticketEntity.FirstResponseAt.IsZero() {
		slaDeadline = ticketEntity.SLAResponseDeadline
		totalTime := slaDeadline.Sub(ticketEntity.CreatedAt).Minutes()
		timeRemaining := slaDeadline.Sub(now).Minutes()
		if totalTime > 0 && timeRemaining > 0 {
			percentage = (timeRemaining / totalTime) * 100
		}
	} else if warningType == "resolution_time" && !ticketEntity.SLAResolutionDeadline.IsZero() && ticketEntity.ResolvedAt.IsZero() {
		slaDeadline = ticketEntity.SLAResolutionDeadline
		totalTime := slaDeadline.Sub(ticketEntity.CreatedAt).Minutes()
		timeRemaining := slaDeadline.Sub(now).Minutes()
		if totalTime > 0 && timeRemaining > 0 {
			percentage = (timeRemaining / totalTime) * 100
		}
	}

	if percentage <= 0 {
		return false, nil
	}

	triggered := s.checkAndCreateAlert(ctx, ticketEntity, alertRules, warningType, percentage, tenantID)
	return triggered, nil
}

// checkAndCreateAlert 检查并创建预警记录，返回是否触发了告警
func (s *SLAAlertService) checkAndCreateAlert(ctx context.Context, ticketEntity *ent.Ticket, alertRules []*ent.SLAAlertRule, alertType string, percentage float64, tenantID int) bool {
	alertTriggered := false
	for _, rule := range alertRules {
		threshold := float64(rule.ThresholdPercentage)
		if percentage <= threshold {
			// 检查是否已存在未解决的预警
			exists, _ := s.client.SLAAlertHistory.Query().
				Where(
					slaalerthistory.TicketIDEQ(ticketEntity.ID),
					slaalerthistory.AlertRuleIDEQ(rule.ID),
					slaalerthistory.ResolvedAtIsNil(),
				).
				Exist(ctx)
			if exists {
				continue // 已存在预警，跳过
			}

			// Cooldown 抑制：避免同一 (ticket, rule) 在窗口内重复告警
			cooldownMin := resolveCooldownMinutes(rule)
			if cooldownMin > 0 {
				lastAlert, err := s.client.SLAAlertHistory.Query().
					Where(
						slaalerthistory.TicketIDEQ(ticketEntity.ID),
						slaalerthistory.AlertRuleIDEQ(rule.ID),
					).
					Order(ent.Desc(slaalerthistory.FieldCreatedAt)).
					First(ctx)
				if err == nil && lastAlert != nil {
					elapsed := time.Since(lastAlert.CreatedAt)
					if elapsed < time.Duration(cooldownMin)*time.Minute {
						remaining := time.Duration(cooldownMin)*time.Minute - elapsed
						s.logger.Infow(
							"SLA alert suppressed by cooldown",
							"ticket_id", ticketEntity.ID,
							"alert_rule_id", rule.ID,
							"alert_level", rule.AlertLevel,
							"alert_type", alertType,
							"cooldown_minutes", cooldownMin,
							"cooldown_remaining_seconds", int(remaining.Seconds()),
						)
						continue
					}
				}
			}

			// 创建预警历史记录
			alertHistory, err := s.client.SLAAlertHistory.Create().
				SetTicketID(ticketEntity.ID).
				SetTicketNumber(ticketEntity.TicketNumber).
				SetTicketTitle(ticketEntity.Title).
				SetAlertRuleID(rule.ID).
				SetAlertRuleName(rule.Name).
				SetAlertLevel(rule.AlertLevel).
				SetThresholdPercentage(rule.ThresholdPercentage).
				SetActualPercentage(percentage).
				SetNotificationSent(false).
				SetEscalationLevel(0).
				SetTenantID(tenantID).
				SetCreatedAt(time.Now()).
				Save(ctx)
			if err != nil {
				s.logger.Errorw("Failed to create alert history", "error", err)
				continue
			}

			// 发送预警通知
			if s.notificationSvc != nil {
				// 根据预警级别确定通知渠道
				alertLevel := rule.AlertLevel
				if alertLevel == "" {
					alertLevel = "warning"
				}

				// 发送站内通知
				if err := s.notificationSvc.NotifySLAAlertLevelChanged(ctx, ticketEntity.ID, alertLevel, percentage, tenantID); err != nil {
					s.logger.Warnw("failed to send SLA alert level change notification", "error", err, "ticket_id", ticketEntity.ID, "alert_level", alertLevel)
				}

				// 如果是严重级别，发送邮件通知
				if alertLevel == "critical" && s.notificationSvc.emailService != nil {
					// 获取处理人和创建人
					userIDs := []int{ticketEntity.RequesterID}
					if ticketEntity.AssigneeID > 0 {
						userIDs = append(userIDs, ticketEntity.AssigneeID)
					}

					var emails []string
					for _, userID := range userIDs {
						userEntity, _ := s.client.User.Get(ctx, userID)
						if userEntity != nil && userEntity.Email != "" {
							emails = append(emails, userEntity.Email)
						}
					}

					if len(emails) > 0 {
						content := fmt.Sprintf("【严重SLA预警】工单 #%s 剩余时间不足 %.1f%%，请立即处理！",
							ticketEntity.TicketNumber, percentage)
						if err := s.notificationSvc.emailService.SendTicketNotification(
							ctx, emails, ticketEntity.TicketNumber, ticketEntity.Title, "sla_alert", content,
						); err != nil {
							s.logger.Warnw("failed to send SLA critical alert email", "error", err, "ticket_id", ticketEntity.ID)
						}
					}
				}

				// 标记通知已发送
				if alertHistory != nil {
					if _, err := s.client.SLAAlertHistory.UpdateOneID(alertHistory.ID).
						SetNotificationSent(true).
						Save(ctx); err != nil {
						s.logger.Warnw("failed to mark alert notification as sent", "error", err, "alert_history_id", alertHistory.ID)
					}
				}
			}

			s.logger.Infow("SLA alert triggered", "ticket_id", ticketEntity.ID,
				"alert_rule", rule.Name, "level", rule.AlertLevel, "percentage", percentage)
			alertTriggered = true
		}
	}
	return alertTriggered
}

// 辅助方法：转换为响应DTO
func (s *SLAAlertService) toAlertRuleResponse(rule *ent.SLAAlertRule) *dto.SLAAlertRuleResponse {
	escalationLevels := make([]dto.EscalationLevelConfig, 0)
	if rule.EscalationLevels != nil {
		// EscalationLevels已经是[]map[string]interface{}类型
		for _, levelMap := range rule.EscalationLevels {
			config := dto.EscalationLevelConfig{}
			if levelVal, ok := levelMap["level"].(float64); ok {
				config.Level = int(levelVal)
			} else if levelVal, ok := levelMap["level"].(int); ok {
				config.Level = levelVal
			}
			if thresholdVal, ok := levelMap["threshold"].(float64); ok {
				config.Threshold = int(thresholdVal)
			} else if thresholdVal, ok := levelMap["threshold"].(int); ok {
				config.Threshold = thresholdVal
			}
			if notifyUsers, ok := levelMap["notify_users"].([]interface{}); ok {
				config.NotifyUsers = make([]int, 0, len(notifyUsers))
				for _, u := range notifyUsers {
					if userID, ok := u.(float64); ok {
						config.NotifyUsers = append(config.NotifyUsers, int(userID))
					} else if userID, ok := u.(int); ok {
						config.NotifyUsers = append(config.NotifyUsers, userID)
					}
				}
			}
			escalationLevels = append(escalationLevels, config)
		}
	}

	return &dto.SLAAlertRuleResponse{
		ID:                   rule.ID,
		Name:                 rule.Name,
		SLADefinitionID:      rule.SLADefinitionID,
		AlertLevel:           rule.AlertLevel,
		ThresholdPercentage:  rule.ThresholdPercentage,
		NotificationChannels: rule.NotificationChannels,
		EscalationEnabled:    rule.EscalationEnabled,
		EscalationLevels:     escalationLevels,
		IsActive:             rule.IsActive,
		TenantID:             rule.TenantID,
		CreatedAt:            rule.CreatedAt,
		UpdatedAt:            rule.UpdatedAt,
	}
}

func (s *SLAAlertService) toAlertHistoryResponse(history *ent.SLAAlertHistory) *dto.SLAAlertHistoryResponse {
	response := &dto.SLAAlertHistoryResponse{
		ID:                  history.ID,
		TicketID:            history.TicketID,
		TicketNumber:        history.TicketNumber,
		TicketTitle:         history.TicketTitle,
		AlertRuleID:         history.AlertRuleID,
		AlertRuleName:       history.AlertRuleName,
		AlertLevel:          history.AlertLevel,
		ThresholdPercentage: history.ThresholdPercentage,
		ActualPercentage:    history.ActualPercentage,
		NotificationSent:    history.NotificationSent,
		EscalationLevel:     history.EscalationLevel,
		CreatedAt:           history.CreatedAt,
	}

	if !history.ResolvedAt.IsZero() {
		response.ResolvedAt = &history.ResolvedAt
	}

	// Cooldown 信息：未解决的告警仍然处于冷却窗口时返回剩余秒数
	// 默认 cooldown 为 defaultAlertCooldownMinutes
	// （自定义场景需从 alert_rule.EscalationLevels JSON 读取，这里以默认值为准）
	response.CooldownMinutes = defaultAlertCooldownMinutes
	if history.ResolvedAt.IsZero() {
		elapsed := time.Since(history.CreatedAt)
		cooldownDur := time.Duration(defaultAlertCooldownMinutes) * time.Minute
		if elapsed < cooldownDur {
			remaining := cooldownDur - elapsed
			response.CooldownRemainingSeconds = int(remaining.Seconds())
			response.SuppressedByCooldown = true
		} else {
			response.CooldownRemainingSeconds = 0
			response.SuppressedByCooldown = false
		}
	}

	return response
}

// resolveCooldownMinutes 解析告警规则的 cooldown 配置
//
// 优先级（高到低）：
//  1. rule.EscalationLevels JSON 第一个元素的 cooldown_minutes 字段
//  2. 默认常量 defaultAlertCooldownMinutes
//
// 约定：cooldown_minutes <= 0 表示禁用抑制（生产环境慎用）。
func resolveCooldownMinutes(rule *ent.SLAAlertRule) int {
	if rule == nil {
		return defaultAlertCooldownMinutes
	}
	if len(rule.EscalationLevels) == 0 {
		return defaultAlertCooldownMinutes
	}
	first := rule.EscalationLevels[0]
	if first == nil {
		return defaultAlertCooldownMinutes
	}
	v, ok := first["cooldown_minutes"]
	if !ok {
		return defaultAlertCooldownMinutes
	}
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float32:
		return int(n)
	case float64:
		return int(n)
	}
	return defaultAlertCooldownMinutes
}

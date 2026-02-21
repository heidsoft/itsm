package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/slaviolation"
	"itsm-backend/ent/ticket"

	"go.uber.org/zap"
)

type SLAMonitorService struct {
	client          *ent.Client
	logger          *zap.SugaredLogger
	alertService    *SLAAlertService
	notificationSvc *TicketNotificationService
}

type SLAMetrics struct {
	ResponseTime       float64 `json:"response_time"`
	ResolutionTime     float64 `json:"resolution_time"`
	SLACompliance      float64 `json:"sla_compliance"`
	TotalTickets       int     `json:"total_tickets"`
	ViolatedTickets    int     `json:"violated_tickets"`
	AvgResponseMinutes float64 `json:"avg_response_minutes"`
	AvgResolutionHours float64 `json:"avg_resolution_hours"`
}

func NewSLAMonitorService(client *ent.Client, logger *zap.SugaredLogger) *SLAMonitorService {
	return &SLAMonitorService{
		client: client,
		logger: logger,
	}
}

// SetAlertService 设置告警服务
func (s *SLAMonitorService) SetAlertService(alertService *SLAAlertService) {
	s.alertService = alertService
}

// SetNotificationService 设置通知服务
func (s *SLAMonitorService) SetNotificationService(notificationSvc *TicketNotificationService) {
	s.notificationSvc = notificationSvc
}

type SLACheckStats struct {
	TotalChecked       int `json:"total_checked"`       // 检查的工单总数
	NewViolations      int `json:"new_violations"`      // 新创建的违规数
	ExistingViolations int `json:"existing_violations"` // 已存在的违规数
	WarningsTriggered  int `json:"warnings_triggered"`  // 触发的预警数
	AlertsTriggered    int `json:"alerts_triggered"`    // 触发的告警数
}

// CheckSLAViolations 检查所有工单的SLA违规情况
func (s *SLAMonitorService) CheckSLAViolations(ctx context.Context, tenantID int) (*SLACheckStats, error) {
	s.logger.Infow("Starting SLA violation check", "tenant_id", tenantID)

	now := time.Now()

	// 批量获取所有活跃工单（增加分页避免内存问题）
	stats := &SLACheckStats{}
	pageSize := 100
	offset := 0

	// 预加载该租户所有未解决的SLA违规，避免N+1查询
	existingViolations, err := s.client.SLAViolation.Query().
		Where(
			slaviolation.TenantIDEQ(tenantID),
			slaviolation.ResolvedAtIsNil(),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing violations: %w", err)
	}

	// 构建违规map快速查找: map[ticketID]map[violationType]bool
	existingViolationMap := make(map[int]map[string]bool)
	for _, v := range existingViolations {
		if existingViolationMap[v.TicketID] == nil {
			existingViolationMap[v.TicketID] = make(map[string]bool)
		}
		existingViolationMap[v.TicketID][v.ViolationType] = true
	}

	// 批量获取SLA定义
	slaDefinitions, err := s.client.SLADefinition.Query().
		Where(sladefinition.TenantIDEQ(tenantID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query SLA definitions: %w", err)
	}
	slaDefMap := make(map[int]string)
	for _, sd := range slaDefinitions {
		slaDefMap[sd.ID] = sd.Name
	}

	for {
		tickets, err := s.client.Ticket.Query().
			Where(
				ticket.TenantIDEQ(tenantID),
				ticket.SLADefinitionIDNEQ(0),
				ticket.ResolvedAtIsNil(),
			).
			Limit(pageSize).
			Offset(offset).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query tickets: %w", err)
		}

		if len(tickets) == 0 {
			break
		}

		stats.TotalChecked += len(tickets)

		for _, t := range tickets {
			// 检查是否需要发送预警（在SLA截止前）
			if s.alertService != nil {
				if warned := s.checkAndTriggerWarning(ctx, t, now); warned {
					stats.WarningsTriggered++
				}
			}

			// 检查响应时间SLA
			if t.FirstResponseAt.IsZero() && !t.SLAResponseDeadline.IsZero() && now.After(t.SLAResponseDeadline) {
				existingMap := existingViolationMap[t.ID]
				if existingMap == nil || !existingMap["response_time"] {
					// 新违规
					if err := s.createViolation(ctx, t, "response_time", t.SLAResponseDeadline, slaDefMap); err != nil {
						s.logger.Errorw("Failed to create response violation", "ticket_id", t.ID, "error", err)
					} else {
						stats.NewViolations++
						s.logger.Warnw("Ticket violated response SLA (new)", "ticket_id", t.ID, "ticket_number", t.TicketNumber)
					}
				} else {
					stats.ExistingViolations++
					s.logger.Debugw("Ticket already has response SLA violation", "ticket_id", t.ID, "ticket_number", t.TicketNumber)
				}
			}

			// 检查解决时间SLA
			if !t.SLAResolutionDeadline.IsZero() && now.After(t.SLAResolutionDeadline) {
				existingMap := existingViolationMap[t.ID]
				if existingMap == nil || !existingMap["resolution_time"] {
					// 新违规
					if err := s.createViolation(ctx, t, "resolution_time", t.SLAResolutionDeadline, slaDefMap); err != nil {
						s.logger.Errorw("Failed to create resolution violation", "ticket_id", t.ID, "error", err)
					} else {
						stats.NewViolations++
						s.logger.Warnw("Ticket violated resolution SLA (new)", "ticket_id", t.ID, "ticket_number", t.TicketNumber)
					}
				} else {
					stats.ExistingViolations++
					s.logger.Debugw("Ticket already has resolution SLA violation", "ticket_id", t.ID, "ticket_number", t.TicketNumber)
				}
			}

			// 检查告警阈值，触发告警
			if s.alertService != nil {
				if alerted, err := s.alertService.CheckAndTriggerAlerts(ctx, t.ID, tenantID); err != nil {
					s.logger.Errorw("Failed to trigger alerts", "ticket_id", t.ID, "error", err)
				} else if alerted {
					stats.AlertsTriggered++
				}
			}
		}

		offset += pageSize

		// 如果获取的记录少于pageSize，说明已经处理完毕
		if len(tickets) < pageSize {
			break
		}
	}

	s.logger.Infow("SLA violation check completed", "tenant_id", tenantID,
		"total_checked", stats.TotalChecked,
		"new_violations", stats.NewViolations,
		"existing_violations", stats.ExistingViolations,
		"warnings", stats.WarningsTriggered,
		"alerts", stats.AlertsTriggered)
	return stats, nil
}

// createViolation 创建SLA违规记录
// 注意: 已在调用方检查重复，此处不再检查
func (s *SLAMonitorService) createViolation(ctx context.Context, t *ent.Ticket, violationType string, deadline time.Time, slaDefMap map[int]string) error {
	// 计算超时时间（分钟）
	var exceededMinutes float64
	if violationType == "response_time" {
		exceededMinutes = time.Since(t.CreatedAt).Minutes() - deadline.Sub(t.CreatedAt).Minutes()
	} else {
		exceededMinutes = time.Since(t.CreatedAt).Minutes() - deadline.Sub(t.CreatedAt).Minutes()
	}

	// 描述信息
	description := fmt.Sprintf("工单 %s 违反SLA (%s): 超过截止时间 %.1f 分钟",
		t.TicketNumber, violationType, exceededMinutes)

	// 根据超时时间设置严重程度
	severity := "low"
	if exceededMinutes > 60 {
		severity = "medium"
	}
	if exceededMinutes > 240 {
		severity = "high"
	}
	if exceededMinutes > 480 {
		severity = "critical"
	}

	now := time.Now()
	// 如果没有 SLA 定义，跳过创建违规记录
	if t.SLADefinitionID == 0 {
		return nil
	}

	// 从预加载的map中获取SLA名称
	slaName := slaDefMap[t.SLADefinitionID]
	if slaName == "" {
		slaName = "Default SLA"
	}

	_, err := s.client.SLAViolation.Create().
		SetCreatedBy(0). // 系统自动创建，使用默认用户ID 0
		SetTicketID(t.ID).
		SetTicketType("ticket"). // Ticket 表没有类型字段，使用默认值
		SetSLADefinitionID(t.SLADefinitionID).
		SetSLAName(slaName).
		SetViolationType(violationType).
		SetViolationTime(now).
		SetDescription(description).
		SetSeverity(severity).
		SetIsResolved(false).
		SetTenantID(t.TenantID).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return err
	}

	// 发送SLA违规通知
	if s.notificationSvc != nil {
		_ = s.notificationSvc.NotifySLABreached(ctx, t.ID, violationType, exceededMinutes, t.TenantID)
	}

	s.logger.Infow("SLA violation created and notification sent", "ticket_id", t.ID,
		"violation_type", violationType, "exceeded_minutes", exceededMinutes)

	return nil
}

// checkAndTriggerWarning 检查是否需要发送SLA预警（在截止时间前触发）
// 返回是否发送了预警
func (s *SLAMonitorService) checkAndTriggerWarning(ctx context.Context, t *ent.Ticket, now time.Time) bool {
	// SLA预警阈值：默认在截止时间前20%时预警
	warningThreshold := 0.8

	sentWarning := false

	// 检查响应时间SLA预警
	if t.FirstResponseAt.IsZero() && !t.SLAResponseDeadline.IsZero() {
		totalDuration := t.SLAResponseDeadline.Sub(t.CreatedAt)
		elapsed := now.Sub(t.CreatedAt)
		progress := elapsed.Seconds() / totalDuration.Seconds()

		if progress >= warningThreshold && now.Before(t.SLAResponseDeadline) {
			if s.alertService != nil {
				if warned, _ := s.alertService.TriggerSLAWarning(ctx, t.ID, "response_time", t.TenantID); warned {
					sentWarning = true
					s.logger.Infow("SLA response warning sent", "ticket_id", t.ID, "ticket_number", t.TicketNumber,
						"deadline", t.SLAResponseDeadline)
				}
			}
		}
	}

	// 检查解决时间SLA预警
	if !t.SLAResolutionDeadline.IsZero() {
		totalDuration := t.SLAResolutionDeadline.Sub(t.CreatedAt)
		elapsed := now.Sub(t.CreatedAt)
		progress := elapsed.Seconds() / totalDuration.Seconds()

		if progress >= warningThreshold && now.Before(t.SLAResolutionDeadline) {
			if s.alertService != nil {
				if warned, _ := s.alertService.TriggerSLAWarning(ctx, t.ID, "resolution_time", t.TenantID); warned {
					sentWarning = true
					s.logger.Infow("SLA resolution warning sent", "ticket_id", t.ID, "ticket_number", t.TicketNumber,
						"deadline", t.SLAResolutionDeadline)
				}
			}
		}
	}

	return sentWarning
}

// CalculateSLAMetrics 计算SLA指标
func (s *SLAMonitorService) CalculateSLAMetrics(ctx context.Context, tenantID int, startTime, endTime time.Time) (*SLAMetrics, error) {
	s.logger.Infow("Calculating SLA metrics", "tenant_id", tenantID, "start_time", startTime, "end_time", endTime)

	metrics := &SLAMetrics{}

	// 获取时间范围内的工单
	tickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantIDEQ(tenantID),
			ticket.CreatedAtGTE(startTime),
			ticket.CreatedAtLTE(endTime),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query tickets: %w", err)
	}

	metrics.TotalTickets = len(tickets)

	var totalResponseMinutes float64
	var totalResolutionHours float64
	responseCount := 0
	resolutionCount := 0
	violatedCount := 0

	for _, t := range tickets {
		// 计算首次响应时间
		if !t.FirstResponseAt.IsZero() {
			responseMinutes := t.FirstResponseAt.Sub(t.CreatedAt).Minutes()
			totalResponseMinutes += responseMinutes
			responseCount++
		}

		// 计算解决时间
		if !t.ResolvedAt.IsZero() {
			resolutionHours := t.ResolvedAt.Sub(t.CreatedAt).Hours()
			totalResolutionHours += resolutionHours
			resolutionCount++

			// 检查是否违反SLA
			hasViolation, _ := s.client.SLAViolation.Query().
				Where(
					slaviolation.TicketID(t.ID),
					slaviolation.ResolvedAtIsNil(),
				).
				Exist(ctx)
			if hasViolation {
				violatedCount++
			}
		}
	}

	// 计算平均值
	if responseCount > 0 {
		metrics.AvgResponseMinutes = totalResponseMinutes / float64(responseCount)
	}
	if resolutionCount > 0 {
		metrics.AvgResolutionHours = totalResolutionHours / float64(resolutionCount)
	}

	// 计算SLA达成率
	if metrics.TotalTickets > 0 {
		metrics.SLACompliance = float64(metrics.TotalTickets-violatedCount) / float64(metrics.TotalTickets) * 100
	}

	metrics.ViolatedTickets = violatedCount

	s.logger.Infow("SLA metrics calculated",
		"tenant_id", tenantID,
		"compliance", metrics.SLACompliance,
		"avg_response", metrics.AvgResponseMinutes,
		"avg_resolution", metrics.AvgResolutionHours)

	return metrics, nil
}

// GetSLAComplianceByDefinition 获取按SLA定义的合规率统计
func (s *SLAMonitorService) GetSLAComplianceByDefinition(ctx context.Context, tenantID int) ([]*SLAComplianceStat, error) {
	slas, err := s.client.SLADefinition.Query().
		Where(sladefinition.TenantIDEQ(tenantID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var stats []*SLAComplianceStat
	for _, sla := range slas {
		// 获取该SLA的工单数量
		tickets, err := s.client.Ticket.Query().
			Where(
				ticket.TenantIDEQ(tenantID),
				ticket.SLADefinitionID(sla.ID),
			).
			All(ctx)
		if err != nil {
			continue
		}

		total := len(tickets)
		if total == 0 {
			continue
		}

		violated, _ := s.client.SLAViolation.Query().
			Where(
				slaviolation.SLADefinitionID(sla.ID),
				slaviolation.ResolvedAtIsNil(),
			).
			Count(ctx)

		stats = append(stats, &SLAComplianceStat{
			SLADefinitionID:   sla.ID,
			SLADefinitionName: sla.Name,
			TotalTickets:      total,
			ViolatedTickets:   violated,
			ComplianceRate:    float64(total-violated) / float64(total) * 100,
		})
	}

	return stats, nil
}

type SLAComplianceStat struct {
	SLADefinitionID   int     `json:"sla_definition_id"`
	SLADefinitionName string  `json:"sla_definition_name"`
	TotalTickets      int     `json:"total_tickets"`
	ViolatedTickets   int     `json:"violated_tickets"`
	ComplianceRate    float64 `json:"compliance_rate"`
}

// StartSLAWatcher 启动SLA定时检查任务
// interval: 检查间隔，默认5分钟
func (s *SLAMonitorService) StartSLAWatcher(ctx context.Context, interval time.Duration) {
	if interval == 0 {
		interval = 5 * time.Minute // 默认5分钟检查一次
	}

	s.logger.Infow("Starting SLA watcher", "interval", interval.String())

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("SLA watcher stopped")
			return
		case <-ticker.C:
			// 获取所有租户并检查SLA
			tenants, err := s.client.Tenant.Query().All(ctx)
			if err != nil {
				s.logger.Errorw("Failed to query tenants", "error", err)
				continue
			}

			for _, tenant := range tenants {
				if _, err := s.CheckSLAViolations(ctx, tenant.ID); err != nil {
					s.logger.Errorw("Failed to check SLA violations", "tenant_id", tenant.ID, "error", err)
				}
			}

			s.logger.Info("SLA watcher completed one round")
		}
	}
}

// CheckAllTenantsSLA 检查所有租户的SLA（用于定时任务调用）
func (s *SLAMonitorService) CheckAllTenantsSLA(ctx context.Context) error {
	tenants, err := s.client.Tenant.Query().All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query tenants: %w", err)
	}

	for _, tenant := range tenants {
		if _, err := s.CheckSLAViolations(ctx, tenant.ID); err != nil {
			s.logger.Errorw("Failed to check SLA violations", "tenant_id", tenant.ID, "error", err)
		}
	}

	return nil
}

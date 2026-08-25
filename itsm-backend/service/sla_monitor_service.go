package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/common/tenantctx"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/incident"
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
	ResponseTime       float64 `json:"responseTime"`
	ResolutionTime     float64 `json:"resolutionTime"`
	SLACompliance      float64 `json:"slaCompliance"`
	TotalTickets       int     `json:"totalTickets"`
	ViolatedTickets    int     `json:"violatedTickets"`
	AvgResponseMinutes float64 `json:"avgResponseMinutes"`
	AvgResolutionHours float64 `json:"avgResolutionHours"`
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
	TotalChecked       int `json:"totalChecked"`       // 检查的工单总数
	NewViolations      int `json:"newViolations"`      // 新创建的违规数
	ExistingViolations int `json:"existingViolations"` // 已存在的违规数
	WarningsTriggered  int `json:"warningsTriggered"`  // 触发的预警数
	AlertsTriggered    int `json:"alertsTriggered"`    // 触发的告警数
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
			slaviolation.IsResolvedEQ(false),
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
				ticket.DeletedAtIsNil(),
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
					// 乐观检查未命中：尝试创建。
					// 即使乐观检查在多 worker / 实例场景下产生“假命中”，createViolation
					// 内部还会通过事务内检查与数据库唯一约束再次去重，
					// 返回的 created 标志会准确反映“是否真的新增了一条记录”。
					created, cErr := s.createViolation(ctx, t, "response_time", t.SLAResponseDeadline, slaDefMap)
					if cErr != nil {
						s.logger.Errorw("Failed to create response violation", "ticket_id", t.ID, "error", cErr)
					} else if created {
						stats.NewViolations++
						s.logger.Warnw("Ticket violated response SLA (new)", "ticket_id", t.ID, "ticket_number", t.TicketNumber)
					} else {
						// 重复跳过：与现有违规的 stats 保持一致。
						stats.ExistingViolations++
						s.logger.Debugw("Ticket response SLA violation already exists, suppressed duplicate", "ticket_id", t.ID, "ticket_number", t.TicketNumber)
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
					created, cErr := s.createViolation(ctx, t, "resolution_time", t.SLAResolutionDeadline, slaDefMap)
					if cErr != nil {
						s.logger.Errorw("Failed to create resolution violation", "ticket_id", t.ID, "error", cErr)
					} else if created {
						stats.NewViolations++
						s.logger.Warnw("Ticket violated resolution SLA (new)", "ticket_id", t.ID, "ticket_number", t.TicketNumber)
					} else {
						stats.ExistingViolations++
						s.logger.Debugw("Ticket resolution SLA violation already exists, suppressed duplicate", "ticket_id", t.ID, "ticket_number", t.TicketNumber)
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

	// === Incident SLA 违规检查（P0-1）===
	incidentStats := s.checkIncidentSLAViolations(ctx, tenantID, now, slaDefMap)
	stats.TotalChecked += incidentStats.TotalChecked
	stats.NewViolations += incidentStats.NewViolations
	stats.ExistingViolations += incidentStats.ExistingViolations

	s.logger.Infow("SLA violation check completed", "tenant_id", tenantID,
		"total_checked", stats.TotalChecked,
		"new_violations", stats.NewViolations,
		"existing_violations", stats.ExistingViolations,
		"warnings", stats.WarningsTriggered,
		"alerts", stats.AlertsTriggered)
	return stats, nil
}

// checkIncidentSLAViolations 检查所有活跃 Incident 的 SLA 违规情况（P0-1）
func (s *SLAMonitorService) checkIncidentSLAViolations(ctx context.Context, tenantID int, now time.Time, slaDefMap map[int]string) *SLACheckStats {
	stats := &SLACheckStats{}

	// 预加载该租户 Incident 未解决的 SLA 违规
	existingViolations, err := s.client.SLAViolation.Query().
		Where(
			slaviolation.TenantIDEQ(tenantID),
			slaviolation.IsResolvedEQ(false),
			slaviolation.TicketTypeEQ("incident"),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to query existing incident violations", "error", err)
		return stats
	}

	existingViolationMap := make(map[int]map[string]bool)
	for _, v := range existingViolations {
		if existingViolationMap[v.TicketID] == nil {
			existingViolationMap[v.TicketID] = make(map[string]bool)
		}
		existingViolationMap[v.TicketID][v.ViolationType] = true
	}

	incidents, err := s.client.Incident.Query().
		Where(
			incident.TenantIDEQ(tenantID),
			incident.SLADefinitionIDNEQ(0),
			incident.ResolvedAtIsNil(),
			incident.DeletedAtIsNil(),
			incident.SLAStatusEQ("active"),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to query incidents for SLA check", "error", err)
		return stats
	}

	stats.TotalChecked = len(incidents)

	for _, inc := range incidents {
		// 响应时间 SLA
		if inc.SLAFirstResponseAt.IsZero() && !inc.SLAResponseDeadline.IsZero() && now.After(inc.SLAResponseDeadline) {
			existingMap := existingViolationMap[inc.ID]
			if existingMap == nil || !existingMap["response_time"] {
				if created, cErr := s.createIncidentViolation(ctx, inc, "response_time", inc.SLAResponseDeadline, slaDefMap); cErr != nil {
					s.logger.Errorw("Failed to create incident response violation", "incident_id", inc.ID, "error", cErr)
				} else if created {
					stats.NewViolations++
				} else {
					stats.ExistingViolations++
				}
			} else {
				stats.ExistingViolations++
			}
		}

		// 解决时间 SLA
		if !inc.SLAResolutionDeadline.IsZero() && now.After(inc.SLAResolutionDeadline) {
			existingMap := existingViolationMap[inc.ID]
			if existingMap == nil || !existingMap["resolution_time"] {
				if created, cErr := s.createIncidentViolation(ctx, inc, "resolution_time", inc.SLAResolutionDeadline, slaDefMap); cErr != nil {
					s.logger.Errorw("Failed to create incident resolution violation", "incident_id", inc.ID, "error", cErr)
				} else if created {
					stats.NewViolations++
				} else {
					stats.ExistingViolations++
				}
			} else {
				stats.ExistingViolations++
			}
		}
	}

	return stats
}

// createIncidentViolation 为 Incident 创建 SLA 违规记录（P0-1）
func (s *SLAMonitorService) createIncidentViolation(ctx context.Context, inc *ent.Incident, violationType string, deadline time.Time, slaDefMap map[int]string) (bool, error) {
	exceededMinutes := time.Since(deadline).Minutes()
	if exceededMinutes < 0 {
		exceededMinutes = 0
	}

	description := fmt.Sprintf("事件 %s 违反SLA (%s): 超过截止时间 %.1f 分钟",
		inc.IncidentNumber, violationType, exceededMinutes)

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
	if inc.SLADefinitionID == 0 {
		return false, nil
	}

	slaName := slaDefMap[inc.SLADefinitionID]
	if slaName == "" {
		slaName = "Default SLA"
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return false, fmt.Errorf("begin incident SLA violation tx: %w", err)
	}

	// 幂等检查
	exists, err := tx.SLAViolation.Query().
		Where(
			slaviolation.TicketIDEQ(inc.ID),
			slaviolation.TenantIDEQ(inc.TenantID),
			slaviolation.ViolationTypeEQ(violationType),
			slaviolation.IsResolvedEQ(false),
			slaviolation.TicketTypeEQ("incident"),
		).
		Exist(ctx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			s.logger.Errorw("failed to rollback tx", "error", rbErr)
		}
		return false, fmt.Errorf("check existing incident violation: %w", err)
	}
	if exists {
		if rbErr := tx.Rollback(); rbErr != nil {
			s.logger.Errorw("failed to rollback tx", "error", rbErr)
		}
		return false, nil
	}

	_, err = tx.SLAViolation.Create().
		SetCreatedBy(0).
		SetTicketID(inc.ID).
		SetTicketType("incident").
		SetSLADefinitionID(inc.SLADefinitionID).
		SetSLAName(slaName).
		SetViolationType(violationType).
		SetViolationTime(now).
		SetDescription(description).
		SetSeverity(severity).
		SetIsResolved(false).
		SetTenantID(inc.TenantID).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			s.logger.Errorw("failed to rollback tx", "error", rbErr)
		}
		if ent.IsConstraintError(err) {
			return false, nil
		}
		return false, fmt.Errorf("create incident SLA violation: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit incident SLA violation: %w", err)
	}

	s.logger.Infow("Incident SLA violation created", "incident_id", inc.ID,
		"violation_type", violationType, "exceeded_minutes", exceededMinutes)
	return true, nil
}

// createViolation 创建SLA违规记录
// 跨实例竞态保护（issue #85）：
// CheckSLAViolations 内部预加载 existingViolationMap 仅是“乐观检查”；真正
// 的互斥由以下两层保证：
//  1. 事务内“再查一次”，保证普通重试幂等
//  2. 数据库上的部分唯一索引
//     (ticket_id, violation_type) WHERE is_resolved = false，收口跨实例竞态
//
// 同时，本函数会明确区分「插入成功」与「重复跳过」两种结果，仅在插入成功时才
// 提交事务 + 发送通知，重复路径下会主动中止事务以避免重复入箱。
func (s *SLAMonitorService) createViolation(ctx context.Context, t *ent.Ticket, violationType string, deadline time.Time, slaDefMap map[int]string) (created bool, err error) {
	// 计算超时时间（分钟）：从 deadline 到当前时间的差值
	// response_time / resolution_time 的差异在于 deadline 语义不同，
	// 由调用方决定传入哪种 deadline；这里的超时时间计算逻辑一致。
	exceededMinutes := time.Since(deadline).Minutes()
	if exceededMinutes < 0 {
		exceededMinutes = 0
	}
	_ = violationType // 保留以便日志中区分

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
		return false, nil
	}

	// 从预加载的map中获取SLA名称
	slaName := slaDefMap[t.SLADefinitionID]
	if slaName == "" {
		slaName = "Default SLA"
	}

	// 在事务中创建违规与入箱通知，以保证同生同死。
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return false, fmt.Errorf("begin SLA violation transaction: %w", err)
	}
	commitAndNotify := func() (bool, error) {
		if s.notificationSvc != nil {
			if err := s.notificationSvc.NotifySLABreachedTx(ctx, tx, t.ID, violationType, exceededMinutes, t.TenantID); err != nil {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					s.logger.Errorw("failed to rollback SLA violation transaction", "error", rollbackErr)
				}
				return false, fmt.Errorf("enqueue SLA breach notification: %w", err)
			}
		}
		if err := tx.Commit(); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Errorw("failed to rollback SLA violation transaction", "error", rollbackErr)
			}
			return false, fmt.Errorf("commit SLA violation notification transaction: %w", err)
		}
		return true, nil
	}

	// 违规记录和通知必须共用同一个 Ent 事务。不能借用全局 rawDB 插入，
	// 否则通知入箱失败时违规记录无法回滚，会破坏 outbox 的原子性。
	exists, err := tx.SLAViolation.Query().
		Where(
			slaviolation.TicketIDEQ(t.ID),
			slaviolation.TenantIDEQ(t.TenantID),
			slaviolation.ViolationTypeEQ(violationType),
			slaviolation.IsResolvedEQ(false),
		).
		Exist(ctx)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			s.logger.Errorw("failed to rollback SLA violation transaction", "error", rollbackErr)
		}
		return false, fmt.Errorf("check existing SLA violation in tx: %w", err)
	}
	if exists {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			s.logger.Errorw("failed to rollback SLA violation transaction", "error", rollbackErr)
		}
		s.logger.Infow("SLA violation already exists, skipped duplicate insert (tx path)",
			"ticket_id", t.ID, "violation_type", violationType)
		return false, nil
	}

	if _, err := tx.SLAViolation.Create().
		SetCreatedBy(0).
		SetTicketID(t.ID).
		SetTicketType("ticket").
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
		Save(ctx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			s.logger.Errorw("failed to rollback SLA violation transaction", "error", rollbackErr)
		}
		if ent.IsConstraintError(err) {
			return false, nil
		}
		return false, fmt.Errorf("create SLA violation: %w", err)
	}

	if committed, cErr := commitAndNotify(); !committed {
		return false, cErr
	}

	s.logger.Infow("SLA violation created and notification enqueued", "ticket_id", t.ID,
		"violation_type", violationType, "exceeded_minutes", exceededMinutes)

	return true, nil
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
			ticket.DeletedAtIsNil(),
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

			// 检查是否违反SLA（工单已解决，但只要有违规记录即算违规，不要求违规记录未解决）
			hasViolation, _ := s.client.SLAViolation.Query().
				Where(
					slaviolation.TicketID(t.ID),
					slaviolation.TenantIDEQ(tenantID),
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
				ticket.DeletedAtIsNil(),
			).
			All(ctx)
		if err != nil {
			continue
		}

		total := len(tickets)
		if total == 0 {
			continue
		}

		// 统计有SLA违规记录的工单数量（去重）
		// 注意：不应使用 slaviolation.ResolvedAtIsNil() 过滤，
		// 因为违规记录的 resolved_at 表示违规是否已处理，
		// 与工单是否仍未解决是两个语义。
		// 只要工单有过违规记录，就应计入违规数。
		violated, _ := s.client.SLAViolation.Query().
			Where(
				slaviolation.SLADefinitionID(sla.ID),
				slaviolation.TenantIDEQ(tenantID),
			).
			Select(slaviolation.FieldTicketID).
			Unique(true).
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
	SLADefinitionID   int     `json:"slaDefinitionId"`
	SLADefinitionName string  `json:"slaDefinitionName"`
	TotalTickets      int     `json:"totalTickets"`
	ViolatedTickets   int     `json:"violatedTickets"`
	ComplianceRate    float64 `json:"complianceRate"`
}

// StartSLAWatcher 启动SLA定时检查任务
// interval: 检查间隔，默认5分钟
//
// RLS 说明：本 watcher 需扫描全租户 tenant 表并逐租户 CheckSLAViolations。
// 顶层 loop 用 SystemContext 豁免（枚举 tenant 是跨租户操作），
// 但每个租户的实际 SLA 检查会用 WithTenantID(ctx, tenant.ID) 收窄到该租户。
func (s *SLAMonitorService) StartSLAWatcher(ctx context.Context, interval time.Duration) {
	if interval == 0 {
		interval = 5 * time.Minute // 默认5分钟检查一次
	}

	s.logger.Infow("Starting SLA watcher", "interval", interval.String())

	// RLS：顶层 goroutine 明确标注为 system operation
	ctx = tenantctx.SystemContext(ctx, "sla_monitor:watch",
		"scan all tenants for SLA violations at tick interval")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("SLA watcher stopped")
			return
		case <-ticker.C:
			// 获取所有租户并检查SLA（bypass 已生效，可跨租户查 tenants）
			tenants, err := s.client.Tenant.Query().All(ctx)
			if err != nil {
				s.logger.Errorw("Failed to query tenants", "error", err)
				continue
			}

			for _, tenant := range tenants {
				// RLS：切到具体租户上下文，走 policy 正常过滤
				tenantCtx := tenantctx.WithTenantID(ctx, tenant.ID)
				if _, err := s.CheckSLAViolations(tenantCtx, tenant.ID); err != nil {
					s.logger.Errorw("Failed to check SLA violations", "tenant_id", tenant.ID, "error", err)
				}
			}

			s.logger.Info("SLA watcher completed one round")
		}
	}
}

// PauseSLA 暂停工单/事件的SLA计时（P0-2）
func (s *SLAMonitorService) PauseSLA(ctx context.Context, tenantID int, entityType string, entityID int, reason string) error {
	now := time.Now()

	switch entityType {
	case "ticket":
		t, err := s.client.Ticket.Query().
			Where(ticket.IDEQ(entityID), ticket.TenantIDEQ(tenantID), ticket.DeletedAtIsNil()).
			Only(ctx)
		if err != nil {
			return fmt.Errorf("查询工单失败: %w", err)
		}
		if t.SLAStatus == "paused" {
			return fmt.Errorf("工单SLA已处于暂停状态")
		}
		return s.client.Ticket.UpdateOneID(entityID).
			SetSLAStatus("paused").
			SetSLAPausedAt(now).
			SetSLAPauseReason(reason).
			Exec(ctx)

	case "incident":
		inc, err := s.client.Incident.Query().
			Where(incident.IDEQ(entityID), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
			Only(ctx)
		if err != nil {
			return fmt.Errorf("查询事件失败: %w", err)
		}
		if inc.SLAStatus == "paused" {
			return fmt.Errorf("事件SLA已处于暂停状态")
		}
		return s.client.Incident.UpdateOneID(entityID).
			SetSLAStatus("paused").
			SetSLAPausedAt(now).
			SetSLAPauseReason(reason).
			Exec(ctx)

	default:
		return fmt.Errorf("不支持的实体类型: %s", entityType)
	}
}

// ResumeSLA 恢复工单/事件的SLA计时，将暂停时长追加到deadline（P0-2）
func (s *SLAMonitorService) ResumeSLA(ctx context.Context, tenantID int, entityType string, entityID int) error {
	now := time.Now()

	switch entityType {
	case "ticket":
		t, err := s.client.Ticket.Query().
			Where(ticket.IDEQ(entityID), ticket.TenantIDEQ(tenantID), ticket.DeletedAtIsNil()).
			Only(ctx)
		if err != nil {
			return fmt.Errorf("查询工单失败: %w", err)
		}
		if t.SLAStatus != "paused" {
			return fmt.Errorf("工单SLA未处于暂停状态")
		}

		// 计算暂停时长，延长deadline
		updater := s.client.Ticket.UpdateOneID(entityID).
			SetSLAStatus("active").
			ClearSLAPausedAt().
			ClearSLAPauseReason()

		if !t.SLAPausedAt.IsZero() {
			pausedDuration := now.Sub(t.SLAPausedAt)
			if !t.SLAResponseDeadline.IsZero() {
				updater.SetSLAResponseDeadline(t.SLAResponseDeadline.Add(pausedDuration))
			}
			if !t.SLAResolutionDeadline.IsZero() {
				updater.SetSLAResolutionDeadline(t.SLAResolutionDeadline.Add(pausedDuration))
			}
		}
		return updater.Exec(ctx)

	case "incident":
		inc, err := s.client.Incident.Query().
			Where(incident.IDEQ(entityID), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
			Only(ctx)
		if err != nil {
			return fmt.Errorf("查询事件失败: %w", err)
		}
		if inc.SLAStatus != "paused" {
			return fmt.Errorf("事件SLA未处于暂停状态")
		}

		updater := s.client.Incident.UpdateOneID(entityID).
			SetSLAStatus("active").
			ClearSLAPausedAt().
			ClearSLAPauseReason()

		if !inc.SLAPausedAt.IsZero() {
			pausedDuration := now.Sub(inc.SLAPausedAt)
			if !inc.SLAResponseDeadline.IsZero() {
				updater.SetSLAResponseDeadline(inc.SLAResponseDeadline.Add(pausedDuration))
			}
			if !inc.SLAResolutionDeadline.IsZero() {
				updater.SetSLAResolutionDeadline(inc.SLAResolutionDeadline.Add(pausedDuration))
			}
		}
		return updater.Exec(ctx)

	default:
		return fmt.Errorf("不支持的实体类型: %s", entityType)
	}
}

// CheckAllTenantsSLA 检查所有租户的SLA（用于定时任务调用）
//
// RLS 说明：入口是跨租户操作，必须 system-bypass。逐租户执行时切回 tenant 上下文。
func (s *SLAMonitorService) CheckAllTenantsSLA(ctx context.Context) error {
	ctx = tenantctx.SystemContext(ctx, "sla_monitor:check_all",
		"one-shot SLA violation scan across all tenants")

	tenants, err := s.client.Tenant.Query().All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query tenants: %w", err)
	}

	for _, tenant := range tenants {
		tenantCtx := tenantctx.WithTenantID(ctx, tenant.ID)
		if _, err := s.CheckSLAViolations(tenantCtx, tenant.ID); err != nil {
			s.logger.Errorw("Failed to check SLA violations", "tenant_id", tenant.ID, "error", err)
		}
	}

	return nil
}

// GetDashboardMetrics 获取SLA监控仪表板完整指标
func (s *SLAMonitorService) GetDashboardMetrics(ctx context.Context, tenantID int) (*dto.SLAMonitoringDashboard, error) {
	s.logger.Infow("Getting SLA dashboard metrics", "tenant_id", tenantID)

	now := time.Now()
	dashboard := &dto.SLAMonitoringDashboard{
		UpcomingDeadlines: make([]dto.SLADeadline, 0),
		TopViolations:     make([]dto.SLAViolationItem, 0),
		SLAByPriority:     make(map[string]float64),
		TrendData:         make([]dto.SLATrendPoint, 0),
	}

	// 获取所有活跃工单（带有SLA定义的）
	tickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantIDEQ(tenantID),
			ticket.SLADefinitionIDNEQ(0),
			ticket.DeletedAtIsNil(),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query tickets: %w", err)
	}

	dashboard.TotalTickets = len(tickets)

	// 获取未解决的违规
	violations, err := s.client.SLAViolation.Query().
		Where(
			slaviolation.TenantIDEQ(tenantID),
			slaviolation.ResolvedAtIsNil(),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query violations: %w", err)
	}

	violationMap := make(map[int][]*ent.SLAViolation)
	for _, v := range violations {
		violationMap[v.TicketID] = append(violationMap[v.TicketID], v)
	}

	// 遍历工单进行分类统计
	atRiskCount := 0
	breachedCount := 0
	priorityMap := make(map[string]int)
	priorityViolationMap := make(map[string]int)

	for _, t := range tickets {
		ticketViolations := violationMap[t.ID]
		hasViolation := len(ticketViolations) > 0

		// 按优先级统计
		priority := t.Priority
		if priority == "" {
			priority = "unknown"
		}
		priorityMap[priority]++

		if hasViolation {
			breachedCount++
			priorityViolationMap[priority]++
		} else if !t.FirstResponseAt.IsZero() || (!t.SLAResponseDeadline.IsZero() && now.After(t.SLAResponseDeadline)) {
			// 检查是否处于风险中（接近SLA截止时间）
			if !t.SLAResponseDeadline.IsZero() && now.Before(t.SLAResponseDeadline) {
				timeLeft := t.SLAResponseDeadline.Sub(now)
				if timeLeft <= 30*time.Minute {
					atRiskCount++
				}
			}
			if !t.SLAResolutionDeadline.IsZero() && now.Before(t.SLAResolutionDeadline) {
				timeLeft := t.SLAResolutionDeadline.Sub(now)
				if timeLeft <= 30*time.Minute {
					atRiskCount++
				}
			}
		}
	}

	dashboard.AtRiskTickets = atRiskCount
	dashboard.BreachedTickets = breachedCount

	// 计算合规率和违规率
	if dashboard.TotalTickets > 0 {
		compliantCount := dashboard.TotalTickets - breachedCount
		dashboard.ComplianceRate = float64(compliantCount) / float64(dashboard.TotalTickets) * 100
		dashboard.ViolationRate = float64(breachedCount) / float64(dashboard.TotalTickets) * 100
	}

	// 按优先级计算SLA达成率
	for priority, total := range priorityMap {
		violated := priorityViolationMap[priority]
		if total > 0 {
			dashboard.SLAByPriority[priority] = float64(total-violated) / float64(total) * 100
		}
	}

	// 获取即将到期的工单（未来24小时内）
	upcomingDeadline := now.Add(24 * time.Hour)
	upcomingTickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantIDEQ(tenantID),
			ticket.SLADefinitionIDNEQ(0),
			ticket.ResolvedAtIsNil(),
			ticket.SLAResolutionDeadlineGT(now),
			ticket.SLAResolutionDeadlineLT(upcomingDeadline),
		).
		All(ctx)
	if err == nil {
		for _, t := range upcomingTickets {
			timeLeft := time.Until(t.SLAResolutionDeadline)
			timeLeftStr := formatDuration(timeLeft)

			slaName := "Default SLA"
			if t.SLADefinitionID != 0 {
				slaDef, err := s.client.SLADefinition.Get(ctx, t.SLADefinitionID)
				if err == nil && slaDef != nil {
					slaName = slaDef.Name
				}
			}

			dashboard.UpcomingDeadlines = append(dashboard.UpcomingDeadlines, dto.SLADeadline{
				TicketID:    t.ID,
				TicketTitle: t.Title,
				Deadline:    t.SLAResolutionDeadline,
				SLAPolicy:   slaName,
				TimeLeft:    timeLeftStr,
			})
		}
	}

	// 获取最新的违规记录作为Top Violations
	recentViolations, err := s.client.SLAViolation.Query().
		Where(
			slaviolation.TenantIDEQ(tenantID),
			slaviolation.ResolvedAtIsNil(),
		).
		Order(ent.Desc(slaviolation.FieldViolationTime)).
		Limit(10).
		All(ctx)
	if err == nil {
		for _, v := range recentViolations {
			ticket, err := s.client.Ticket.Get(ctx, v.TicketID)
			ticketTitle := fmt.Sprintf("Ticket #%d", v.TicketID)
			if err == nil && ticket != nil {
				ticketTitle = ticket.Title
			}

			// 计算延迟分钟数
			delayMinutes := 0
			if !v.ViolationTime.IsZero() {
				delayMinutes = int(time.Since(v.ViolationTime).Minutes())
			}

			dashboard.TopViolations = append(dashboard.TopViolations, dto.SLAViolationItem{
				TicketID:    v.TicketID,
				TicketTitle: ticketTitle,
				SLAPolicy:   v.SLAName,
				ViolatedAt:  v.ViolationTime.Format(time.RFC3339),
				Delay:       delayMinutes,
			})
		}
	}

	// 生成趋势数据（最近7天）
	for i := 6; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		startOfDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		dayTickets, err := s.client.Ticket.Query().
			Where(
				ticket.TenantIDEQ(tenantID),
				ticket.CreatedAtGTE(startOfDay),
				ticket.CreatedAtLT(endOfDay),
			).
			All(ctx)
		if err != nil {
			continue
		}

		dayViolations, _ := s.client.SLAViolation.Query().
			Where(
				slaviolation.TenantIDEQ(tenantID),
				slaviolation.ViolationTimeGTE(startOfDay),
				slaviolation.ViolationTimeLT(endOfDay),
			).
			Count(ctx)

		ticketCount := len(dayTickets)
		complianceRate := 100.0
		if ticketCount > 0 {
			compliant := ticketCount - dayViolations
			complianceRate = float64(compliant) / float64(ticketCount) * 100
		}

		dashboard.TrendData = append(dashboard.TrendData, dto.SLATrendPoint{
			Date:           startOfDay.Format("2006-01-02"),
			ComplianceRate: complianceRate,
			TicketCount:    ticketCount,
		})
	}

	s.logger.Infow("SLA dashboard metrics generated",
		"tenant_id", tenantID,
		"total_tickets", dashboard.TotalTickets,
		"compliance_rate", dashboard.ComplianceRate,
		"breached_tickets", dashboard.BreachedTickets)

	return dashboard, nil
}

// formatDuration 格式化时间间隔为可读字符串
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "overdue"
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

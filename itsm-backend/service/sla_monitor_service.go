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
	client            *ent.Client
	logger            *zap.SugaredLogger
	alertService      *SLAAlertService
	notificationSvc   *TicketNotificationService
}

type SLAMetrics struct {
	ResponseTime        float64 `json:"response_time"`
	ResolutionTime      float64 `json:"resolution_time"`
	SLACompliance       float64 `json:"sla_compliance"`
	TotalTickets        int     `json:"total_tickets"`
	ViolatedTickets     int     `json:"violated_tickets"`
	AvgResponseMinutes  float64 `json:"avg_response_minutes"`
	AvgResolutionHours  float64 `json:"avg_resolution_hours"`
}

func NewSLAMonitorService(client *ent.Client, logger *zap.SugaredLogger) *SLAMonitorService {
	return &SLAMonitorService{
		client:          client,
		logger:          logger,
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

// CheckSLAViolations 检查所有工单的SLA违规情况
func (s *SLAMonitorService) CheckSLAViolations(ctx context.Context, tenantID int) error {
	s.logger.Infow("Checking SLA violations", "tenant_id", tenantID)

	now := time.Now()

	// 获取所有活跃工单（有SLA定义但未解决）
	tickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantIDEQ(tenantID),
			ticket.SLADefinitionIDNEQ(0),
			ticket.ResolvedAtIsNil(),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query tickets: %w", err)
	}

	violationCount := 0
	for _, t := range tickets {
		// 检查响应时间SLA
		if t.FirstResponseAt.IsZero() && !t.SLAResponseDeadline.IsZero() && now.After(t.SLAResponseDeadline) {
			if err := s.createViolation(ctx, t, "response_time", t.SLAResponseDeadline); err != nil {
				s.logger.Errorw("Failed to create response violation", "ticket_id", t.ID, "error", err)
			} else {
				violationCount++
				s.logger.Warnw("Ticket violated response SLA", "ticket_id", t.ID, "ticket_number", t.TicketNumber)
			}
		}

		// 检查解决时间SLA
		if !t.SLAResolutionDeadline.IsZero() && now.After(t.SLAResolutionDeadline) {
			if err := s.createViolation(ctx, t, "resolution_time", t.SLAResolutionDeadline); err != nil {
				s.logger.Errorw("Failed to create resolution violation", "ticket_id", t.ID, "error", err)
			} else {
				violationCount++
				s.logger.Warnw("Ticket violated resolution SLA", "ticket_id", t.ID, "ticket_number", t.TicketNumber)
			}
		}

		// 检查告警阈值，触发预警
		if s.alertService != nil {
			if err := s.alertService.CheckAndTriggerAlerts(ctx, t.ID, tenantID); err != nil {
				s.logger.Errorw("Failed to trigger alerts", "ticket_id", t.ID, "error", err)
			}
		}
	}

	s.logger.Infow("SLA violation check completed", "tenant_id", tenantID, "violations_found", violationCount)
	return nil
}

// createViolation 创建SLA违规记录
func (s *SLAMonitorService) createViolation(ctx context.Context, t *ent.Ticket, violationType string, deadline time.Time) error {
	// 检查是否已存在相同类型的违规记录
	exists, err := s.client.SLAViolation.Query().
		Where(
			slaviolation.TicketID(t.ID),
			slaviolation.ViolationType(violationType),
			slaviolation.ResolvedAtIsNil(),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil // 已存在未解决的违规
	}

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

	_, err = s.client.SLAViolation.Create().
		SetTicketID(t.ID).
		SetSLADefinitionID(t.SLADefinitionID).
		SetViolationType(violationType).
		SetViolationTime(time.Now()).
		SetDescription(description).
		SetSeverity(severity).
		SetIsResolved(false).
		SetTenantID(t.TenantID).
		Save(ctx)
	return err
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
	SLADefinitionID    int     `json:"sla_definition_id"`
	SLADefinitionName  string  `json:"sla_definition_name"`
	TotalTickets       int     `json:"total_tickets"`
	ViolatedTickets    int     `json:"violated_tickets"`
	ComplianceRate     float64 `json:"compliance_rate"`
}

package sla

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"itsm-backend/dto"
)

type Service struct {
	repo   Repository
	logger *zap.SugaredLogger
}

func NewService(repo Repository, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// SLADefinition CRUD

func (s *Service) CreateDefinition(ctx context.Context, def *SLADefinition) (*SLADefinition, error) {
	s.logger.Infow("Creating SLA Definition", "name", def.Name)
	return s.repo.CreateDefinition(ctx, def)
}

func (s *Service) GetDefinition(ctx context.Context, id int, tenantID int) (*SLADefinition, error) {
	return s.repo.GetDefinition(ctx, id, tenantID)
}

func (s *Service) ListDefinitions(ctx context.Context, tenantID int, page, size int) ([]*SLADefinition, int, error) {
	return s.repo.ListDefinitions(ctx, tenantID, page, size)
}

func (s *Service) UpdateDefinition(ctx context.Context, def *SLADefinition) (*SLADefinition, error) {
	s.logger.Infow("Updating SLA Definition", "id", def.ID)
	return s.repo.UpdateDefinition(ctx, def)
}

func (s *Service) DeleteDefinition(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting SLA Definition", "id", id)
	return s.repo.DeleteDefinition(ctx, id, tenantID)
}

// SLA Compliance and Monitoring

// SLAComplianceResult 返回值
type SLAComplianceResult struct {
	TicketID                int        `json:"ticketId"`
	TenantID                int        `json:"tenantId"`
	Found                   bool       `json:"found"`
	CreatedAt               *time.Time `json:"createdAt,omitempty"`
	FirstResponseAt         *time.Time `json:"firstResponseAt,omitempty"`
	ResolvedAt              *time.Time `json:"resolvedAt,omitempty"`
	ActualResponseMinutes   float64    `json:"actualResponseMinutes"`
	ActualResolutionMinutes float64    `json:"actualResolutionMinutes"`
	Compliant               bool       `json:"compliant"`
	Message                 string     `json:"message"`
}

// CheckSLACompliance P1-07 修复：真正计算 actual_response_minutes。
func (s *Service) CheckSLACompliance(ctx context.Context, ticketID int, tenantID int) (*SLAComplianceResult, error) {
	s.logger.Infow("Checking SLA Compliance", "ticketID", ticketID)
	createdAt, firstResponseAt, resolvedAt, found, err := s.repo.GetTicketSLA(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}
	res := &SLAComplianceResult{
		TicketID: ticketID,
		TenantID: tenantID,
		Found:    found,
	}
	if !found {
		res.Message = "工单不存在或不属于该租户"
		return res, nil
	}
	res.CreatedAt = &createdAt
	if !firstResponseAt.IsZero() {
		t := firstResponseAt
		res.FirstResponseAt = &t
		res.ActualResponseMinutes = firstResponseAt.Sub(createdAt).Minutes()
	}
	if !resolvedAt.IsZero() {
		t := resolvedAt
		res.ResolvedAt = &t
		res.ActualResolutionMinutes = resolvedAt.Sub(createdAt).Minutes()
	}
	res.Compliant = !firstResponseAt.IsZero()
	if res.Compliant {
		res.Message = fmt.Sprintf("SLA计时正常，首次响应 %.1f 分钟", res.ActualResponseMinutes)
	} else {
		res.Message = "工单尚未首次响应"
	}
	return res, nil
}

// Violations

func (s *Service) GetSLAViolations(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*SLAViolation, int, error) {
	return s.repo.ListViolations(ctx, tenantID, page, size, filters)
}

func (s *Service) UpdateSLAViolationStatus(ctx context.Context, id int, isResolved bool, notes string, tenantID int) (*SLAViolation, error) {
	s.logger.Infow("Updating SLA Violation status", "id", id, "isResolved", isResolved)
	err := s.repo.UpdateViolationStatus(ctx, id, isResolved, notes, tenantID)
	if err != nil {
		return nil, err
	}
	// Return updated violation - fetch from repo
	violations, _, err := s.repo.ListViolations(ctx, tenantID, 1, 1, map[string]interface{}{"id": id})
	if err != nil || len(violations) == 0 {
		return nil, err
	}
	return violations[0], nil
}

// Metrics

func (s *Service) GetSLAMetrics(ctx context.Context, tenantID int, filters map[string]interface{}) ([]*SLAMetric, error) {
	return s.repo.GetMetrics(ctx, tenantID, filters)
}

func (s *Service) GetSLAMonitoring(ctx context.Context, tenantID int, startTime, endTime string) (map[string]interface{}, error) {
	return s.repo.GetSLAMonitoring(ctx, tenantID, startTime, endTime)
}

// Alert Rules

func (s *Service) CreateAlertRule(ctx context.Context, rule *SLAAlertRule) (*SLAAlertRule, error) {
	return s.repo.CreateAlertRule(ctx, rule)
}

func (s *Service) GetAlertRule(ctx context.Context, id int, tenantID int) (*SLAAlertRule, error) {
	return s.repo.GetAlertRule(ctx, id, tenantID)
}

func (s *Service) ListAlertRules(ctx context.Context, tenantID int, filters map[string]interface{}) ([]*SLAAlertRule, error) {
	return s.repo.ListAlertRules(ctx, tenantID, filters)
}

func (s *Service) UpdateAlertRule(ctx context.Context, rule *SLAAlertRule) (*SLAAlertRule, error) {
	return s.repo.UpdateAlertRule(ctx, rule)
}

func (s *Service) DeleteAlertRule(ctx context.Context, id int, tenantID int) error {
	return s.repo.DeleteAlertRule(ctx, id, tenantID)
}

// SLAAlertHistoryDefaultCooldownMinutes DDD 层默认告警抑制间隔（分钟）
//
// 与 service.SLAAlertService 的 defaultAlertCooldownMinutes 保持一致。
// 同一 (ticket_id, alert_rule_id) 在窗口内只保留一条未解决告警。
const SLAAlertHistoryDefaultCooldownMinutes = 15

func (s *Service) GetAlertHistory(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*SLAAlertHistory, int, error) {
	histories, total, err := s.repo.ListAlertHistory(ctx, tenantID, page, size, filters)
	if err != nil {
		return nil, 0, err
	}
	// 为每条 history 计算 cooldown 字段（未解决的告警仍处于冷却窗口时返回剩余秒数）
	now := time.Now()
	for _, h := range histories {
		h.CooldownMinutes = SLAAlertHistoryDefaultCooldownMinutes
		if h.ResolvedAt == nil {
			elapsed := now.Sub(h.CreatedAt)
			cooldownDur := time.Duration(SLAAlertHistoryDefaultCooldownMinutes) * time.Minute
			if elapsed < cooldownDur {
				remaining := cooldownDur - elapsed
				h.CooldownRemainingSeconds = int(remaining.Seconds())
				h.SuppressedByCooldown = true
			} else {
				h.CooldownRemainingSeconds = 0
				h.SuppressedByCooldown = false
			}
		} else {
			h.CooldownRemainingSeconds = 0
			h.SuppressedByCooldown = false
		}
	}
	return histories, total, nil
}

// GetSLAStats 获取SLA统计信息
func (s *Service) GetSLAStats(ctx context.Context, tenantID int) (map[string]interface{}, error) {
	// 获取总SLA定义数
	definitions, _, err := s.repo.ListDefinitions(ctx, tenantID, 1, 1000)
	if err != nil {
		return nil, err
	}

	// 获取活跃SLA定义数
	activeCount := 0
	for _, def := range definitions {
		if def.IsActive {
			activeCount++
		}
	}

	// 获取总违规数
	violations, _, err := s.repo.ListViolations(ctx, tenantID, 1, 1000, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	// 获取待处理违规数
	openViolations := 0
	for _, v := range violations {
		if !v.IsResolved {
			openViolations++
		}
	}

	// 计算合规率 - 基于总工单数而非违规数
	// 从 repository 获取总工单数和达标工单数来计算
	var complianceRate float64
	totalTickets, metSLA, err := s.repo.GetTicketStats(ctx, tenantID)
	if err != nil || totalTickets == 0 {
		// 如果无法获取工单数据，回退到基于违规的计算
		if len(violations) > 0 {
			complianceRate = float64(len(violations)-openViolations) / float64(len(violations)) * 100
		} else {
			complianceRate = 100.0
		}
	} else {
		complianceRate = float64(metSLA) / float64(totalTickets) * 100
	}

	return map[string]interface{}{
		"total_definitions":       len(definitions),
		"active_definitions":      activeCount,
		"total_violations":        len(violations),
		"open_violations":         openViolations,
		"overall_compliance_rate": complianceRate,
	}, nil
}

// GetComplianceReport generates SLA compliance report for a date range
func (s *Service) GetComplianceReport(ctx context.Context, tenantID int, startDate, endDate time.Time) (*dto.SLAComplianceReport, error) {
	s.logger.Infow("Generating SLA compliance report", "tenantID", tenantID, "startDate", startDate, "endDate", endDate)

	total, met, violated, avgResp, avgRes, err := s.repo.GetComplianceReportData(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	complianceRate := 0.0
	if total > 0 {
		complianceRate = float64(met) / float64(total) * 100
	}

	return &dto.SLAComplianceReport{
		TotalTickets:      total,
		MetSLA:            met,
		ViolatedSLA:       violated,
		ComplianceRate:    complianceRate,
		AvgResponseTime:   avgResp,
		AvgResolutionTime: avgRes,
		ReportPeriod: dto.SLAReportPeriod{
			StartDate: startDate.Format(time.RFC3339),
			EndDate:   endDate.Format(time.RFC3339),
		},
	}, nil
}

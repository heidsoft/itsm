package sla

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"

	"itsm-backend/dto"
)

// 监控/绩效查询的失败语义。handler 据此映射 HTTP status 与业务 code，
// 不得把参数错误、缺少租户和内部故障一律返回 500。
var (
	// ErrTenantRequired 缺少租户上下文时 fail closed，禁止回退到默认租户。
	ErrTenantRequired = errors.New("sla: tenant context is required")
	// ErrInvalidWindow 统计窗口非法（结束时间必须晚于开始时间）。
	ErrInvalidWindow = errors.New("sla: invalid monitoring window")
	// ErrInvalidDimension 绩效分组维度不受支持。
	ErrInvalidDimension = errors.New("sla: unsupported performance dimension")
)

// DefaultMonitoringWindow 未显式传入统计窗口时沿用历史默认值：最近 30 天。
const DefaultMonitoringWindow = 30 * 24 * time.Hour

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
// 阻断3 修复：合规判定必须基于 SLA 截止时间，而非"是否已首次响应"。
// 旧逻辑 `res.Compliant = !firstResponseAt.IsZero()` 把"已响应"等同于"合规"，
// 导致超时响应的工单仍被判为合规；同时未解决工单永远判为"不合规"，
// 与 SLA 定义中"未到截止时间即合规"的语义相悖。
func (s *Service) CheckSLACompliance(ctx context.Context, ticketID int, tenantID int) (*SLAComplianceResult, error) {
	s.logger.Infow("Checking SLA Compliance", "ticketID", ticketID)
	createdAt, firstResponseAt, resolvedAt, slaResponseDeadline, slaResolutionDeadline, found, err := s.repo.GetTicketSLA(ctx, ticketID, tenantID)
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

	// 合规判定：
	// 1) 响应合规：已首次响应 且 响应时间 <= slaResponseDeadline（deadline 未配置时视为合规）
	// 2) 解决合规：已解决 且 解决时间 <= slaResolutionDeadline（deadline 未配置时视为合规）
	// 3) 仍处于处理中（未解决）：若当前时间未超过 slaResolutionDeadline，则视为临时合规
	responseCompliant := true
	resolutionCompliant := true

	if !slaResponseDeadline.IsZero() {
		// 已配置响应截止时间
		if firstResponseAt.IsZero() {
			// 尚未响应：截止时间已过则违规
			responseCompliant = time.Now().Before(slaResponseDeadline)
		} else {
			// 已响应：比对响应时间与截止时间
			responseCompliant = !firstResponseAt.After(slaResponseDeadline)
		}
	}

	if !slaResolutionDeadline.IsZero() {
		// 已配置解决截止时间
		if resolvedAt.IsZero() {
			// 尚未解决：截止时间已过则违规
			resolutionCompliant = time.Now().Before(slaResolutionDeadline)
		} else {
			// 已解决：比对解决时间与截止时间
			resolutionCompliant = !resolvedAt.After(slaResolutionDeadline)
		}
	}

	res.Compliant = responseCompliant && resolutionCompliant
	switch {
	case !res.Compliant:
		res.Message = fmt.Sprintf("SLA 违规：响应合规=%v 解决合规=%v", responseCompliant, resolutionCompliant)
	case !firstResponseAt.IsZero() && !resolvedAt.IsZero():
		res.Message = fmt.Sprintf("SLA 计时正常，首次响应 %.1f 分钟，解决 %.1f 分钟",
			res.ActualResponseMinutes, res.ActualResolutionMinutes)
	case !firstResponseAt.IsZero():
		res.Message = fmt.Sprintf("SLA 计时正常，首次响应 %.1f 分钟，待解决",
			res.ActualResponseMinutes)
	default:
		res.Message = "SLA 计时正常，等待首次响应"
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

func (s *Service) GetSLAMonitoring(ctx context.Context, tenantID int, start, end time.Time) (*SLAMonitoringData, error) {
	if tenantID <= 0 {
		return nil, ErrTenantRequired
	}
	if end.IsZero() {
		end = time.Now().UTC()
	}
	if start.IsZero() {
		start = end.Add(-DefaultMonitoringWindow)
	}
	if !end.After(start) {
		return nil, ErrInvalidWindow
	}

	res, err := s.repo.GetSLAMonitoring(ctx, tenantID, start, end)
	if err != nil {
		// 原始错误只进结构化日志，响应体不得携带 SQL/驱动细节。
		s.logger.Errorw("failed to load SLA monitoring data",
			"tenantId", tenantID,
			"startTime", start,
			"endTime", end,
			"error", err,
		)
		return nil, err
	}
	return res, nil
}

// SLAPerformanceQuery 是按维度聚合绩效的查询条件。
type SLAPerformanceQuery struct {
	Dimension   string
	Start       time.Time
	End         time.Time
	ServiceType string
	Priority    string
	Page        int
	PageSize    int
}

// SLAPerformanceResult 是分页后的绩效结果。Total 是符合过滤条件的分组行数；
// Truncated 为 true 表示窗口内工单数命中扫描上限，分组结果不完整。
type SLAPerformanceResult struct {
	Items     []*SLAPerformanceRow
	Total     int
	Page      int
	PageSize  int
	Truncated bool
}

// ListSLAPerformance 校验维度与窗口，聚合后做服务端分页。
func (s *Service) ListSLAPerformance(ctx context.Context, tenantID int, q SLAPerformanceQuery) (*SLAPerformanceResult, error) {
	if tenantID <= 0 {
		return nil, ErrTenantRequired
	}
	if q.Dimension != SLADimensionServiceType && q.Dimension != SLADimensionPriority {
		return nil, ErrInvalidDimension
	}
	if q.End.IsZero() {
		q.End = time.Now().UTC()
	}
	if q.Start.IsZero() {
		q.Start = q.End.Add(-DefaultMonitoringWindow)
	}
	if !q.End.After(q.Start) {
		return nil, ErrInvalidWindow
	}

	page := q.Page
	if page <= 0 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	rows, truncated, err := s.repo.ListSLAPerformance(ctx, tenantID, q.Dimension, q.Start, q.End, q.ServiceType, q.Priority)
	if err != nil {
		s.logger.Errorw("failed to load SLA performance rows",
			"tenantId", tenantID,
			"dimension", q.Dimension,
			"error", err,
		)
		return nil, err
	}

	total := len(rows)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return &SLAPerformanceResult{
		Items:     rows[start:end],
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		Truncated: truncated,
	}, nil
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
		// 如果无法获取工单数据，回退到基于违规的计算；
		// 完全无样本时诚实返回 0（暂无数据），不伪装成 100% 合规。
		if len(violations) > 0 {
			complianceRate = float64(len(violations)-openViolations) / float64(len(violations)) * 100
		} else {
			complianceRate = 0
		}
	} else {
		complianceRate = float64(metSLA) / float64(totalTickets) * 100
	}
	// 保留一位小数，避免前端展示原始浮点噪声
	complianceRate = math.Round(complianceRate*10) / 10

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

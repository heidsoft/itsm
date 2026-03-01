package service

import (
	"context"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/ticket"

	"go.uber.org/zap"
)

// TicketSLAServiceInterface 工单SLA服务接口
type TicketSLAServiceInterface interface {
	// GetTicketSLAInfo 获取工单SLA信息
	GetTicketSLAInfo(ctx context.Context, ticketID int, tenantID int) (*TicketSLAInfoResult, error)
	// GetOverdueTickets 获取逾期工单
	GetOverdueTickets(ctx context.Context, tenantID int) ([]*ent.Ticket, error)
	// GetTicketStats 获取工单统计
	GetTicketStats(ctx context.Context, tenantID int) (*TicketStats, error)
	// CalculateSLADeadline 计算SLA截止时间
	CalculateSLADeadline(ctx context.Context, tenantID int, ticketType, priority string) (*SLADeadlineResult, error)
	// CalculateSLADeadlineFromRequest 根据请求参数计算SLA截止时间（包含SLADefinitionID）
	CalculateSLADeadlineFromRequest(ctx context.Context, tenantID int, ticketType, priority string) (*SLADeadlineResult, error)
	// AdjustToBusinessHours 调整时间到工作时间
	AdjustToBusinessHours(t time.Time) time.Time
}

// TicketSLAInfoResult 工单SLA信息（计算结果）
type TicketSLAInfoResult struct {
	TicketID           int        `json:"ticket_id"`
	TicketNumber       string     `json:"ticket_number"`
	Priority           string     `json:"priority"`
	TicketType         string     `json:"ticket_type"`
	ResponseDeadline   *time.Time `json:"response_deadline"`
	ResolutionDeadline *time.Time `json:"resolution_deadline"`
	ResponseTimeUsed   int        `json:"response_time_used"`   // 分钟
	ResolutionTimeUsed int        `json:"resolution_time_used"` // 分钟
	ResponseBreached   bool       `json:"response_breached"`
	ResolutionBreached bool       `json:"resolution_breached"`
	SLAStatus          string     `json:"sla_status"` // ok, warning, breached
}

// SLADeadlineResult SLA截止时间计算结果
type SLADeadlineResult struct {
	SLADefinitionID    int
	ResponseDeadline   *time.Time
	ResolutionDeadline *time.Time
	BusinessHoursOnly  bool
}

// TicketStats 工单统计
type TicketStats struct {
	TotalTickets      int `json:"total_tickets"`
	OpenTickets       int `json:"open_tickets"`
	InProgressTickets int `json:"in_progress_tickets"`
	ResolvedTickets   int `json:"resolved_tickets"`
	ClosedTickets     int `json:"closed_tickets"`
	OverdueTickets    int `json:"overdue_tickets"`
	BreachedTickets   int `json:"breached_tickets"`
}

// TicketSLAService 工单SLA服务
type TicketSLAService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewTicketSLAService 创建工单SLA服务
func NewTicketSLAService(client *ent.Client, logger *zap.SugaredLogger) *TicketSLAService {
	return &TicketSLAService{
		client: client,
		logger: logger,
	}
}

// GetTicketSLAInfo 获取工单SLA信息
func (s *TicketSLAService) GetTicketSLAInfo(ctx context.Context, ticketID int, tenantID int) (*TicketSLAInfoResult, error) {
	// 查询工单
	t, err := s.client.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to find ticket", "ticketID", ticketID, "error", err)
		return nil, err
	}

	// 计算已用时间
	responseTimeUsed := int(time.Since(t.CreatedAt).Minutes())
	resolutionTimeUsed := int(time.Since(t.CreatedAt).Minutes())

	// 如果已有首次响应时间或解决时间，使用实际时间
	if !t.FirstResponseAt.IsZero() {
		responseTimeUsed = int(t.FirstResponseAt.Sub(t.CreatedAt).Minutes())
	}
	if !t.ResolvedAt.IsZero() {
		resolutionTimeUsed = int(t.ResolvedAt.Sub(t.CreatedAt).Minutes())
	}

	// 获取SLA定义
	slaDef, err := s.getSLADefinition(ctx, tenantID, t.Type, t.Priority)
	if err != nil {
		s.logger.Warnw("Failed to get SLA definition", "error", err)
		// 返回没有SLA信息的结果
		return &TicketSLAInfoResult{
			TicketID:            t.ID,
			TicketNumber:        t.TicketNumber,
			Priority:            t.Priority,
			TicketType:          t.Type,
			ResponseTimeUsed:   responseTimeUsed,
			ResolutionTimeUsed: resolutionTimeUsed,
			SLAStatus:          "unknown",
		}, nil
	}

	// 计算截止时间（默认不使用工作时间）
	var responseDeadline, resolutionDeadline *time.Time
	if slaDef.ResponseTime > 0 {
		respDeadline := s.calculateDeadline(t.CreatedAt, slaDef.ResponseTime, false)
		responseDeadline = &respDeadline
	}
	if slaDef.ResolutionTime > 0 {
		resDeadline := s.calculateDeadline(t.CreatedAt, slaDef.ResolutionTime, false)
		resolutionDeadline = &resDeadline
	}

	// 判断是否违规
	responseBreached := false
	resolutionBreached := false
	slaStatus := "ok"

	if responseDeadline != nil && time.Now().After(*responseDeadline) {
		responseBreached = true
		slaStatus = "breached"
	}

	if resolutionDeadline != nil && time.Now().After(*resolutionDeadline) {
		resolutionBreached = true
		slaStatus = "breached"
	}

	// 检查警告状态（默认30分钟警告）
	if !responseBreached && !resolutionBreached && responseDeadline != nil {
		timeLeft := responseDeadline.Sub(time.Now())
		if timeLeft.Minutes() < 30 {
			slaStatus = "warning"
		}
	}

	return &TicketSLAInfoResult{
		TicketID:           t.ID,
		TicketNumber:       t.TicketNumber,
		Priority:           t.Priority,
		TicketType:         t.Type,
		ResponseDeadline:   responseDeadline,
		ResolutionDeadline: resolutionDeadline,
		ResponseTimeUsed:   responseTimeUsed,
		ResolutionTimeUsed: resolutionTimeUsed,
		ResponseBreached:   responseBreached,
		ResolutionBreached: resolutionBreached,
		SLAStatus:          slaStatus,
	}, nil
}

// GetOverdueTickets 获取逾期工单
func (s *TicketSLAService) GetOverdueTickets(ctx context.Context, tenantID int) ([]*ent.Ticket, error) {
	// 获取所有未关闭的工单
	tickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.StatusNEQ(common.TicketStatusClosed),
			ticket.StatusNEQ(common.TicketStatusResolved),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to query tickets", "error", err)
		return nil, err
	}

	// 筛选逾期工单
	var overdueTickets []*ent.Ticket
	now := time.Now()

	for _, t := range tickets {
		// 获取SLA定义
		slaDef, err := s.getSLADefinition(ctx, tenantID, t.Type, t.Priority)
		if err != nil {
			continue
		}

		if slaDef.ResolutionTime > 0 {
			deadline := s.calculateDeadline(t.CreatedAt, slaDef.ResolutionTime, false)
			if now.After(deadline) {
				overdueTickets = append(overdueTickets, t)
			}
		}
	}

	return overdueTickets, nil
}

// GetTicketStats 获取工单统计
func (s *TicketSLAService) GetTicketStats(ctx context.Context, tenantID int) (*TicketStats, error) {
	stats := &TicketStats{}

	// 统计总数
	total, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalTickets = total

	// 统计各状态数量
	openCount, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.Status(common.TicketStatusOpen)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.OpenTickets = openCount

	inProgressCount, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.Status(common.TicketStatusInProgress)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.InProgressTickets = inProgressCount

	resolvedCount, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.Status(common.TicketStatusResolved)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.ResolvedTickets = resolvedCount

	closedCount, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.Status(common.TicketStatusClosed)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.ClosedTickets = closedCount

	// 统计逾期工单
	overdueTickets, err := s.GetOverdueTickets(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	stats.OverdueTickets = len(overdueTickets)

	return stats, nil
}

// CalculateSLADeadline 计算SLA截止时间
func (s *TicketSLAService) CalculateSLADeadline(ctx context.Context, tenantID int, ticketType, priority string) (*SLADeadlineResult, error) {
	slaDef, err := s.getSLADefinition(ctx, tenantID, ticketType, priority)
	if err != nil {
		return nil, err
	}

	result := &SLADeadlineResult{}

	now := time.Now()

	if slaDef.ResponseTime > 0 {
		respDeadline := s.calculateDeadline(now, slaDef.ResponseTime, false)
		result.ResponseDeadline = &respDeadline
	}

	if slaDef.ResolutionTime > 0 {
		resDeadline := s.calculateDeadline(now, slaDef.ResolutionTime, false)
		result.ResolutionDeadline = &resDeadline
	}

	return result, nil
}

// getSLADefinition 获取SLA定义
func (s *TicketSLAService) getSLADefinition(ctx context.Context, tenantID int, ticketType, priority string) (*ent.SLADefinition, error) {
	// 尝试查找匹配的类型和优先级
	sla, err := s.client.SLADefinition.Query().
		Where(
			sladefinition.TenantID(tenantID),
			sladefinition.ServiceType(ticketType),
			sladefinition.Priority(priority),
			sladefinition.IsActive(true),
		).
		Only(ctx)
	if err == nil {
		return sla, nil
	}

	// 如果没有精确匹配，尝试只匹配类型
	sla, err = s.client.SLADefinition.Query().
		Where(
			sladefinition.TenantID(tenantID),
			sladefinition.ServiceType(ticketType),
			sladefinition.PriorityIsNil(),
			sladefinition.IsActive(true),
		).
		Only(ctx)
	if err == nil {
		return sla, nil
	}

	// 如果没有匹配的类型，返回默认SLA
	return &ent.SLADefinition{
		ResponseTime: 60,   // 默认1小时响应
		ResolutionTime: 480, // 默认8小时解决
	}, nil
}

// calculateDeadline 计算截止时间
func (s *TicketSLAService) calculateDeadline(startTime time.Time, durationMinutes int, businessHoursOnly bool) time.Time {
	if businessHoursOnly {
		return s.AdjustToBusinessHours(startTime.Add(time.Duration(durationMinutes) * time.Minute))
	}
	return startTime.Add(time.Duration(durationMinutes) * time.Minute)
}

// AdjustToBusinessHours 调整到工作时间（公开方法，供外部调用）
func (s *TicketSLAService) AdjustToBusinessHours(t time.Time) time.Time {
	// 简化的业务时间处理：假设工作时间为周一至周五 9:00-18:00
	year, month, day := t.Date()
	hour := t.Hour()

	// 如果是周末，调整到周一
	if t.Weekday() == time.Saturday {
		return time.Date(year, month, day+2, 9, 0, 0, 0, t.Location())
	}
	if t.Weekday() == time.Sunday {
		return time.Date(year, month, day+1, 9, 0, 0, 0, t.Location())
	}

	// 如果在工作时间之外
	if hour < 9 {
		return time.Date(year, month, day, 9, 0, 0, 0, t.Location())
	}
	if hour >= 18 {
		// 如果是周五18点后，调整到周一
		if t.Weekday() == time.Friday {
			return time.Date(year, month, day+3, 9, 0, 0, 0, t.Location())
		}
		return time.Date(year, month, day+1, 9, 0, 0, 0, t.Location())
	}

	return t
}

// CalculateSLADeadlineFromRequest 根据请求参数计算SLA截止时间（包含SLADefinitionID）
func (s *TicketSLAService) CalculateSLADeadlineFromRequest(ctx context.Context, tenantID int, ticketType, priority string) (*SLADeadlineResult, error) {
	now := time.Now()

	// 确定service_type
	var serviceType string
	switch ticketType {
	case "incident":
		serviceType = "incident"
	case "service_request":
		serviceType = "service_request"
	case "change":
		serviceType = "change"
	default:
		// 默认为incident类型
		serviceType = "incident"
	}

	// 标准化优先级
	normalizedPriority := strings.ToLower(priority)
	if normalizedPriority == "urgent" {
		normalizedPriority = "critical"
	}

	// 从数据库查找匹配的SLA定义
	sla, err := s.client.SLADefinition.Query().
		Where(
			sladefinition.TenantID(tenantID),
			sladefinition.ServiceType(serviceType),
			sladefinition.Priority(normalizedPriority),
			sladefinition.IsActive(true),
		).
		First(ctx)
	if err != nil {
		// 如果找不到精确匹配，尝试查找默认SLA（不带优先级）
		if ent.IsNotFound(err) {
			sla, err = s.client.SLADefinition.Query().
				Where(
					sladefinition.TenantID(tenantID),
					sladefinition.ServiceType(serviceType),
					sladefinition.IsActive(true),
				).
				First(ctx)
		}

		if err != nil || sla == nil {
			// 如果还是找不到，返回默认值
			s.logger.Warnw("No SLA definition found, using defaults", "service_type", serviceType, "priority", normalizedPriority)
			return &SLADeadlineResult{
				SLADefinitionID:    0,
				ResponseDeadline:   toPointer(now.Add(8 * time.Hour)),
				ResolutionDeadline: toPointer(now.Add(24 * time.Hour)),
			}, nil
		}
	}

	// 计算截止时间（单位是分钟）
	responseDeadline := now.Add(time.Duration(sla.ResponseTime) * time.Minute)
	resolutionDeadline := now.Add(time.Duration(sla.ResolutionTime) * time.Minute)

	// 考虑工作时间
	responseDeadline = s.AdjustToBusinessHours(responseDeadline)
	resolutionDeadline = s.AdjustToBusinessHours(resolutionDeadline)

	return &SLADeadlineResult{
		SLADefinitionID:    sla.ID,
		ResponseDeadline:   &responseDeadline,
		ResolutionDeadline: &resolutionDeadline,
	}, nil
}

// toPointer 返回指针（辅助函数）
func toPointer[T any](v T) *T {
	return &v
}

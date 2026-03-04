package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
)

// TicketSearchService 工单搜索服务
type TicketSearchService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// TicketSearchServiceInterface 工单搜索服务接口
type TicketSearchServiceInterface interface {
	SearchTickets(ctx context.Context, searchTerm string, tenantID int) ([]*ent.Ticket, error)
	GetOverdueTickets(ctx context.Context, tenantID int) ([]*ent.Ticket, error)
	GetTicketStats(ctx context.Context, tenantID int) (*dto.TicketStatsResponse, error)
	GetTicketAnalytics(ctx context.Context, tenantID int, dateFrom, dateTo time.Time) (*dto.TicketAnalyticsResponse, error)
}

// NewTicketSearchService 创建工单搜索服务
func NewTicketSearchService(client *ent.Client, logger *zap.SugaredLogger) *TicketSearchService {
	return &TicketSearchService{
		client: client,
		logger: logger,
	}
}

// SearchTickets 搜索工单（标题、描述、标签等）
func (s *TicketSearchService) SearchTickets(ctx context.Context, searchTerm string, tenantID int) ([]*ent.Ticket, error) {
	if strings.TrimSpace(searchTerm) == "" {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}

	// 使用Contains进行模糊搜索，先转小写实现不区分大小写
	searchLower := strings.ToLower(strings.TrimSpace(searchTerm))

	tickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.Or(
				ticket.TitleContainsFold(searchLower),
				ticket.DescriptionContainsFold(searchLower),
			),
		).
		Order(ent.Desc(ticket.FieldCreatedAt)).
		Limit(100). // 限制搜索结果
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to search tickets", "searchTerm", searchTerm, "error", err)
		return nil, fmt.Errorf("搜索工单失败: %w", err)
	}

	s.logger.Infow("Tickets searched", "tenant_id", tenantID, "term", searchTerm, "count", len(tickets))
	return tickets, nil
}

// GetOverdueTickets 获取已过期的工单
func (s *TicketSearchService) GetOverdueTickets(ctx context.Context, tenantID int) ([]*ent.Ticket, error) {
	now := time.Now()

	// 查找响应超时或解决超时的工单
	tickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.StatusIn("open", "in_progress", "pending"),
			ticket.Or(
				ticket.SLAResponseDeadlineLTE(now),
				ticket.SLAResolutionDeadlineLTE(now),
			),
		).
		Order(ent.Desc(ticket.FieldSLAResolutionDeadline)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get overdue tickets", "error", err)
		return nil, fmt.Errorf("获取过期工单失败: %w", err)
	}

	s.logger.Infow("Overdue tickets retrieved", "tenant_id", tenantID, "count", len(tickets))
	return tickets, nil
}

// GetTicketStats 获取工单统计信息
func (s *TicketSearchService) GetTicketStats(ctx context.Context, tenantID int) (*dto.TicketStatsResponse, error) {
	response := &dto.TicketStatsResponse{}

	// 统计总工单数
	total, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID)).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("统计总工单数失败: %w", err)
	}
	response.Total = total

	// 按状态统计
	openCount, _ := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.StatusEQ("open")).
		Count(ctx)
	response.Open = openCount

	inProgressCount, _ := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.StatusEQ("in_progress")).
		Count(ctx)
	response.InProgress = inProgressCount

	resolvedCount, _ := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.StatusEQ("resolved")).
		Count(ctx)
	response.Resolved = resolvedCount

	// 高优先级工单数 (high + critical)
	highPriorityCount, _ := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.PriorityIn("high", "critical"),
		).
		Count(ctx)
	response.HighPriority = highPriorityCount

	// 逾期工单数（通过GetOverdueTickets获取）
	overdue, _ := s.GetOverdueTickets(ctx, tenantID)
	response.Overdue = len(overdue)

	s.logger.Infow("Ticket stats calculated", "tenant_id", tenantID, "total", total)
	return response, nil
}

// GetTicketAnalytics 获取工单分析数据
func (s *TicketSearchService) GetTicketAnalytics(ctx context.Context, tenantID int, dateFrom, dateTo time.Time) (*dto.TicketAnalyticsResponse, error) {
	response := &dto.TicketAnalyticsResponse{
		Data:        make([]map[string]interface{}, 0),
		Summary:     make(map[string]interface{}),
		Trends:      make([]map[string]interface{}, 0),
		GeneratedAt: time.Now(),
	}

	// 统计时间段内的创建数
	createdCount, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.CreatedAtGTE(dateFrom),
			ticket.CreatedAtLTE(dateTo),
		).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("统计创建数失败: %w", err)
	}

	// 统计时间段内的解决数
	resolvedCount, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.StatusEQ("resolved"),
			ticket.UpdatedAtGTE(dateFrom),
			ticket.UpdatedAtLTE(dateTo),
		).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("统计解决数失败: %w", err)
	}

	// 设置摘要
	response.Summary["tickets_created"] = createdCount
	response.Summary["tickets_resolved"] = resolvedCount
	response.Summary["net_change"] = createdCount - resolvedCount

	// 按日统计趋势（简化）
	days := int(dateTo.Sub(dateFrom).Hours()/24) + 1
	if days < 1 {
		days = 1
	}

	dailyTrends := make([]map[string]interface{}, days)
	dayDuration := 24 * time.Hour

	for i := 0; i < days; i++ {
		dayStart := dateFrom.Add(time.Duration(i) * dayDuration)
		dayEnd := dayStart.Add(dayDuration)

		dayCreated, _ := s.client.Ticket.Query().
			Where(
				ticket.TenantID(tenantID),
				ticket.CreatedAtGTE(dayStart),
				ticket.CreatedAtLTE(dayEnd),
			).
			Count(ctx)

		dayResolved, _ := s.client.Ticket.Query().
			Where(
				ticket.TenantID(tenantID),
				ticket.StatusEQ("resolved"),
				ticket.UpdatedAtGTE(dayStart),
				ticket.UpdatedAtLTE(dayEnd),
			).
			Count(ctx)

		dailyTrends[i] = map[string]interface{}{
			"date":       dayStart.Format("2006-01-02"),
			"created":    dayCreated,
			"resolved":   dayResolved,
			"net_change": dayCreated - dayResolved,
		}
	}

	response.Trends = dailyTrends

	s.logger.Infow("Ticket analytics calculated", "tenant_id", tenantID, "from", dateFrom, "to", dateTo)
	return response, nil
}

package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"time"

	"go.uber.org/zap"
)

type TicketStatsService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewTicketStatsService(client *ent.Client, logger *zap.SugaredLogger) *TicketStatsService {
	return &TicketStatsService{
		client: client,
		logger: logger,
	}
}

// GetTicketStats 获取工单统计信息
func (s *TicketStatsService) GetTicketStats(ctx context.Context, tenantID int) (*dto.TicketStatsResponse, error) {
	s.logger.Infow("Getting ticket stats", "tenant_id", tenantID)

	// 获取总数
	total, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID)).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count total tickets", "error", err)
		return nil, fmt.Errorf("failed to count total tickets: %w", err)
	}

	// 获取各状态工单数量
	open, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.StatusEQ("open")).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count open tickets", "error", err)
		return nil, fmt.Errorf("failed to count open tickets: %w", err)
	}

	inProgress, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.StatusEQ("in_progress")).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count in_progress tickets", "error", err)
		return nil, fmt.Errorf("failed to count in_progress tickets: %w", err)
	}

	resolved, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.StatusEQ("resolved")).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count resolved tickets", "error", err)
		return nil, fmt.Errorf("failed to count resolved tickets: %w", err)
	}

	// 获取高优先级工单数量
	highPriority, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.PriorityIn("high", "critical")).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count high priority tickets", "error", err)
		return nil, fmt.Errorf("failed to count high priority tickets: %w", err)
	}

	// 获取逾期工单数量（这里简化处理，实际应该根据SLA计算）
	overdue, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.StatusIn("open", "in_progress"),
			ticket.CreatedAtLT(time.Now().Add(-24*time.Hour)), // 简化：超过24小时未解决
		).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count overdue tickets", "error", err)
		return nil, fmt.Errorf("failed to count overdue tickets: %w", err)
	}

	return &dto.TicketStatsResponse{
		Total:        total,
		Open:         open,
		InProgress:   inProgress,
		Resolved:     resolved,
		HighPriority: highPriority,
		Overdue:      overdue,
	}, nil
}

// GetTicketAnalytics 获取工单分析数据
func (s *TicketStatsService) GetTicketAnalytics(ctx context.Context, req *dto.TicketAnalyticsRequest, tenantID int) (*dto.TicketAnalyticsResponse, error) {
	s.logger.Infow("Getting ticket analytics", "tenant_id", tenantID, "request", req)

	// 基础查询
	query := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.CreatedAtGTE(req.DateFrom),
			ticket.CreatedAtLTE(req.DateTo),
		)

	// 应用筛选条件
	if filters, ok := req.Filters["status"].(string); ok && filters != "" {
		query.Where(ticket.StatusEQ(filters))
	}
	if filters, ok := req.Filters["priority"].(string); ok && filters != "" {
		query.Where(ticket.PriorityEQ(filters))
	}
	if filters, ok := req.Filters["assignee_id"].(int); ok && filters != 0 {
		query.Where(ticket.AssigneeID(filters))
	}

	// 获取工单列表
	tickets, err := query.All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get tickets for analytics", "error", err)
		return nil, fmt.Errorf("failed to get tickets for analytics: %w", err)
	}

	// 根据GroupBy分组数据
	data := s.groupTicketsByPeriod(tickets, req.GroupBy)

	// 计算汇总信息
	summary := s.calculateSummary(tickets, req.Metrics)

	// 计算趋势数据
	trends := s.calculateTrends(tickets, req.GroupBy)

	return &dto.TicketAnalyticsResponse{
		Data:        data,
		Summary:     summary,
		Trends:      trends,
		GeneratedAt: time.Now(),
	}, nil
}

// groupTicketsByPeriod 按时间周期分组工单
func (s *TicketStatsService) groupTicketsByPeriod(tickets []*ent.Ticket, groupBy string) []map[string]interface{} {
	groups := make(map[string]map[string]int)

	for _, ticket := range tickets {
		var key string
		switch groupBy {
		case "day":
			key = ticket.CreatedAt.Format("2006-01-02")
		case "week":
			year, week := ticket.CreatedAt.ISOWeek()
			key = fmt.Sprintf("%d-W%02d", year, week)
		case "month":
			key = ticket.CreatedAt.Format("2006-01")
		case "category":
			// 这里需要根据实际的分类字段调整
			key = "未分类" // 简化处理
		case "priority":
			key = ticket.Priority
		case "assignee":
			key = fmt.Sprintf("assignee_%d", ticket.AssigneeID)
		default:
			key = ticket.CreatedAt.Format("2006-01-02")
		}

		if groups[key] == nil {
			groups[key] = make(map[string]int)
		}
		groups[key]["total"]++
		groups[key][ticket.Status]++
		groups[key][ticket.Priority]++
	}

	// 转换为切片格式
	var result []map[string]interface{}
	for key, values := range groups {
		item := map[string]interface{}{
			"period": key,
		}
		for k, v := range values {
			item[k] = v
		}
		result = append(result, item)
	}

	return result
}

// calculateSummary 计算汇总信息
func (s *TicketStatsService) calculateSummary(tickets []*ent.Ticket, metrics []string) map[string]interface{} {
	summary := make(map[string]interface{})

	summary["total_tickets"] = len(tickets)

	// 状态分布
	statusCount := make(map[string]int)
	priorityCount := make(map[string]int)
	
	for _, ticket := range tickets {
		statusCount[ticket.Status]++
		priorityCount[ticket.Priority]++
	}

	summary["status_distribution"] = statusCount
	summary["priority_distribution"] = priorityCount

	// 平均处理时间（简化计算）
	var totalProcessingTime time.Duration
	processedCount := 0
	
	for _, ticket := range tickets {
		if ticket.Status == "resolved" || ticket.Status == "closed" {
			processingTime := ticket.UpdatedAt.Sub(ticket.CreatedAt)
			totalProcessingTime += processingTime
			processedCount++
		}
	}

	if processedCount > 0 {
		avgProcessingTime := totalProcessingTime / time.Duration(processedCount)
		summary["avg_processing_time_hours"] = avgProcessingTime.Hours()
	}

	return summary
}

// calculateTrends 计算趋势数据
func (s *TicketStatsService) calculateTrends(tickets []*ent.Ticket, groupBy string) []map[string]interface{} {
	// 简化的趋势计算
	trends := []map[string]interface{}{
		{
			"metric": "creation_trend",
			"value":  len(tickets),
			"change": 0, // 这里应该与上一周期比较
		},
	}

	return trends
}
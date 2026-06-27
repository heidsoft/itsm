package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
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
	// 使用并发查询优化性能（避免串行的 N+1 查询）
	errCh := make(chan error, 5)
	resultCh := make(chan *dto.TicketStatsResponse, 1)
	overdueCh := make(chan []*ent.Ticket, 1)

	go func() {
		// 并发查询逾期工单（单独因为需要复杂条件）
		defer close(overdueCh)
		overdue, err := s.GetOverdueTickets(ctx, tenantID)
		if err != nil {
			s.logger.Warnw("Failed to get overdue tickets", "error", err)
		}
		overdueCh <- overdue
	}()

	go func() {
		// 并发执行多个统计查询
		response := &dto.TicketStatsResponse{}
		var wg sync.WaitGroup
		var mu sync.Mutex

		// 总数
		wg.Add(1)
		go func() {
			defer wg.Done()
			total, err := s.client.Ticket.Query().
				Where(ticket.TenantID(tenantID)).
				Count(ctx)
			if err != nil {
				errCh <- fmt.Errorf("统计总工单数失败: %w", err)
				return
			}
			mu.Lock()
			response.Total = total
			mu.Unlock()
		}()

		// open 数量
		wg.Add(1)
		go func() {
			defer wg.Done()
			count, err := s.client.Ticket.Query().
				Where(ticket.TenantID(tenantID), ticket.StatusEQ("open")).
				Count(ctx)
			if err != nil {
				errCh <- fmt.Errorf("统计open工单失败: %w", err)
				return
			}
			mu.Lock()
			response.Open = count
			mu.Unlock()
		}()

		// in_progress 数量
		wg.Add(1)
		go func() {
			defer wg.Done()
			count, err := s.client.Ticket.Query().
				Where(ticket.TenantID(tenantID), ticket.StatusEQ("in_progress")).
				Count(ctx)
			if err != nil {
				errCh <- fmt.Errorf("统计in_progress工单失败: %w", err)
				return
			}
			mu.Lock()
			response.InProgress = count
			mu.Unlock()
		}()

		// resolved 数量
		wg.Add(1)
		go func() {
			defer wg.Done()
			count, err := s.client.Ticket.Query().
				Where(ticket.TenantID(tenantID), ticket.StatusEQ("resolved")).
				Count(ctx)
			if err != nil {
				errCh <- fmt.Errorf("统计resolved工单失败: %w", err)
				return
			}
			mu.Lock()
			response.Resolved = count
			mu.Unlock()
		}()

		// high/critical 优先级数量
		wg.Add(1)
		go func() {
			defer wg.Done()
			count, err := s.client.Ticket.Query().
				Where(
					ticket.TenantID(tenantID),
					ticket.PriorityIn("high", "critical"),
				).
				Count(ctx)
			if err != nil {
				errCh <- fmt.Errorf("统计高优先级工单失败: %w", err)
				return
			}
			mu.Lock()
			response.HighPriority = count
			mu.Unlock()
		}()

		wg.Wait()
		close(errCh)
		resultCh <- response
	}()

	response := <-resultCh
	overdue := <-overdueCh
	response.Overdue = len(overdue)

	// 检查错误
	for err := range errCh {
		if err != nil {
			return nil, err
		}
	}

	s.logger.Infow("Ticket stats calculated", "tenant_id", tenantID, "total", response.Total)
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

	// 并发查询统计数据
	type summaryResult struct {
		created  int
		resolved int
	}

	summaryCh := make(chan summaryResult, 1)
	errCh := make(chan error, 1)

	go func() {
		createdCount, err := s.client.Ticket.Query().
			Where(
				ticket.TenantID(tenantID),
				ticket.CreatedAtGTE(dateFrom),
				ticket.CreatedAtLTE(dateTo),
			).
			Count(ctx)
		if err != nil {
			errCh <- fmt.Errorf("统计创建数失败: %w", err)
			return
		}

		resolvedCount, err := s.client.Ticket.Query().
			Where(
				ticket.TenantID(tenantID),
				ticket.StatusEQ("resolved"),
				ticket.UpdatedAtGTE(dateFrom),
				ticket.UpdatedAtLTE(dateTo),
			).
			Count(ctx)
		if err != nil {
			errCh <- fmt.Errorf("统计解决数失败: %w", err)
			return
		}

		summaryCh <- summaryResult{created: createdCount, resolved: resolvedCount}
	}()

	// 并发获取每日趋势
	trendsCh := make(chan []map[string]interface{}, 1)

	go func() {
		// 获取所有工单并按天分组（在内存中计算，避免 N+1）
		tickets, err := s.client.Ticket.Query().
			Where(
				ticket.TenantID(tenantID),
				ticket.CreatedAtGTE(dateFrom),
				ticket.CreatedAtLTE(dateTo),
			).
			All(ctx)
		if err != nil {
			s.logger.Warnw("Failed to get tickets for daily trends", "error", err)
			trendsCh <- []map[string]interface{}{}
			return
		}

		// 按天分组计算
		dailyMap := make(map[string]map[string]int)
		for _, t := range tickets {
			dateKey := t.CreatedAt.Format("2006-01-02")
			if dailyMap[dateKey] == nil {
				dailyMap[dateKey] = map[string]int{"created": 0, "resolved": 0}
			}
			dailyMap[dateKey]["created"]++
			if t.Status == "resolved" || t.Status == "closed" {
				dailyMap[dateKey]["resolved"]++
			}
		}

		// 转换为趋势数组
		trends := make([]map[string]interface{}, 0, len(dailyMap))
		for date, stats := range dailyMap {
			trends = append(trends, map[string]interface{}{
				"date":       date,
				"created":    stats["created"],
				"resolved":   stats["resolved"],
				"net_change": stats["created"] - stats["resolved"],
			})
		}
		trendsCh <- trends
	}()

	// 等待结果
	select {
	case summary := <-summaryCh:
		response.Summary["tickets_created"] = summary.created
		response.Summary["tickets_resolved"] = summary.resolved
		response.Summary["net_change"] = summary.created - summary.resolved
	case err := <-errCh:
		return nil, err
	}

	response.Trends = <-trendsCh

	s.logger.Infow("Ticket analytics calculated", "tenant_id", tenantID, "from", dateFrom, "to", dateTo)
	return response, nil
}

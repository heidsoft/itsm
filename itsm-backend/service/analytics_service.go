package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"

	"go.uber.org/zap"
)

type AnalyticsService struct {
	client           *ent.Client
	logger           *zap.SugaredLogger
	reportExport     *ReportExportService
}

func NewAnalyticsService(client *ent.Client, logger *zap.SugaredLogger) *AnalyticsService {
	return &AnalyticsService{
		client:           client,
		logger:           logger,
		reportExport:     NewReportExportService(),
	}
}

// GetDeepAnalytics 获取深度分析数据
func (s *AnalyticsService) GetDeepAnalytics(ctx context.Context, req *dto.DeepAnalyticsRequest, tenantID int) (*dto.DeepAnalyticsResponse, error) {
	s.logger.Infow("Getting deep analytics", "tenant_id", tenantID, "dimensions", req.Dimensions)

	// 解析时间范围
	startTime, err := time.Parse("2006-01-02", req.TimeRange[0])
	if err != nil {
		return nil, fmt.Errorf("invalid start time format: %w", err)
	}
	endTime, err := time.Parse("2006-01-02", req.TimeRange[1])
	if err != nil {
		return nil, fmt.Errorf("invalid end time format: %w", err)
	}
	endTime = endTime.Add(24 * time.Hour) // 包含结束日期

	// 构建查询
	query := s.client.Ticket.Query().
		Where(
			ticket.TenantIDEQ(tenantID),
			ticket.CreatedAtGTE(startTime),
			ticket.CreatedAtLT(endTime),
		)

	// 应用过滤器
	if req.Filters != nil {
		if status, ok := req.Filters["status"].(string); ok && status != "" {
			query = query.Where(ticket.StatusEQ(status))
		}
		if priority, ok := req.Filters["priority"].(string); ok && priority != "" {
			query = query.Where(ticket.PriorityEQ(priority))
		}
		if assigneeID, ok := req.Filters["assignee_id"].(float64); ok {
			query = query.Where(ticket.AssigneeIDEQ(int(assigneeID)))
		}
	}

	// 获取所有工单
	allTickets, err := query.All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get tickets", "error", err)
		return nil, fmt.Errorf("failed to get tickets: %w", err)
	}

	// 按维度分组统计
	dataPoints := s.analyzeByDimensions(allTickets, req.Dimensions, req.Metrics, req.GroupBy)

	// 计算汇总数据
	summary := s.calculateSummary(allTickets)

	response := &dto.DeepAnalyticsResponse{
		Data:        dataPoints,
		Summary:     summary,
		GeneratedAt: time.Now(),
	}

	return response, nil
}

// analyzeByDimensions 按维度分析数据
func (s *AnalyticsService) analyzeByDimensions(tickets []*ent.Ticket, dimensions []string, metrics []string, groupBy *string) []dto.AnalyticsDataPoint {
	dataPoints := make([]dto.AnalyticsDataPoint, 0)
	
	// 如果指定了groupBy，按该维度分组
	if groupBy != nil && *groupBy != "" {
		groups := make(map[string][]*ent.Ticket)
		for _, t := range tickets {
			key := s.getDimensionValue(t, *groupBy)
			groups[key] = append(groups[key], t)
		}

		for key, groupTickets := range groups {
			point := dto.AnalyticsDataPoint{
				Name: key,
			}
			point = s.calculateMetrics(point, groupTickets, metrics)
			dataPoints = append(dataPoints, point)
		}
	} else {
		// 按第一个维度分组
		if len(dimensions) > 0 {
			groups := make(map[string][]*ent.Ticket)
			for _, t := range tickets {
				key := s.getDimensionValue(t, dimensions[0])
				groups[key] = append(groups[key], t)
			}

			for key, groupTickets := range groups {
				point := dto.AnalyticsDataPoint{
					Name: key,
				}
				point = s.calculateMetrics(point, groupTickets, metrics)
				dataPoints = append(dataPoints, point)
			}
		}
	}

	return dataPoints
}

// getDimensionValue 获取维度值
func (s *AnalyticsService) getDimensionValue(t *ent.Ticket, dimension string) string {
	switch dimension {
	case "status":
		return t.Status
	case "priority":
		return t.Priority
	case "assignee":
		if t.AssigneeID > 0 {
			return fmt.Sprintf("用户%d", t.AssigneeID)
		}
		return "未分配"
	case "category":
		if t.CategoryID > 0 {
			return fmt.Sprintf("分类%d", t.CategoryID)
		}
		return "未分类"
	default:
		return "未知"
	}
}

// calculateMetrics 计算指标
func (s *AnalyticsService) calculateMetrics(point dto.AnalyticsDataPoint, tickets []*ent.Ticket, metrics []string) dto.AnalyticsDataPoint {
	point.Count = len(tickets)

	for _, metric := range metrics {
		switch metric {
		case "count":
			point.Value = float64(len(tickets))
		case "response_time":
			// 计算平均响应时间（如果有首次响应时间）
			var totalMinutes float64
			var count int
			for _, t := range tickets {
				if !t.FirstResponseAt.IsZero() && !t.CreatedAt.IsZero() {
					minutes := t.FirstResponseAt.Sub(t.CreatedAt).Minutes()
					totalMinutes += minutes
					count++
				}
			}
			if count > 0 {
				avg := totalMinutes / float64(count)
				point.AvgTime = &avg
			}
		case "resolution_time":
			// 计算平均解决时间
			var totalMinutes float64
			var count int
			for _, t := range tickets {
				if !t.ResolvedAt.IsZero() && !t.CreatedAt.IsZero() {
					minutes := t.ResolvedAt.Sub(t.CreatedAt).Minutes()
					totalMinutes += minutes
					count++
				}
			}
			if count > 0 {
				avg := totalMinutes / float64(count)
				point.AvgTime = &avg
			}
		}
	}

	return point
}

// calculateSummary 计算汇总数据
func (s *AnalyticsService) calculateSummary(tickets []*ent.Ticket) dto.AnalyticsSummary {
	summary := dto.AnalyticsSummary{
		Total: len(tickets),
	}

	var resolvedCount int
	var totalResponseTime, totalResolutionTime float64
	var responseCount, resolutionCount int

	for _, t := range tickets {
		if t.Status == "resolved" || t.Status == "closed" {
			resolvedCount++
		}

		if !t.FirstResponseAt.IsZero() && !t.CreatedAt.IsZero() {
			totalResponseTime += t.FirstResponseAt.Sub(t.CreatedAt).Minutes()
			responseCount++
		}

		if !t.ResolvedAt.IsZero() && !t.CreatedAt.IsZero() {
			totalResolutionTime += t.ResolvedAt.Sub(t.CreatedAt).Minutes()
			resolutionCount++
		}
	}

	summary.Resolved = resolvedCount

	if responseCount > 0 {
		summary.AvgResponseTime = totalResponseTime / float64(responseCount)
	}
	if resolutionCount > 0 {
		summary.AvgResolutionTime = totalResolutionTime / float64(resolutionCount)
	}

	// 计算SLA合规率（简化版，实际需要查询SLA定义）
	if len(tickets) > 0 {
		summary.SLACompliance = float64(resolvedCount) / float64(len(tickets)) * 100
	}

	// 计算客户满意度（基于评分）
	var totalRating float64
	var ratingCount int
	for _, t := range tickets {
		if t.Rating > 0 {
			totalRating += float64(t.Rating)
			ratingCount++
		}
	}
	if ratingCount > 0 {
		summary.CustomerSatisfaction = totalRating / float64(ratingCount)
	}

	return summary
}

// ExportAnalytics 导出分析数据
func (s *AnalyticsService) ExportAnalytics(ctx context.Context, req *dto.DeepAnalyticsRequest, format string, tenantID int) ([]byte, string, error) {
	s.logger.Infow("Exporting analytics", "format", format, "tenant_id", tenantID)

	// 获取分析数据
	response, err := s.GetDeepAnalytics(ctx, req, tenantID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get analytics data: %w", err)
	}

	// 根据格式生成文件
	switch format {
	case "csv":
		return s.exportToCSV(response)
	case "excel":
		return s.exportToExcel(response)
	case "pdf":
		return s.exportToPDF(response)
	default:
		return nil, "", fmt.Errorf("unsupported format: %s", format)
	}
}

// exportToCSV 导出为CSV格式
func (s *AnalyticsService) exportToCSV(response *dto.DeepAnalyticsResponse) ([]byte, string, error) {
	var csvData string
	
	// CSV头部
	csvData = "名称,数值,数量,平均时间\n"
	
	// 数据行
	for _, point := range response.Data {
		avgTime := ""
		if point.AvgTime != nil {
			avgTime = fmt.Sprintf("%.2f", *point.AvgTime)
		}
		csvData += fmt.Sprintf("%s,%.2f,%d,%s\n", point.Name, point.Value, point.Count, avgTime)
	}
	
	// 添加汇总信息
	csvData += "\n汇总信息\n"
	csvData += fmt.Sprintf("总计,%d\n", response.Summary.Total)
	csvData += fmt.Sprintf("已解决,%d\n", response.Summary.Resolved)
	csvData += fmt.Sprintf("平均响应时间,%.2f\n", response.Summary.AvgResponseTime)
	csvData += fmt.Sprintf("平均解决时间,%.2f\n", response.Summary.AvgResolutionTime)
	csvData += fmt.Sprintf("SLA合规率,%.2f%%\n", response.Summary.SLACompliance)
	csvData += fmt.Sprintf("客户满意度,%.2f\n", response.Summary.CustomerSatisfaction)
	
	filename := fmt.Sprintf("analytics_%s.csv", time.Now().Format("20060102_150405"))
	return []byte(csvData), filename, nil
}

// exportToExcel 导出为Excel格式
func (s *AnalyticsService) exportToExcel(response *dto.DeepAnalyticsResponse) ([]byte, string, error) {
	return s.reportExport.ExportToExcel(context.Background(), response)
}

// exportToPDF 导出为PDF格式
func (s *AnalyticsService) exportToPDF(response *dto.DeepAnalyticsResponse) ([]byte, string, error) {
	return s.reportExport.ExportToPDF(context.Background(), response)
}


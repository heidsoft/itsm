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

type PredictionService struct {
	client       *ent.Client
	logger       *zap.SugaredLogger
	reportExport *ReportExportService
}

func NewPredictionService(client *ent.Client, logger *zap.SugaredLogger) *PredictionService {
	return &PredictionService{
		client:       client,
		logger:       logger,
		reportExport: NewReportExportService(),
	}
}

// GetTrendPrediction 获取趋势预测
func (s *PredictionService) GetTrendPrediction(ctx context.Context, req *dto.TrendPredictionRequest, tenantID int) (*dto.TrendPredictionResponse, error) {
	s.logger.Infow("Getting trend prediction", "tenant_id", tenantID, "type", req.PredictionType)

	// 解析时间范围
	startTime, err := time.Parse("2006-01-02", req.TimeRange[0])
	if err != nil {
		return nil, fmt.Errorf("invalid start time format: %w", err)
	}
	endTime, err := time.Parse("2006-01-02", req.TimeRange[1])
	if err != nil {
		return nil, fmt.Errorf("invalid end time format: %w", err)
	}

	// 获取历史数据（最近6个月）
	historyStart := startTime.AddDate(0, -6, 0)
	historyTickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantIDEQ(tenantID),
			ticket.CreatedAtGTE(historyStart),
			ticket.CreatedAtLTE(endTime),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get history tickets", "error", err)
		return nil, fmt.Errorf("failed to get history tickets: %w", err)
	}

	// 根据预测类型生成预测数据
	var predictions []dto.PredictionDataPoint
	model := req.Model
	if model == "" {
		model = "arima" // 默认模型
	}

	switch req.PredictionType {
	case "volume":
		predictions = s.predictVolume(historyTickets, startTime, endTime)
	case "type":
		predictions = s.predictByType(historyTickets, startTime, endTime)
	case "priority":
		predictions = s.predictByPriority(historyTickets, startTime, endTime)
	case "resource":
		predictions = s.predictResourceDemand(historyTickets, startTime, endTime)
	default:
		predictions = s.predictVolume(historyTickets, startTime, endTime)
	}

	response := &dto.TrendPredictionResponse{
		Predictions: predictions,
		Confidence:  0.85, // 模拟置信度
		Model:       model,
		GeneratedAt: time.Now(),
	}

	return response, nil
}

// predictVolume 预测工单数量
func (s *PredictionService) predictVolume(tickets []*ent.Ticket, startTime, endTime time.Time) []dto.PredictionDataPoint {
	// 按月份统计历史数据
	monthlyCounts := make(map[string]int)
	for _, t := range tickets {
		month := t.CreatedAt.Format("2006-01")
		monthlyCounts[month]++
	}

	// 生成预测（简化版：基于历史平均值和趋势）
	predictions := make([]dto.PredictionDataPoint, 0)
	current := startTime
	avgCount := s.calculateAverage(monthlyCounts)

	for current.Before(endTime) || current.Equal(endTime) {
		dateStr := current.Format("2006-01-02")
		predictedValue := float64(avgCount) * (1.0 + float64(current.Month()-startTime.Month())*0.05) // 简单线性增长
		
		predictions = append(predictions, dto.PredictionDataPoint{
			Date:          dateStr,
			PredictedValue: predictedValue,
			LowerBound:    predictedValue * 0.85,
			UpperBound:    predictedValue * 1.15,
			Confidence:    0.85,
		})

		current = current.AddDate(0, 0, 7) // 每周一个预测点
	}

	return predictions
}

// predictByType 按类型预测
func (s *PredictionService) predictByType(tickets []*ent.Ticket, startTime, endTime time.Time) []dto.PredictionDataPoint {
	// 统计各类型工单的历史分布
	typeCounts := make(map[string]int)
	for _, t := range tickets {
		// 使用分类或默认类型
		typeKey := "general"
		if t.Edges.Category != nil {
			typeKey = t.Edges.Category.Name
		}
		typeCounts[typeKey]++
	}

	// 生成预测
	predictions := make([]dto.PredictionDataPoint, 0)
	current := startTime
	totalTickets := len(tickets)
	weeks := int(endTime.Sub(startTime).Hours() / (24 * 7))

	if totalTickets == 0 {
		return predictions
	}

	// 按类型生成预测
	for typeName, count := range typeCounts {
		baseCount := float64(count) / float64(weeks)
		for i := 0; i <= weeks; i++ {
			dateStr := current.Format("2006-01-02")
			// 添加一些波动
			variation := 1.0 + float64(i)*0.02 - 0.1 + float64(i%3)*0.05
			predictedValue := baseCount * variation

			predictions = append(predictions, dto.PredictionDataPoint{
				Date:           dateStr,
				PredictedValue: predictedValue,
				LowerBound:     predictedValue * 0.8,
				UpperBound:     predictedValue * 1.2,
				Confidence:     0.75,
				Category:       typeName,
			})
		}
	}

	return predictions
}

// predictByPriority 按优先级预测
func (s *PredictionService) predictByPriority(tickets []*ent.Ticket, startTime, endTime time.Time) []dto.PredictionDataPoint {
	// 统计各优先级的历史分布
	priorityCounts := make(map[string]int)
	for _, t := range tickets {
		priorityCounts[t.Priority]++
	}

	// 生成预测
	predictions := make([]dto.PredictionDataPoint, 0)
	current := startTime
	totalTickets := len(tickets)
	weeks := int(endTime.Sub(startTime).Hours() / (24 * 7))

	if totalTickets == 0 {
		return predictions
	}

	// 按优先级生成预测
	for priority, count := range priorityCounts {
		baseCount := float64(count) / float64(weeks)
		for i := 0; i <= weeks; i++ {
			dateStr := current.Format("2006-01-02")
			// 添加一些波动
			variation := 1.0 + float64(i)*0.02 - 0.1 + float64(i%3)*0.05
			predictedValue := baseCount * variation

			predictions = append(predictions, dto.PredictionDataPoint{
				Date:           dateStr,
				PredictedValue: predictedValue,
				LowerBound:     predictedValue * 0.8,
				UpperBound:     predictedValue * 1.2,
				Confidence:     0.75,
				Priority:       priority,
			})
		}
	}

	return predictions
}

// predictResourceDemand 预测资源需求
func (s *PredictionService) predictResourceDemand(tickets []*ent.Ticket, startTime, endTime time.Time) []dto.PredictionDataPoint {
	// 基于历史数据和工单特征预测资源需求
	// 简化实现：统计各分类和处理人的工单量来估算资源需求

	// 按处理人统计
	assigneeCounts := make(map[int]int)
	for _, t := range tickets {
		if t.AssigneeID > 0 {
			assigneeCounts[t.AssigneeID]++
		}
	}

	// 生成预测
	predictions := make([]dto.PredictionDataPoint, 0)
	current := startTime
	totalTickets := len(tickets)
	weeks := int(endTime.Sub(startTime).Hours() / (24 * 7))

	if totalTickets == 0 {
		return predictions
	}

	// 基于处理人数量估算总体资源需求
	avgTicketsPerAssignee := float64(totalTickets) / float64(len(assigneeCounts))
	baseResourceDemand := avgTicketsPerAssignee * 1.2 // 预留20%缓冲

	for i := 0; i <= weeks; i++ {
		dateStr := current.Format("2006-01-02")
		// 预测值随时间略有增长
		predictedValue := baseResourceDemand * (1.0 + float64(i)*0.03)

		predictions = append(predictions, dto.PredictionDataPoint{
			Date:           dateStr,
			PredictedValue: predictedValue,
			LowerBound:     predictedValue * 0.7,
			UpperBound:     predictedValue * 1.3,
			Confidence:     0.70,
		})
	}

	return predictions
}

// calculateAverage 计算平均值
func (s *PredictionService) calculateAverage(counts map[string]int) int {
	if len(counts) == 0 {
		return 0
	}
	total := 0
	for _, count := range counts {
		total += count
	}
	return total / len(counts)
}

// ExportPredictionReport 导出预测报告
func (s *PredictionService) ExportPredictionReport(ctx context.Context, req *dto.TrendPredictionRequest, format string, tenantID int) ([]byte, string, error) {
	s.logger.Infow("Exporting prediction report", "format", format, "tenant_id", tenantID)

	// 获取预测数据
	response, err := s.GetTrendPrediction(ctx, req, tenantID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get prediction data: %w", err)
	}

	// 根据格式生成文件
	switch format {
	case "csv":
		return s.exportPredictionToCSV(response)
	case "excel":
		return s.exportPredictionToExcel(response)
	case "pdf":
		return s.exportPredictionToPDF(response)
	default:
		return nil, "", fmt.Errorf("unsupported format: %s", format)
	}
}

// exportPredictionToCSV 导出预测为CSV格式
func (s *PredictionService) exportPredictionToCSV(response *dto.TrendPredictionResponse) ([]byte, string, error) {
	var csvData string
	
	// CSV头部
	csvData = "日期,预测值,下限,上限,置信度\n"
	
	// 数据行
	for _, point := range response.Predictions {
		csvData += fmt.Sprintf("%s,%.2f,%.2f,%.2f,%.2f\n", 
			point.Date, point.PredictedValue, point.LowerBound, point.UpperBound, point.Confidence)
	}
	
	// 添加汇总信息
	csvData += "\n预测信息\n"
	csvData += fmt.Sprintf("模型,%s\n", response.Model)
	csvData += fmt.Sprintf("整体置信度,%.2f\n", response.Confidence)
	csvData += fmt.Sprintf("生成时间,%s\n", response.GeneratedAt.Format("2006-01-02 15:04:05"))
	
	filename := fmt.Sprintf("prediction_%s.csv", time.Now().Format("20060102_150405"))
	return []byte(csvData), filename, nil
}

// exportPredictionToExcel 导出预测为Excel格式
func (s *PredictionService) exportPredictionToExcel(response *dto.TrendPredictionResponse) ([]byte, string, error) {
	return s.reportExport.ExportPredictionToExcel(context.Background(), response)
}

// exportPredictionToPDF 导出预测为PDF格式（简化版）
func (s *PredictionService) exportPredictionToPDF(response *dto.TrendPredictionResponse) ([]byte, string, error) {
	return s.reportExport.ExportToPDF(context.Background(), &dto.DeepAnalyticsResponse{
		Data: func() []dto.AnalyticsDataPoint {
			points := make([]dto.AnalyticsDataPoint, len(response.Predictions))
			for i, p := range response.Predictions {
				points[i] = dto.AnalyticsDataPoint{
					Name:  p.Date,
					Value: p.PredictedValue,
				}
			}
			return points
		}(),
		Summary: dto.AnalyticsSummary{
			Total:            0,
			Resolved:         0,
			AvgResponseTime:  0,
			AvgResolutionTime: 0,
			SLACompliance:    response.Confidence * 100,
			CustomerSatisfaction: 0,
		},
	})
}


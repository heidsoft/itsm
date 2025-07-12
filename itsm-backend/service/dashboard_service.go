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

// DashboardService 仪表盘服务
type DashboardService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewDashboardService 创建仪表盘服务实例
func NewDashboardService(client *ent.Client, logger *zap.SugaredLogger) *DashboardService {
	return &DashboardService{
		client: client,
		logger: logger,
	}
}

// GetDashboardData 获取仪表盘数据
func (s *DashboardService) GetDashboardData(ctx context.Context) (*dto.DashboardResponse, error) {
	// 获取SLA指标
	slaMetrics, err := s.getSLAMetrics(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get SLA metrics", "error", err)
		return nil, fmt.Errorf("获取SLA指标失败: %w", err)
	}

	// 获取事件指标
	incidentMetrics, err := s.getIncidentMetrics(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get incident metrics", "error", err)
		return nil, fmt.Errorf("获取事件指标失败: %w", err)
	}

	// 获取变更指标
	changeMetrics, err := s.getChangeMetrics(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get change metrics", "error", err)
		return nil, fmt.Errorf("获取变更指标失败: %w", err)
	}

	// 获取资源指标
	resourceMetrics, err := s.getResourceMetrics(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get resource metrics", "error", err)
		return nil, fmt.Errorf("获取资源指标失败: %w", err)
	}

	// 构建KPI数据
	kpis := []dto.DashboardKPIResponse{
		{
			Title:  "SLA 达成率",
			Value:  fmt.Sprintf("%.1f%%", slaMetrics.AchievementRate),
			Trend:  s.stringPtr("+1.2%"),
			Period: s.stringPtr("上月"),
			Color:  "text-green-500",
		},
		{
			Title:  "高优先级事件",
			Value:  fmt.Sprintf("%d", incidentMetrics.HighPriorityCount),
			Trend:  s.stringPtr("-3"),
			Period: s.stringPtr("上周"),
			Color:  "text-red-500",
		},
		{
			Title:  "待审批变更",
			Value:  fmt.Sprintf("%d", changeMetrics.PendingApproval),
			Trend:  s.stringPtr("+1"),
			Period: s.stringPtr("昨天"),
			Color:  "text-yellow-500",
		},
		{
			Title:  "纳管云资源",
			Value:  fmt.Sprintf("%d", resourceMetrics.TotalResources),
			Trend:  s.stringPtr("+28"),
			Period: s.stringPtr("上周"),
			Color:  "text-blue-500",
		},
	}

	return &dto.DashboardResponse{
		KPIs:                kpis,
		MultiCloudResources: resourceMetrics.Distribution,
		ResourceHealth:      resourceMetrics.HealthStatus,
		LastUpdated:         time.Now(),
	}, nil
}

// getSLAMetrics 获取SLA指标
func (s *DashboardService) getSLAMetrics(ctx context.Context) (*dto.SLAMetrics, error) {
	// 获取最近30天的工单数据
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	totalTickets, err := s.client.Ticket.Query().
		Where(ticket.CreatedAtGTE(thirtyDaysAgo)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 计算按时解决的工单数（这里简化处理，实际需要根据SLA时间计算）
	resolvedOnTime, err := s.client.Ticket.Query().
		Where(
			ticket.CreatedAtGTE(thirtyDaysAgo),
			ticket.StatusEQ(ticket.StatusClosed),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	var achievementRate float64
	if totalTickets > 0 {
		achievementRate = float64(resolvedOnTime) / float64(totalTickets) * 100
	} else {
		achievementRate = 100.0
	}

	return &dto.SLAMetrics{
		AchievementRate: achievementRate,
		TotalTickets:    totalTickets,
		ResolvedOnTime:  resolvedOnTime,
	}, nil
}

// getIncidentMetrics 获取事件指标
func (s *DashboardService) getIncidentMetrics(ctx context.Context) (*dto.IncidentMetrics, error) {
	// 获取高优先级事件数量
	highPriorityCount, err := s.client.Ticket.Query().
		Where(
			ticket.PriorityIn(ticket.PriorityHigh, ticket.PriorityCritical),
			ticket.StatusNEQ(ticket.StatusClosed),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 获取总事件数
	totalIncidents, err := s.client.Ticket.Query().Count(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.IncidentMetrics{
		HighPriorityCount: highPriorityCount,
		TotalIncidents:    totalIncidents,
		AvgResolutionTime: 240, // 模拟数据：4小时
	}, nil
}

// getChangeMetrics 获取变更指标
func (s *DashboardService) getChangeMetrics(ctx context.Context) (*dto.ChangeMetrics, error) {
	// 获取待审批的变更数量
	pendingApproval, err := s.client.Ticket.Query().
		Where(ticket.StatusEQ(ticket.StatusPending)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 获取总变更数
	totalChanges, err := s.client.Ticket.Query().Count(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.ChangeMetrics{
		PendingApproval: pendingApproval,
		SuccessRate:     95.5, // 模拟数据
		TotalChanges:    totalChanges,
	}, nil
}

// getResourceMetrics 获取资源指标
func (s *DashboardService) getResourceMetrics(ctx context.Context) (*dto.ResourceMetrics, error) {
	// 这里使用模拟数据，实际应该从CMDB或资源管理系统获取
	distribution := []dto.MultiCloudResourceData{
		{Name: "虚拟机", AliCloud: 40, Tencent: 24, Private: 20},
		{Name: "数据库", AliCloud: 22, Tencent: 18, Private: 30},
		{Name: "存储桶", AliCloud: 55, Tencent: 32, Private: 15},
		{Name: "网络", AliCloud: 30, Tencent: 20, Private: 25},
	}

	healthStatus := []dto.ResourceHealthData{
		{Name: "运行中", Value: 400},
		{Name: "已停止", Value: 78},
		{Name: "警告", Value: 32},
		{Name: "错误", Value: 15},
	}

	// 计算总资源数
	totalResources := 0
	for _, health := range healthStatus {
		totalResources += health.Value
	}

	return &dto.ResourceMetrics{
		TotalResources: totalResources,
		ByCloud: map[string]int{
			"阿里云": 147,
			"腾讯云": 94,
			"私有云": 90,
		},
		ByType: map[string]int{
			"虚拟机": 84,
			"数据库": 70,
			"存储桶": 102,
			"网络":  75,
		},
		ByStatus: map[string]int{
			"运行中": 400,
			"已停止": 78,
			"警告":  32,
			"错误":  15,
		},
		Distribution: distribution,
		HealthStatus: healthStatus,
	}, nil
}

// stringPtr 返回字符串指针
func (s *DashboardService) stringPtr(str string) *string {
	return &str
}

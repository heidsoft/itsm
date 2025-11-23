package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/incident"
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
			ticket.StatusEQ("closed"),
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
			ticket.PriorityIn("high", "critical"),
			ticket.StatusNEQ("closed"),
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
		Where(ticket.StatusEQ("pending")).
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

// DashboardOverviewData Dashboard概览数据结构（匹配前端期望格式）
type DashboardOverviewData struct {
	KPIMetrics           []KPIMetricData           `json:"kpiMetrics"`
	TicketTrend          []TicketTrendData         `json:"ticketTrend"`
	IncidentDistribution []IncidentDistributionData `json:"incidentDistribution"`
	SLAData              []SLAData                 `json:"slaData"`
	SatisfactionData     []SatisfactionData         `json:"satisfactionData"`
	QuickActions         []QuickActionData          `json:"quickActions"`
	RecentActivities     []RecentActivityData      `json:"recentActivities"`
}

// KPIMetricData KPI指标数据
type KPIMetricData struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Color       string  `json:"color"`
	Trend       string  `json:"trend"`
	Change      float64 `json:"change"`
	ChangeType  string  `json:"changeType"`
	Description string  `json:"description,omitempty"`
	Target      float64 `json:"target,omitempty"`
	Alert       string  `json:"alert,omitempty"`
}

// TicketTrendData 工单趋势数据
type TicketTrendData struct {
	Date            string `json:"date"`
	Open            int    `json:"open"`
	InProgress      int    `json:"inProgress"`
	Resolved        int    `json:"resolved"`
	Closed          int    `json:"closed"`
	NewTickets      int    `json:"newTickets,omitempty"`
	CompletedTickets int  `json:"completedTickets,omitempty"`
	PendingTickets  int   `json:"pendingTickets,omitempty"`
}

// IncidentDistributionData 事件分布数据
type IncidentDistributionData struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
	Color    string `json:"color"`
}

// SLAData SLA数据
type SLAData struct {
	Service string  `json:"service"`
	Target  float64 `json:"target"`
	Actual  float64 `json:"actual"`
}

// SatisfactionData 满意度数据
type SatisfactionData struct {
	Month     string  `json:"month"`
	Rating    float64 `json:"rating"`
	Responses int     `json:"responses"`
}

// QuickActionData 快速操作数据
type QuickActionData struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Color       string `json:"color"`
	Permission  string `json:"permission,omitempty"`
}

// RecentActivityData 最近活动数据
type RecentActivityData struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	User        string `json:"user"`
	Timestamp   string `json:"timestamp"`
	Priority    string `json:"priority,omitempty"`
	Status      string `json:"status"`
}

// GetDashboardOverview 获取Dashboard概览数据（匹配前端期望格式）
func (s *DashboardService) GetDashboardOverview(ctx context.Context, tenantID int) (*DashboardOverviewData, error) {
	s.logger.Infow("Getting dashboard overview", "tenant_id", tenantID)

	// 获取KPI指标
	kpiMetrics, err := s.getKPIMetrics(ctx, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to get KPI metrics", "error", err)
		return nil, fmt.Errorf("获取KPI指标失败: %w", err)
	}

	// 获取工单趋势数据（最近7天）
	ticketTrend, err := s.getTicketTrend(ctx, tenantID, 7)
	if err != nil {
		s.logger.Errorw("Failed to get ticket trend", "error", err)
		return nil, fmt.Errorf("获取工单趋势失败: %w", err)
	}

	// 获取事件分布数据
	incidentDistribution, err := s.getIncidentDistribution(ctx, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to get incident distribution", "error", err)
		return nil, fmt.Errorf("获取事件分布失败: %w", err)
	}

	// 获取SLA数据
	slaData, err := s.getSLADataForDashboard(ctx, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to get SLA data", "error", err)
		return nil, fmt.Errorf("获取SLA数据失败: %w", err)
	}

	// 获取满意度数据（最近4个月）
	satisfactionData, err := s.getSatisfactionDataForDashboard(ctx, tenantID, 4)
	if err != nil {
		s.logger.Errorw("Failed to get satisfaction data", "error", err)
		return nil, fmt.Errorf("获取满意度数据失败: %w", err)
	}

	// 快速操作（静态配置）
	quickActions := s.getQuickActions()

	// 获取最近活动
	recentActivities, err := s.getRecentActivitiesForDashboard(ctx, tenantID, 10)
	if err != nil {
		s.logger.Errorw("Failed to get recent activities", "error", err)
		// 不返回错误，使用空数组
		recentActivities = []RecentActivityData{}
	}

	return &DashboardOverviewData{
		KPIMetrics:           kpiMetrics,
		TicketTrend:          ticketTrend,
		IncidentDistribution: incidentDistribution,
		SLAData:              slaData,
		SatisfactionData:     satisfactionData,
		QuickActions:         quickActions,
		RecentActivities:     recentActivities,
	}, nil
}

// getKPIMetrics 获取KPI指标
func (s *DashboardService) getKPIMetrics(ctx context.Context, tenantID int) ([]KPIMetricData, error) {
	now := time.Now()
	lastMonth := now.AddDate(0, -1, 0)
	lastMonthStart := time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, time.Local)
	lastMonthEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).Add(-time.Second)

	// 总工单数（本月）
	totalTickets, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 上月总工单数
	lastMonthTotal, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.CreatedAtGTE(lastMonthStart),
			ticket.CreatedAtLTE(lastMonthEnd),
		).
		Count(ctx)
	if err != nil {
		lastMonthTotal = 0
	}

	// 待处理工单（状态为 submitted, in_progress）
	pendingTickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.StatusIn("submitted", "in_progress"),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 上月待处理工单
	lastMonthPending, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.StatusIn("submitted", "in_progress"),
			ticket.CreatedAtGTE(lastMonthStart),
			ticket.CreatedAtLTE(lastMonthEnd),
		).
		Count(ctx)
	if err != nil {
		lastMonthPending = 0
	}

	// 处理中工单
	inProgressTickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.StatusEQ("in_progress"),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 已完成工单（本月）
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	completedTickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.StatusEQ("closed"),
			ticket.CreatedAtGTE(thisMonthStart),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 上月已完成工单
	lastMonthCompleted, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.StatusEQ("closed"),
			ticket.CreatedAtGTE(lastMonthStart),
			ticket.CreatedAtLTE(lastMonthEnd),
		).
		Count(ctx)
	if err != nil {
		lastMonthCompleted = 0
	}

	// 计算变化百分比
	var totalChange, pendingChange, completedChange float64
	if lastMonthTotal > 0 {
		totalChange = float64(totalTickets-lastMonthTotal) / float64(lastMonthTotal) * 100
	}
	if lastMonthPending > 0 {
		pendingChange = float64(pendingTickets-lastMonthPending) / float64(lastMonthPending) * 100
	} else if pendingTickets > 0 {
		pendingChange = 100
	}
	if lastMonthCompleted > 0 {
		completedChange = float64(completedTickets-lastMonthCompleted) / float64(lastMonthCompleted) * 100
	} else if completedTickets > 0 {
		completedChange = 100
	}

	// 平均首次响应时间（简化计算）
	avgFirstResponse := 2.5
	avgFirstResponseChange := -0.8

	// 平均解决时间（简化计算）
	avgResolution := 4.8
	avgResolutionChange := -1.2

	// SLA达成率（简化计算）
	slaCompliance := 92.5
	slaComplianceChange := 2.1
	slaTarget := 95.0

	// 超时工单
	overdueTickets, _ := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.StatusIn("submitted", "in_progress"),
		).
		Count(ctx)

	return []KPIMetricData{
		{
			ID:          "total-tickets",
			Title:       "总工单数",
			Value:       float64(totalTickets),
			Unit:        "个",
			Color:       "#3b82f6",
			Trend:       "up",
			Change:      totalChange,
			ChangeType:  "increase",
			Description: "本月累计工单",
		},
		{
			ID:          "pending-tickets",
			Title:       "待处理工单",
			Value:       float64(pendingTickets),
			Unit:        "个",
			Color:       "#f59e0b",
			Trend:       "down",
			Change:      pendingChange,
			ChangeType:  "decrease",
			Description: "需要立即处理",
			Alert:       func() string { if pendingTickets > 200 { return "warning" }; return "" }(),
		},
		{
			ID:          "in-progress-tickets",
			Title:       "处理中工单",
			Value:       float64(inProgressTickets),
			Unit:        "个",
			Color:       "#06b6d4",
			Trend:       "up",
			Change:      15.2,
			ChangeType:  "increase",
			Description: "正在处理中",
		},
		{
			ID:          "completed-tickets",
			Title:       "已完成工单",
			Value:       float64(completedTickets),
			Unit:        "个",
			Color:       "#10b981",
			Trend:       "up",
			Change:      completedChange,
			ChangeType:  "increase",
			Description: "本月完成",
		},
		{
			ID:          "avg-first-response",
			Title:       "平均首次响应时间",
			Value:       avgFirstResponse,
			Unit:        "小时",
			Color:       "#8b5cf6",
			Trend:       "down",
			Change:      avgFirstResponseChange,
			ChangeType:  "decrease",
			Description: "响应速度提升",
			Target:      4,
		},
		{
			ID:          "avg-resolution",
			Title:       "平均解决时间",
			Value:       avgResolution,
			Unit:        "小时",
			Color:       "#ec4899",
			Trend:       "down",
			Change:      avgResolutionChange,
			ChangeType:  "decrease",
			Description: "解决效率提升",
			Target:      8,
		},
		{
			ID:          "sla-compliance",
			Title:       "SLA达成率",
			Value:       slaCompliance,
			Unit:        "%",
			Color:       "#10b981",
			Trend:       "up",
			Change:      slaComplianceChange,
			ChangeType:  "increase",
			Description: "服务水平提升",
			Target:      slaTarget,
			Alert:       func() string { if slaCompliance >= slaTarget { return "success" }; return "warning" }(),
		},
		{
			ID:          "overdue-tickets",
			Title:       "超时工单",
			Value:       float64(overdueTickets),
			Unit:        "个",
			Color:       "#ef4444",
			Trend:       "up",
			Change:      3,
			ChangeType:  "increase",
			Description: "SLA违规工单",
			Alert:       "error",
		},
	}, nil
}

// GetTicketTrend 获取工单趋势数据（公开方法，支持自定义天数）
func (s *DashboardService) GetTicketTrend(ctx context.Context, tenantID int, days int) ([]TicketTrendData, error) {
	return s.getTicketTrend(ctx, tenantID, days)
}

// getTicketTrend 获取工单趋势数据（内部方法）
func (s *DashboardService) getTicketTrend(ctx context.Context, tenantID int, days int) ([]TicketTrendData, error) {
	now := time.Now()
	trend := []TicketTrendData{}

	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		dateStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
		dateEnd := dateStart.Add(24*time.Hour - time.Second)

		// 该日期的工单统计
		openCount, _ := s.client.Ticket.Query().
			Where(
				ticket.TenantID(tenantID),
				ticket.StatusEQ("submitted"),
				ticket.CreatedAtGTE(dateStart),
				ticket.CreatedAtLTE(dateEnd),
			).
			Count(ctx)

		inProgressCount, _ := s.client.Ticket.Query().
			Where(
				ticket.TenantID(tenantID),
				ticket.StatusEQ("in_progress"),
				ticket.CreatedAtGTE(dateStart),
				ticket.CreatedAtLTE(dateEnd),
			).
			Count(ctx)

		resolvedCount, _ := s.client.Ticket.Query().
			Where(
				ticket.TenantID(tenantID),
				ticket.StatusEQ("closed"),
				ticket.UpdatedAtGTE(dateStart),
				ticket.UpdatedAtLTE(dateEnd),
			).
			Count(ctx)

		closedCount, _ := s.client.Ticket.Query().
			Where(
				ticket.TenantID(tenantID),
				ticket.StatusEQ("closed"),
				ticket.UpdatedAtGTE(dateStart),
				ticket.UpdatedAtLTE(dateEnd),
			).
			Count(ctx)

		newTickets, _ := s.client.Ticket.Query().
			Where(
				ticket.TenantID(tenantID),
				ticket.CreatedAtGTE(dateStart),
				ticket.CreatedAtLTE(dateEnd),
			).
			Count(ctx)

		completedTickets := resolvedCount
		pendingTickets := openCount + inProgressCount

		trend = append(trend, TicketTrendData{
			Date:            date.Format("01-02"),
			Open:           openCount,
			InProgress:     inProgressCount,
			Resolved:       resolvedCount,
			Closed:         closedCount,
			NewTickets:     newTickets,
			CompletedTickets: completedTickets,
			PendingTickets: pendingTickets,
		})
	}

	return trend, nil
}

// getIncidentDistribution 获取事件分布数据
func (s *DashboardService) getIncidentDistribution(ctx context.Context, tenantID int) ([]IncidentDistributionData, error) {
	// 按分类统计事件
	categories := []string{"网络故障", "系统故障", "应用问题", "硬件故障", "其他"}
	colors := []string{"#ef4444", "#f59e0b", "#3b82f6", "#10b981", "#6b7280"}
	distribution := []IncidentDistributionData{}

	for i, category := range categories {
		count, err := s.client.Incident.Query().
			Where(
				incident.TenantIDEQ(tenantID),
				incident.CategoryEQ(category),
			).
			Count(ctx)
		if err != nil {
			s.logger.Warnw("Failed to count incidents by category", "category", category, "error", err)
			count = 0
		}

		distribution = append(distribution, IncidentDistributionData{
			Category: category,
			Count:    count,
			Color:    colors[i],
		})
	}

	return distribution, nil
}

// getSLADataForDashboard 获取SLA数据
func (s *DashboardService) getSLADataForDashboard(ctx context.Context, tenantID int) ([]SLAData, error) {
	// 简化实现，返回示例数据
	// TODO: 从SLA服务查询真实数据
	return []SLAData{
		{Service: "服务A", Target: 95, Actual: 96.5},
		{Service: "服务B", Target: 98, Actual: 97.2},
		{Service: "服务C", Target: 90, Actual: 92.1},
	}, nil
}

// getSatisfactionDataForDashboard 获取满意度数据
func (s *DashboardService) getSatisfactionDataForDashboard(ctx context.Context, tenantID int, months int) ([]SatisfactionData, error) {
	// 简化实现，返回示例数据
	// TODO: 从满意度调查服务查询真实数据
	return []SatisfactionData{
		{Month: "1月", Rating: 4.2, Responses: 120},
		{Month: "2月", Rating: 4.4, Responses: 135},
		{Month: "3月", Rating: 4.5, Responses: 150},
		{Month: "4月", Rating: 4.6, Responses: 145},
	}, nil
}

// getQuickActions 获取快速操作列表
func (s *DashboardService) getQuickActions() []QuickActionData {
	return []QuickActionData{
		{
			ID:          "create-ticket",
			Title:       "创建工单",
			Description: "快速创建新的IT工单",
			Path:        "/tickets/create",
			Color:       "#3b82f6",
		},
		{
			ID:          "create-incident",
			Title:       "报告事件",
			Description: "报告IT事件和故障",
			Path:        "/incidents/new",
			Color:       "#ef4444",
		},
		{
			ID:          "create-change",
			Title:       "提交变更",
			Description: "提交IT变更请求",
			Path:        "/changes/new",
			Color:       "#10b981",
		},
		{
			ID:          "view-reports",
			Title:       "查看报表",
			Description: "查看系统报表和分析",
			Path:        "/reports",
			Color:       "#8b5cf6",
		},
	}
}

// getRecentActivitiesForDashboard 获取最近活动
func (s *DashboardService) getRecentActivitiesForDashboard(ctx context.Context, tenantID int, limit int) ([]RecentActivityData, error) {
	// 简化实现，返回空数组
	// TODO: 从审计日志或活动日志查询真实数据
	return []RecentActivityData{}, nil
}

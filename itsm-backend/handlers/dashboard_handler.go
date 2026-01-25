package handlers

import (
	"net/http"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DashboardHandler Dashboard相关的HTTP处理器
type DashboardHandler struct {
	dashboardService *service.DashboardService
	ticketService    *service.TicketService
	incidentService  *service.IncidentService
	logger           *zap.SugaredLogger
}

// ErrorResponse 通用错误响应（用于 Swagger 文档声明）
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewDashboardHandler 创建Dashboard处理器
func NewDashboardHandler(
	dashboardService *service.DashboardService,
	ticketService *service.TicketService,
	incidentService *service.IncidentService,
	logger *zap.SugaredLogger,
) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
		ticketService:    ticketService,
		incidentService:  incidentService,
		logger:           logger,
	}
}

// KPIMetric KPI指标
type KPIMetric struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Color       string  `json:"color"`
	Trend       string  `json:"trend"`        // up, down, stable
	Change      float64 `json:"change"`       // 变化百分比
	ChangeType  string  `json:"changeType"`   // increase, decrease, stable
	Description string  `json:"description"`
}

// TicketTrendData 工单趋势数据
type TicketTrendData struct {
	Date       string `json:"date"`
	Open       int    `json:"open"`
	InProgress int    `json:"inProgress"`
	Resolved   int    `json:"resolved"`
	Closed     int    `json:"closed"`
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

// QuickAction 快速操作
type QuickAction struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Color       string `json:"color"`
	Permission  string `json:"permission,omitempty"`
}

// RecentActivity 最近活动
type RecentActivity struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // ticket, incident, change, problem
	Title       string `json:"title"`
	Description string `json:"description"`
	User        string `json:"user"`
	Timestamp   string `json:"timestamp"`
	Priority    string `json:"priority,omitempty"`
	Status      string `json:"status"`
}

// ResponseTimeDistributionData 响应时间分布数据
type ResponseTimeDistributionData struct {
	TimeRange  string  `json:"timeRange"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
	AvgTime    float64 `json:"avgTime,omitempty"`
}

// TeamWorkloadData 团队工作负载数据
type TeamWorkloadData struct {
	Assignee        string  `json:"assignee"`
	TicketCount     int     `json:"ticketCount"`
	AvgResponseTime float64 `json:"avgResponseTime"`
	CompletionRate  float64 `json:"completionRate"`
	ActiveTickets   int     `json:"activeTickets,omitempty"`
}

// PeakHourData 高峰时段数据
type PeakHourData struct {
	Hour            string  `json:"hour"`
	Count           int     `json:"count"`
	AvgResponseTime float64 `json:"avgResponseTime,omitempty"`
}

// DashboardOverview Dashboard概览数据
type DashboardOverview struct {
	KPIMetrics              []KPIMetric                    `json:"kpiMetrics"`
	TicketTrend             []TicketTrendData              `json:"ticketTrend"`
	IncidentDistribution    []IncidentDistributionData    `json:"incidentDistribution"`
	SLAData                 []SLAData                      `json:"slaData"`
	SatisfactionData        []SatisfactionData             `json:"satisfactionData"`
	QuickActions            []QuickAction                  `json:"quickActions"`
	RecentActivities        []RecentActivity               `json:"recentActivities"`
	ResponseTimeDistribution []ResponseTimeDistributionData `json:"responseTimeDistribution,omitempty"`
	TeamWorkload            []TeamWorkloadData             `json:"teamWorkload,omitempty"`
	PeakHours               []PeakHourData                 `json:"peakHours,omitempty"`
}

// GetOverview 获取Dashboard概览数据
// @Summary 获取Dashboard概览
// @Description 获取Dashboard页面的所有数据，包括KPI指标、趋势图表、快速操作等
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {object} DashboardOverview
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/dashboard/overview [get]
func (h *DashboardHandler) GetOverview(c *gin.Context) {
	// 获取租户ID
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		h.logger.Errorw("Failed to get tenant ID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    5001,
			"message": "获取租户ID失败",
		})
		return
	}

	// 从数据库查询真实数据
	overviewData, err := h.dashboardService.GetDashboardOverview(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get dashboard overview", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    5001,
			"message": "获取Dashboard数据失败",
		})
		return
	}

	// 转换为Handler期望的格式
	overview := DashboardOverview{
		KPIMetrics:              convertKPIMetrics(overviewData.KPIMetrics),
		TicketTrend:               convertTicketTrend(overviewData.TicketTrend),
		IncidentDistribution:      convertIncidentDistribution(overviewData.IncidentDistribution),
		SLAData:                   convertSLAData(overviewData.SLAData),
		SatisfactionData:           convertSatisfactionData(overviewData.SatisfactionData),
		QuickActions:               convertQuickActions(overviewData.QuickActions),
		RecentActivities:          convertRecentActivities(overviewData.RecentActivities),
		ResponseTimeDistribution:   convertResponseTimeDistribution(overviewData.ResponseTimeDistribution),
		TeamWorkload:              convertTeamWorkload(overviewData.TeamWorkload),
		PeakHours:                 convertPeakHours(overviewData.PeakHours),
	}

	// 使用统一的响应格式
	common.Success(c, overview)
}

// 转换函数：将Service层的数据结构转换为Handler层的数据结构
func convertKPIMetrics(metrics []service.KPIMetricData) []KPIMetric {
	result := make([]KPIMetric, len(metrics))
	for i, m := range metrics {
		result[i] = KPIMetric{
			ID:          m.ID,
			Title:       m.Title,
			Value:       m.Value,
			Unit:        m.Unit,
			Color:       m.Color,
			Trend:       m.Trend,
			Change:      m.Change,
			ChangeType:  m.ChangeType,
			Description: m.Description,
		}
	}
	return result
}

func convertTicketTrend(trend []service.TicketTrendData) []TicketTrendData {
	result := make([]TicketTrendData, len(trend))
	for i, t := range trend {
		result[i] = TicketTrendData{
			Date:       t.Date,
			Open:       t.Open,
			InProgress: t.InProgress,
			Resolved:   t.Resolved,
			Closed:     t.Closed,
		}
	}
	return result
}

func convertIncidentDistribution(dist []service.IncidentDistributionData) []IncidentDistributionData {
	result := make([]IncidentDistributionData, len(dist))
	for i, d := range dist {
		result[i] = IncidentDistributionData{
			Category: d.Category,
			Count:    d.Count,
			Color:    d.Color,
		}
	}
	return result
}

func convertQuickActions(actions []service.QuickActionData) []QuickAction {
	result := make([]QuickAction, len(actions))
	for i, a := range actions {
		result[i] = QuickAction{
			ID:          a.ID,
			Title:       a.Title,
			Description: a.Description,
			Path:        a.Path,
			Color:       a.Color,
			Permission:  a.Permission,
		}
	}
	return result
}

func convertRecentActivities(activities []service.RecentActivityData) []RecentActivity {
	result := make([]RecentActivity, len(activities))
	for i, a := range activities {
		result[i] = RecentActivity{
			ID:          strconv.Itoa(a.ID),
			Type:        a.Type,
			Title:       a.TicketTitle,
			Description: a.TicketNumber,
			User:        a.Operator,
			Timestamp:   a.UpdatedAt.Format("2006-01-02 15:04:05"),
			Priority:    "",
			Status:      a.StatusName,
		}
	}
	return result
}

func convertSLAData(slaData []service.SLAData) []SLAData {
	result := make([]SLAData, len(slaData))
	for i, s := range slaData {
		result[i] = SLAData{
			Service: s.Service,
			Target:  s.Target,
			Actual:  s.Actual,
		}
	}
	return result
}

func convertSatisfactionData(satisfactionData []service.SatisfactionData) []SatisfactionData {
	result := make([]SatisfactionData, len(satisfactionData))
	for i, s := range satisfactionData {
		result[i] = SatisfactionData{
			Month:     s.Month,
			Rating:    s.Rating,
			Responses: s.Responses,
		}
	}
	return result
}

func convertResponseTimeDistribution(data []service.ResponseTimeDistributionData) []ResponseTimeDistributionData {
	result := make([]ResponseTimeDistributionData, len(data))
	for i, d := range data {
		result[i] = ResponseTimeDistributionData{
			TimeRange:  d.TimeRange,
			Count:      d.Count,
			Percentage: d.Percentage,
			AvgTime:    d.AvgTime,
		}
	}
	return result
}

func convertTeamWorkload(data []service.TeamWorkloadData) []TeamWorkloadData {
	result := make([]TeamWorkloadData, len(data))
	for i, d := range data {
		result[i] = TeamWorkloadData{
			Assignee:        d.Assignee,
			TicketCount:     d.TicketCount,
			AvgResponseTime: d.AvgResponseTime,
			CompletionRate:  d.CompletionRate,
			ActiveTickets:   d.ActiveTickets,
		}
	}
	return result
}

func convertPeakHours(data []service.PeakHourData) []PeakHourData {
	result := make([]PeakHourData, len(data))
	for i, d := range data {
		result[i] = PeakHourData{
			Hour:            d.Hour,
			Count:           d.Count,
			AvgResponseTime: d.AvgResponseTime,
		}
	}
	return result
}

// GetKPIMetrics 获取KPI指标数据
// @Summary 获取KPI指标
// @Description 获取Dashboard的KPI指标数据
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {array} KPIMetric
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/dashboard/kpi-metrics [get]
func (h *DashboardHandler) GetKPIMetrics(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		h.logger.Errorw("Failed to get tenant ID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "获取租户ID失败"})
		return
	}

	overviewData, err := h.dashboardService.GetDashboardOverview(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get dashboard overview", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "获取KPI指标失败"})
		return
	}

	c.JSON(http.StatusOK, convertKPIMetrics(overviewData.KPIMetrics))
}

// GetTicketTrend 获取工单趋势数据
// @Summary 获取工单趋势
// @Description 获取最近几天的工单趋势数据
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param days query int false "天数" default(7)
// @Success 200 {array} TicketTrendData
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/dashboard/ticket-trend [get]
func (h *DashboardHandler) GetTicketTrend(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		h.logger.Errorw("Failed to get tenant ID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "获取租户ID失败"})
		return
	}

	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 7
	}

	trend, err := h.dashboardService.GetTicketTrend(c.Request.Context(), tenantID, days)
	if err != nil {
		h.logger.Errorw("Failed to get ticket trend", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "获取工单趋势失败"})
		return
	}

	c.JSON(http.StatusOK, convertTicketTrend(trend))
}

// GetIncidentDistribution 获取事件分布数据
// @Summary 获取事件分布
// @Description 获取事件类型分布统计
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {array} IncidentDistributionData
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/dashboard/incident-distribution [get]
func (h *DashboardHandler) GetIncidentDistribution(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		h.logger.Errorw("Failed to get tenant ID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "获取租户ID失败"})
		return
	}

	overviewData, err := h.dashboardService.GetDashboardOverview(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get incident distribution", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "获取事件分布失败"})
		return
	}

	c.JSON(http.StatusOK, convertIncidentDistribution(overviewData.IncidentDistribution))
}

// GetSLAData 获取SLA数据
// @Summary 获取SLA数据
// @Description 获取各服务的SLA达成情况
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {array} SLAData
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/dashboard/sla-data [get]
func (h *DashboardHandler) GetSLAData(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		h.logger.Errorw("Failed to get tenant ID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "获取租户ID失败"})
		return
	}

	overviewData, err := h.dashboardService.GetDashboardOverview(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get SLA data", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "获取SLA数据失败"})
		return
	}

	c.JSON(http.StatusOK, convertSLAData(overviewData.SLAData))
}

// GetSatisfactionData 获取满意度数据
// @Summary 获取满意度数据
// @Description 获取用户满意度统计数据
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param months query int false "月数" default(4)
// @Success 200 {array} SatisfactionData
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/dashboard/satisfaction-data [get]
func (h *DashboardHandler) GetSatisfactionData(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		h.logger.Errorw("Failed to get tenant ID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "获取租户ID失败"})
		return
	}

	monthsStr := c.DefaultQuery("months", "4")
	months, err := strconv.Atoi(monthsStr)
	if err != nil || months <= 0 {
		months = 4
	}

	overviewData, err := h.dashboardService.GetDashboardOverview(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get satisfaction data", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "获取满意度数据失败"})
		return
	}

	data := convertSatisfactionData(overviewData.SatisfactionData)

	// 只返回请求的月数
	if len(data) > months {
		data = data[len(data)-months:]
	}

	c.JSON(http.StatusOK, data)
}

// GetQuickActions 获取快速操作列表
// @Summary 获取快速操作
// @Description 获取Dashboard快速操作按钮配置
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {array} QuickAction
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/dashboard/quick-actions [get]
func (h *DashboardHandler) GetQuickActions(c *gin.Context) {
	// 从上下文获取用户权限
	permissions, _ := c.Get("permissions")
	userPermissions, ok := permissions.([]string)
	if !ok {
		userPermissions = []string{}
	}

	allActions := []QuickAction{
		{
			ID:          "create-ticket",
			Title:       "创建工单",
			Description: "快速创建新的IT工单",
			Path:        "/tickets/create",
			Color:       "#3b82f6",
			Permission:  "ticket:create",
		},
		{
			ID:          "create-incident",
			Title:       "报告事件",
			Description: "报告IT事件和故障",
			Path:        "/incidents/new",
			Color:       "#ef4444",
			Permission:  "incident:create",
		},
		{
			ID:          "create-change",
			Title:       "提交变更",
			Description: "提交IT变更请求",
			Path:        "/changes/new",
			Color:       "#10b981",
			Permission:  "change:create",
		},
		{
			ID:          "view-reports",
			Title:       "查看报表",
			Description: "查看系统报表和分析",
			Path:        "/reports",
			Color:       "#8b5cf6",
			Permission:  "report:view",
		},
	}

	// 根据用户权限过滤可用的操作
	filteredActions := []QuickAction{}
	permissionSet := make(map[string]bool)
	for _, p := range userPermissions {
		permissionSet[p] = true
	}

	for _, action := range allActions {
		if _, hasPermission := permissionSet[action.Permission]; hasPermission || action.Permission == "" {
			filteredActions = append(filteredActions, action)
		}
	}

	c.JSON(http.StatusOK, filteredActions)
}

// GetRecentActivities 获取最近活动
// @Summary 获取最近活动
// @Description 获取系统最近的操作活动记录
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param limit query int false "限制数量" default(10)
// @Success 200 {array} RecentActivity
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/dashboard/recent-activities [get]
func (h *DashboardHandler) GetRecentActivities(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		h.logger.Errorw("Failed to get tenant ID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "获取租户ID失败"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	overviewData, err := h.dashboardService.GetDashboardOverview(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get recent activities", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "获取最近活动失败"})
		return
	}

	activities := convertRecentActivities(overviewData.RecentActivities)

	// 限制返回数量
	if len(activities) > limit {
		activities = activities[:limit]
	}

	c.JSON(http.StatusOK, activities)
}


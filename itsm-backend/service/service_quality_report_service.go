package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
)

// ServiceQualityMetricVersion 管理者统一服务质量视图的总体口径版本。
//
// 口径版本（metric version）的语义：本响应里任何一个数字的**计算语义**发生变化
// （统计窗口、过滤条件、合规率分母、环比基线等），都必须递增对应版本号，让
// 下游（管理层月报、外部对账脚本）能感知「同样的数字现在含义不同了」。
// 只修 bug 不改变正常输入下输出语义的改动不需要递增。
//
// 版本记录：
//   - 1.0（2026-10-23）：首版。KPI=工作台行级口径（datascope.ForTicketReader），
//     SLA 合规率=sla_data 全量口径，趋势按自然日聚合，满意度按月聚合。
const ServiceQualityMetricVersion = "1.0"

// serviceQualityBlockVersions 各数据块的独立口径版本。块内语义变化只递增块版本，
// 不牵连整体版本——下游只消费某一快时可只盯自己那列。
var serviceQualityBlockVersions = map[string]string{
	"kpi":                  "1.0", // 行级范围口径（scope 字段标注 tenant/own），环比基线=上一窗口
	"sla":                  "1.0", // 合规率分母=窗口内有 sla_states 记录的工单
	"incidentDistribution": "1.0", // 按事件分类计数，全租户
	"satisfaction":         "1.0", // 按月聚合，窗口内最近 N 月
	"ticketTrend":          "1.0", // 自然日聚合，窗口=请求 days
	"ticketStats":          "1.0", // 状态/优先级分布 + 30 天趋势（analytics 既有口径）
}

// ServiceQualityWindow 报表统计窗口。
type ServiceQualityWindow struct {
	Days int       `json:"days"`
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// ServiceQualityReport 管理者统一服务质量视图。
//
// 定位：把此前散落在 /reports/tickets、/reports/sla、/reports/incidents、
// 工作台 KPI 与 analytics 工单统计里的局部指标合成**一次请求一张纸**的管理视图。
// 全部复用既有服务的既有口径（聚合为主，不发明新指标）——这是它与各单域报表的
// 唯一区别，也是 metric_version 存在的原因：聚合器的稳定性独立于各域。
type ServiceQualityReport struct {
	MetricVersion  string            `json:"metricVersion"`
	MetricVersions map[string]string `json:"metricVersions"`
	GeneratedAt    time.Time         `json:"generatedAt"`
	Window         ServiceQualityWindow `json:"window"`
	Scope          string            `json:"scope"` // KPI 行级口径：tenant | own

	KPI                  []KPIMetricData            `json:"kpi"`
	SLA                  *SLAComplianceData         `json:"sla"`
	IncidentDistribution []IncidentDistributionData `json:"incidentDistribution"`
	Satisfaction         []SatisfactionData         `json:"satisfaction"`
	TicketTrend          []TicketTrendData          `json:"ticketTrend"`
	TicketStats          *dto.AnalyticsTicketStatsResponse `json:"ticketStats,omitempty"` // analytics 未装配时缺省
}

// DashboardOverviewSource 统一视图对工作台服务的窄接口：只依赖「概览 + 趋势」
// 两个用例。*DashboardService 天然实现；测试注入假实现，避免绑定 PG 方言 raw SQL。
type DashboardOverviewSource interface {
	GetDashboardOverview(ctx context.Context, tenantID int, viewer DashboardViewer) (*DashboardOverviewData, error)
	GetTicketTrend(ctx context.Context, tenantID, days int) ([]TicketTrendData, error)
}

// ServiceQualityReportService 管理者统一服务质量视图的聚合器。
// 依赖全部可选：dashboard 未装配时无法产出（报错），analytics 未装配时
// ticketStats 块缺省——聚合视图的能力降级是显式可见的（块缺失），
// 不是静默给半份数据冒充全量。
type ServiceQualityReportService struct {
	dashboard DashboardOverviewSource // nil → Build 显式报错
	analytics *AnalyticsService       // nil-able：未装配时 ticketStats 块缺省
	now       func() time.Time
}

// NewServiceQualityReportService 创建聚合器。analytics 可为 nil。
func NewServiceQualityReportService(dashboard DashboardOverviewSource, analytics *AnalyticsService) *ServiceQualityReportService {
	return &ServiceQualityReportService{
		dashboard: dashboard,
		analytics: analytics,
		now:       time.Now,
	}
}

// MaxQualityWindowDays 窗口上限：管理者视图是给「看一段趋势」用的，
// 不是全历史导出口；超出的历史走各单域报表或导出通道。
const MaxQualityWindowDays = 90

// Build 组装统一服务质量视图。viewer 沿用工作台同一行级规则——
// 管理角色拿全租户口径，坐席拿本人口径，scope 字段显式回传。
func (s *ServiceQualityReportService) Build(ctx context.Context, tenantID int, viewer DashboardViewer, days int) (*ServiceQualityReport, error) {
	if s.dashboard == nil {
		return nil, fmt.Errorf("dashboard service not configured")
	}
	if days <= 0 {
		days = 30
	}
	if days > MaxQualityWindowDays {
		days = MaxQualityWindowDays
	}

	to := s.now()
	from := to.AddDate(0, 0, -days)

	overview, err := s.dashboard.GetDashboardOverview(ctx, tenantID, viewer)
	if err != nil {
		return nil, fmt.Errorf("build overview block: %w", err)
	}
	trend, err := s.dashboard.GetTicketTrend(ctx, tenantID, days)
	if err != nil {
		return nil, fmt.Errorf("build trend block: %w", err)
	}

	report := &ServiceQualityReport{
		MetricVersion:  ServiceQualityMetricVersion,
		MetricVersions: serviceQualityBlockVersions,
		GeneratedAt:    to,
		Window:         ServiceQualityWindow{Days: days, From: from, To: to},
		Scope:          viewer.scopeKey(),
		KPI:            overview.KPIMetrics,
		SLA:            overview.SLAData,
		IncidentDistribution: overview.IncidentDistribution,
		Satisfaction:   overview.SatisfactionData,
		TicketTrend:    trend,
	}
	if report.KPI == nil {
		report.KPI = []KPIMetricData{}
	}
	if report.IncidentDistribution == nil {
		report.IncidentDistribution = []IncidentDistributionData{}
	}
	if report.Satisfaction == nil {
		report.Satisfaction = []SatisfactionData{}
	}
	if report.TicketTrend == nil {
		report.TicketTrend = []TicketTrendData{}
	}

	if s.analytics != nil {
		stats, statsErr := s.analytics.GetTicketStats(ctx, tenantID)
		if statsErr != nil {
			return nil, fmt.Errorf("build ticketStats block: %w", statsErr)
		}
		report.TicketStats = stats
	}
	return report, nil
}

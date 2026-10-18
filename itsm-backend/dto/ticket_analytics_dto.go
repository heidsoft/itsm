package dto

import "time"

// 深度数据分析相关DTO（使用DeepAnalytics前缀避免与ticket_dto.go中的TicketAnalytics冲突）
type DeepAnalyticsRequest struct {
	Dimensions []string `json:"dimensions" binding:"omitempty,min=1" example:"[\"status\",\"priority\"]"`
	Metrics    []string `json:"metrics" binding:"omitempty,min=1" example:"[\"count\",\"response_time\"]"`
	ChartType  string   `json:"chartType" binding:"omitempty,oneof=line bar pie area table" example:"bar"`
	// timeRange 是深度分析的必填前置：服务层要求恰好 2 个日期（起、止）。
	// 原先写 omitempty，缺省时能过绑定校验，一路走到 service 才报错，
	// 再被 handler 兜底成 500（2026-10-18 缺陷 P1-5）。改 required 后
	// 缺省在绑定阶段即返回 400，且带出缺失字段名。
	TimeRange []string               `json:"timeRange" binding:"required,len=2" example:"[\"2024-01-01\",\"2024-01-31\"]"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	GroupBy   *string                `json:"groupBy,omitempty" example:"status"`
	Page      int                    `json:"page" example:"1"`
	PageSize  int                    `json:"pageSize" example:"20"`
}

type DeepAnalyticsResponse struct {
	Data        []AnalyticsDataPoint `json:"data"`
	Summary     AnalyticsSummary     `json:"summary"`
	GeneratedAt time.Time            `json:"generatedAt" example:"2024-01-01T00:00:00Z"`
}

type AnalyticsDataPoint struct {
	Name     string                 `json:"name" example:"待处理"`
	Value    float64                `json:"value" example:"45.0"`
	Count    int                    `json:"count,omitempty" example:"45"`
	AvgTime  *float64               `json:"avgTime,omitempty" example:"2.3"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type AnalyticsSummary struct {
	Total                int     `json:"total" example:"294"`
	Resolved             int     `json:"resolved" example:"217"`
	AvgResponseTime      float64 `json:"avgResponseTime" example:"2.3"`
	AvgResolutionTime    float64 `json:"avgResolutionTime" example:"8.7"`
	SLACompliance        float64 `json:"slaCompliance" example:"94.2"`
	CustomerSatisfaction float64 `json:"customerSatisfaction" example:"4.2"`
}

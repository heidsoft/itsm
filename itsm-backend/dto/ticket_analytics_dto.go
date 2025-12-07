package dto

import "time"

// 深度数据分析相关DTO（使用DeepAnalytics前缀避免与ticket_dto.go中的TicketAnalytics冲突）
type DeepAnalyticsRequest struct {
	Dimensions   []string               `json:"dimensions" binding:"required,min=1" example:"[\"status\",\"priority\"]"`
	Metrics      []string               `json:"metrics" binding:"required,min=1" example:"[\"count\",\"response_time\"]"`
	ChartType    string                 `json:"chart_type" binding:"required,oneof=line bar pie area table" example:"bar"`
	TimeRange    []string               `json:"time_range" binding:"required,len=2" example:"[\"2024-01-01\",\"2024-01-31\"]"`
	Filters      map[string]interface{} `json:"filters,omitempty"`
	GroupBy      *string                `json:"group_by,omitempty" example:"status"`
	Page         int                    `json:"page" example:"1"`
	PageSize     int                    `json:"page_size" example:"20"`
}

type DeepAnalyticsResponse struct {
	Data        []AnalyticsDataPoint    `json:"data"`
	Summary     AnalyticsSummary         `json:"summary"`
	GeneratedAt time.Time               `json:"generated_at" example:"2024-01-01T00:00:00Z"`
}

type AnalyticsDataPoint struct {
	Name  string                 `json:"name" example:"待处理"`
	Value float64                `json:"value" example:"45.0"`
	Count int                    `json:"count,omitempty" example:"45"`
	AvgTime *float64             `json:"avg_time,omitempty" example:"2.3"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type AnalyticsSummary struct {
	Total              int     `json:"total" example:"294"`
	Resolved            int     `json:"resolved" example:"217"`
	AvgResponseTime     float64 `json:"avg_response_time" example:"2.3"`
	AvgResolutionTime   float64 `json:"avg_resolution_time" example:"8.7"`
	SLACompliance       float64 `json:"sla_compliance" example:"94.2"`
	CustomerSatisfaction float64 `json:"customer_satisfaction" example:"4.2"`
}


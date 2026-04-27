package dto

import "time"

// ProblemTrendRequest 问题趋势分析请求
type ProblemTrendRequest struct {
	StartDate string `json:"start_date" form:"start_date" binding:"required,datetime=2006-01-02"`
	EndDate   string `json:"end_date" form:"end_date" binding:"required,datetime=2006-01-02"`
}

// ProblemTrendResponse 问题趋势分析响应
type ProblemTrendResponse struct {
	Period            string                        `json:"period"`
	TotalProblems     int                           `json:"total_problems"`
	ResolvedProblems  int                           `json:"resolved_problems"`
	OpenProblems      int                           `json:"open_problems"`
	ResolutionRate    float64                       `json:"resolution_rate"`
	AvgResolutionTime float64                       `json:"avg_resolution_time_hours"`
	CategoryBreakdown map[string]int                `json:"category_breakdown"`
	PriorityBreakdown map[string]int                `json:"priority_breakdown"`
	TrendDirection    string                        `json:"trend_direction"`
	TopCategories     []CategoryCountResponse       `json:"top_categories"`
	MonthlyTrend      []MonthlyProblemCountResponse `json:"monthly_trend"`
}

// CategoryCountResponse 分类统计响应
type CategoryCountResponse struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// MonthlyProblemCountResponse 月度问题统计响应
type MonthlyProblemCountResponse struct {
	Month    string `json:"month"`
	Count    int    `json:"count"`
	Resolved int    `json:"resolved"`
	Open     int    `json:"open"`
}

// ProblemHotspotsResponse 问题热点响应
type ProblemHotspotsResponse struct {
	PeriodStart       string            `json:"period_start"`
	PeriodEnd         string            `json:"period_end"`
	CategoryBreakdown map[string]int    `json:"category_breakdown"`
	PriorityBreakdown map[string]int    `json:"priority_breakdown"`
	Hotspots          []string          `json:"hotspots"`
	AvgPerCategory    float64           `json:"avg_per_category"`
}

// ProblemPredictionResponse 问题预测响应
type ProblemPredictionResponse struct {
	Predictions map[string]int `json:"predictions"`
	Method      string         `json:"method"`
	GeneratedAt time.Time      `json:"generated_at"`
}

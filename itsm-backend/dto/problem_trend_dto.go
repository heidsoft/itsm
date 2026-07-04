package dto

import "time"

// ProblemTrendRequest 问题趋势分析请求
type ProblemTrendRequest struct {
	StartDate string `json:"startDate" form:"start_date" binding:"required,datetime=2006-01-02"`
	EndDate   string `json:"endDate" form:"end_date" binding:"required,datetime=2006-01-02"`
}

// ProblemTrendResponse 问题趋势分析响应
type ProblemTrendResponse struct {
	Period            string                        `json:"period"`
	TotalProblems     int                           `json:"totalProblems"`
	ResolvedProblems  int                           `json:"resolvedProblems"`
	OpenProblems      int                           `json:"openProblems"`
	ResolutionRate    float64                       `json:"resolutionRate"`
	AvgResolutionTime float64                       `json:"avgResolutionTimeHours"`
	CategoryBreakdown map[string]int                `json:"categoryBreakdown"`
	PriorityBreakdown map[string]int                `json:"priorityBreakdown"`
	TrendDirection    string                        `json:"trendDirection"`
	TopCategories     []CategoryCountResponse       `json:"topCategories"`
	MonthlyTrend      []MonthlyProblemCountResponse `json:"monthlyTrend"`
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
	PeriodStart       string         `json:"periodStart"`
	PeriodEnd         string         `json:"periodEnd"`
	CategoryBreakdown map[string]int `json:"categoryBreakdown"`
	PriorityBreakdown map[string]int `json:"priorityBreakdown"`
	Hotspots          []string       `json:"hotspots"`
	AvgPerCategory    float64        `json:"avgPerCategory"`
}

// ProblemPredictionResponse 问题预测响应
type ProblemPredictionResponse struct {
	Predictions map[string]int `json:"predictions"`
	Method      string         `json:"method"`
	GeneratedAt time.Time      `json:"generatedAt"`
}

package dto

import "time"

// DashboardKPIResponse 仪表盘KPI响应
type DashboardKPIResponse struct {
	Title  string  `json:"title"`
	Value  string  `json:"value"`
	Trend  *string `json:"trend,omitempty"`
	Period *string `json:"period,omitempty"`
	Color  string  `json:"color"`
}

// MultiCloudResourceData 多云资源分布数据
type MultiCloudResourceData struct {
	Name     string `json:"name"`
	AliCloud int    `json:"阿里云"`
	Tencent  int    `json:"腾讯云"`
	Private  int    `json:"私有云"`
}

// ResourceHealthData 资源健康状态数据
type ResourceHealthData struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// DashboardResponse 仪表盘完整响应
type DashboardResponse struct {
	KPIs                []DashboardKPIResponse   `json:"kpis"`
	MultiCloudResources []MultiCloudResourceData `json:"multiCloudResources"`
	ResourceHealth      []ResourceHealthData     `json:"resourceHealth"`
	LastUpdated         time.Time                `json:"lastUpdated"`
}

// SLAMetrics SLA指标
type SLAMetrics struct {
	AchievementRate float64 `json:"achievementRate"`
	TotalTickets    int     `json:"totalTickets"`
	ResolvedOnTime  int     `json:"resolvedOnTime"`
}

// IncidentMetrics 事件指标
type IncidentMetrics struct {
	HighPriorityCount int `json:"highPriorityCount"`
	TotalIncidents    int `json:"totalIncidents"`
	AvgResolutionTime int `json:"avgResolutionTime"` // 分钟
}

// ChangeMetrics 变更指标
type ChangeMetrics struct {
	PendingApproval int     `json:"pendingApproval"`
	SuccessRate     float64 `json:"successRate"`
	TotalChanges    int     `json:"totalChanges"`
}

// ResourceMetrics 资源指标
type ResourceMetrics struct {
	TotalResources int                      `json:"totalResources"`
	ByCloud        map[string]int           `json:"byCloud"`
	ByType         map[string]int           `json:"byType"`
	ByStatus       map[string]int           `json:"byStatus"`
	Distribution   []MultiCloudResourceData `json:"distribution"`
	HealthStatus   []ResourceHealthData     `json:"healthStatus"`
}

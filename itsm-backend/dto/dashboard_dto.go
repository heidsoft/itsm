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
	MultiCloudResources []MultiCloudResourceData `json:"multi_cloud_resources"`
	ResourceHealth      []ResourceHealthData     `json:"resource_health"`
	LastUpdated         time.Time                `json:"last_updated"`
}

// SLAMetrics SLA指标
type SLAMetrics struct {
	AchievementRate float64 `json:"achievement_rate"`
	TotalTickets    int     `json:"total_tickets"`
	ResolvedOnTime  int     `json:"resolved_on_time"`
}

// IncidentMetrics 事件指标
type IncidentMetrics struct {
	HighPriorityCount int `json:"high_priority_count"`
	TotalIncidents    int `json:"total_incidents"`
	AvgResolutionTime int `json:"avg_resolution_time"` // 分钟
}

// ChangeMetrics 变更指标
type ChangeMetrics struct {
	PendingApproval int     `json:"pending_approval"`
	SuccessRate     float64 `json:"success_rate"`
	TotalChanges    int     `json:"total_changes"`
}

// ResourceMetrics 资源指标
type ResourceMetrics struct {
	TotalResources int                      `json:"total_resources"`
	ByCloud        map[string]int           `json:"by_cloud"`
	ByType         map[string]int           `json:"by_type"`
	ByStatus       map[string]int           `json:"by_status"`
	Distribution   []MultiCloudResourceData `json:"distribution"`
	HealthStatus   []ResourceHealthData     `json:"health_status"`
}

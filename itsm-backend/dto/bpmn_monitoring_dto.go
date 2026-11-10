package dto

import "time"

// BPMN 监控 / 审计 / 指标 / SLA / 租户统计的共享值类型（批次 3 第 2 步）。
//
// 这些结构体原先定义在 service 包的 bpmn_*_service.go 里，字段与 JSON tag
// 原样下沉到此处；service 包保留同名别名（type X = dto.X），因此类型身份、
// 序列化结果与调用方代码都不变，而 handlers/bpmn 的窄契约只需引用本包。

// QueryAuditLogsRequest 查询审计日志请求
type QueryAuditLogsRequest struct {
	ProcessInstanceID    int
	ProcessInstanceKey   string
	ProcessDefinitionKey string
	ActivityID           string
	ActivityType         string
	Action               string
	UserID               int
	AssigneeID           int
	TenantID             int
	StartTime            time.Time
	EndTime              time.Time
	Page                 int
	PageSize             int
	SortBy               string
	SortOrder            string
}

// ProcessMetricsRequest 流程指标请求
type ProcessMetricsRequest struct {
	ProcessDefinitionKey string     `json:"processDefinitionKey"`
	TenantID             int        `json:"tenantId"`
	TimeRange            string     `json:"timeRange"`
	StartTime            *time.Time `json:"startTime"`
	EndTime              *time.Time `json:"endTime"`
}

// ListProcessInstanceStatusQuery 流程实例状态查询请求
type ListProcessInstanceStatusQuery struct {
	TenantID   int
	Page       int
	PageSize   int
	ProcessKey string
	Status     string
	Assignee   string
	StartTime  *time.Time
	EndTime    *time.Time
}

// DashboardMetrics BPMN 仪表盘指标
type DashboardMetrics struct {
	TotalProcesses    int     `json:"totalProcesses"`
	ActiveInstances   int     `json:"activeInstances"`
	CompletedToday    int     `json:"completedToday"`
	OpenTasks         int     `json:"openTasks"`
	SLAComplianceRate float64 `json:"slaComplianceRate"`
	// SLACompliantSamples/SLATotalSamples 暴露合规率样本量：
	// 前端据此区分“暂无样本”与“真实 0%”，不得把 0 样本伪装成高合规。
	SLACompliantSamples int            `json:"slaCompliantSamples"`
	SLATotalSamples     int            `json:"slaTotalSamples"`
	AvgCompletionTime   float64        `json:"avgCompletionTimeMinutes"`
	ProcessHealth       *ProcessHealth `json:"processHealth"`
	TopProcesses        []ProcessStat  `json:"topProcesses"`
	TaskDistribution    []TaskStat     `json:"taskDistribution"`
	TrendData           []TrendPoint   `json:"trendData"`
}

// ProcessHealth 流程健康度分布
type ProcessHealth struct {
	Healthy     int     `json:"healthy"`
	Warning     int     `json:"warning"`
	Critical    int     `json:"critical"`
	HealthScore float64 `json:"healthScore"` // 0-100
}

// ProcessStat 单个流程定义的实例统计
type ProcessStat struct {
	ProcessDefinitionKey string  `json:"processDefinitionKey"`
	TotalInstances       int     `json:"totalInstances"`
	RunningInstances     int     `json:"runningInstances"`
	CompletedInstances   int     `json:"completedInstances"`
	AvgDuration          float64 `json:"avgDurationMinutes"`
}

// TaskStat 任务状态分布
type TaskStat struct {
	Status  string  `json:"status"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// TrendPoint 趋势图上的一个点
type TrendPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// BPMNProcessMetrics 单个流程定义的指标
type BPMNProcessMetrics struct {
	TotalInstances      int     `json:"totalInstances"`
	RunningInstances    int     `json:"runningInstances"`
	CompletedInstances  int     `json:"completedInstances"`
	TerminatedInstances int     `json:"terminatedInstances"`
	CompletionRate      float64 `json:"completionRate"`
	SLAComplianceRate   float64 `json:"slaComplianceRate"`
	AvgCompletionTime   float64 `json:"avgCompletionTimeMinutes"`
}

// BottleneckInfo 瓶颈任务统计
type BottleneckInfo struct {
	TaskName    string  `json:"taskName"`
	TotalCount  int     `json:"totalCount"`
	AvgDuration float64 `json:"avgDurationMinutes"`
	MaxDuration float64 `json:"maxDurationMinutes"`
	MinDuration float64 `json:"minDurationMinutes"`
}

// ProcessMetrics 流程监控指标聚合
type ProcessMetrics struct {
	ProcessDefinitionKey string                  `json:"processDefinitionKey"`
	TotalInstances       int                     `json:"totalInstances"`
	RunningInstances     int                     `json:"runningInstances"`
	CompletedInstances   int                     `json:"completedInstances"`
	FailedInstances      int                     `json:"failedInstances"`
	AverageDuration      time.Duration           `json:"averageDuration"`
	SuccessRate          float64                 `json:"successRate"`
	PerformanceMetrics   *PerformanceMetrics     `json:"performanceMetrics"`
	TaskMetrics          map[string]*TaskMetrics `json:"taskMetrics"`
	BottleneckAnalysis   *BottleneckAnalysis     `json:"bottleneckAnalysis"`
	TenantID             int                     `json:"tenantId"`
	TimeRange            string                  `json:"timeRange"`
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	Throughput          float64 `json:"throughput"`          // 吞吐量（实例/小时）
	AverageResponseTime float64 `json:"averageResponseTime"` // 平均响应时间
	PeakConcurrency     int     `json:"peakConcurrency"`     // 峰值并发数
	ResourceUtilization float64 `json:"resourceUtilization"` // 资源利用率
	ErrorRate           float64 `json:"errorRate"`           // 错误率
	Availability        float64 `json:"availability"`        // 可用性
	LatencyPercentile95 float64 `json:"latencyPercentile95"` // 95%延迟
	LatencyPercentile99 float64 `json:"latencyPercentile99"` // 99%延迟
}

// TaskMetrics 任务级指标
type TaskMetrics struct {
	TaskID              string             `json:"taskId"`
	TaskName            string             `json:"taskName"`
	TotalExecutions     int                `json:"totalExecutions"`
	CompletedExecutions int                `json:"completedExecutions"`
	FailedExecutions    int                `json:"failedExecutions"`
	AverageDuration     time.Duration      `json:"averageDuration"`
	SuccessRate         float64            `json:"successRate"`
	AssigneePerformance map[string]float64 `json:"assigneePerformance"`
	WaitTime            time.Duration      `json:"waitTime"`       // 等待时间
	ProcessingTime      time.Duration      `json:"processingTime"` // 处理时间
	QueueLength         int                `json:"queueLength"`    // 队列长度
	Priority            string             `json:"priority"`       // 优先级
}

// BottleneckAnalysis 瓶颈分析聚合
type BottleneckAnalysis struct {
	BottleneckTasks     []*BottleneckTask     `json:"bottleneckTasks"`
	SlowestPaths        []*SlowestPath        `json:"slowestPaths"`
	ResourceConstraints []*ResourceConstraint `json:"resourceConstraints"`
	Recommendations     []string              `json:"recommendations"`
	Severity            string                `json:"severity"` // low, medium, high, critical
}

// BottleneckTask 瓶颈任务明细
type BottleneckTask struct {
	TaskID                   string        `json:"taskId"`
	TaskName                 string        `json:"taskName"`
	BottleneckType           string        `json:"bottleneckType"` // processing, waiting, resource
	ImpactScore              float64       `json:"impactScore"`    // 0-100
	AverageDuration          time.Duration `json:"averageDuration"`
	WaitTime                 time.Duration `json:"waitTime"`
	QueueLength              int           `json:"queueLength"`
	Assignee                 string        `json:"assignee"`
	Recommendation           string        `json:"recommendation"`
	WaitTimeSeconds          int           `json:"waitTimeSeconds"`
	ProcessingTimeSeconds    int           `json:"processingTimeSeconds"`
	TotalDurationSeconds     int           `json:"totalDurationSeconds"`
	P95WaitTimeSeconds       int           `json:"p95WaitTimeSeconds"`
	P95ProcessingTimeSeconds int           `json:"p95ProcessingTimeSeconds"`
	P95TotalDurationSeconds  int           `json:"p95TotalDurationSeconds"`
	SampleCount              int           `json:"sampleCount"`
}

// SlowestPath 最慢执行路径
type SlowestPath struct {
	PathID               string   `json:"pathId"`
	PathName             string   `json:"pathName"`
	TotalDurationSeconds int64    `json:"totalDurationSeconds"`
	TaskCount            int      `json:"taskCount"`
	BottleneckTasks      []string `json:"bottleneckTasks"`
	Optimization         string   `json:"optimization"`
}

// ResourceConstraint 资源约束
type ResourceConstraint struct {
	ResourceType   string  `json:"resourceType"` // human, system, external
	ResourceName   string  `json:"resourceName"`
	Utilization    float64 `json:"utilization"` // 0-100
	Capacity       int     `json:"capacity"`
	CurrentLoad    int     `json:"currentLoad"`
	ConstraintType string  `json:"constraintType"` // capacity, skill, availability
	Impact         string  `json:"impact"`         // low, medium, high
}

// ProcessInstanceStatus 流程实例状态
type ProcessInstanceStatus struct {
	ProcessInstanceID string                 `json:"processInstanceId"`
	Status            string                 `json:"status"`
	CurrentTask       string                 `json:"currentTask"`
	Assignee          string                 `json:"assignee"`
	StartTime         time.Time              `json:"startTime"`
	ExpectedEndTime   *time.Time             `json:"expectedEndTime"`
	Variables         map[string]interface{} `json:"variables"`
	Progress          float64                `json:"progress"`
	TenantID          int                    `json:"tenantId"`
	EstimatedDuration time.Duration          `json:"estimatedDuration"`
	RiskLevel         string                 `json:"riskLevel"` // low, medium, high
}

// ProcessTimelineEntry 流程执行时间线条目
type ProcessTimelineEntry struct {
	ID              string                 `json:"id"`
	Sequence        int                    `json:"sequence"`
	Timestamp       time.Time              `json:"timestamp"`
	EventType       string                 `json:"eventType"`
	ActivityID      string                 `json:"activityId"`
	ActivityName    string                 `json:"activityName"`
	ActivityType    string                 `json:"activityType"`
	UserID          int                    `json:"userId"`
	UserName        string                 `json:"userName"`
	AssigneeID      int                    `json:"assigneeId"`
	AssigneeName    string                 `json:"assigneeName"`
	TenantID        int                    `json:"tenantId"`
	VariablesBefore map[string]interface{} `json:"variablesBefore,omitempty"`
	VariablesAfter  map[string]interface{} `json:"variablesAfter,omitempty"`
	Comment         string                 `json:"comment,omitempty"`
	DurationMs      int                    `json:"durationMs"`
	NodeDurationMs  int                    `json:"nodeDurationMs"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// SLAViolation SLA 违规记录
type SLAViolation struct {
	ResourceType   string
	ResourceID     int
	ResourceKey    string
	SLAStatus      string
	StartTime      time.Time
	Deadline       time.Time
	ElapsedMinutes int
	TenantID       int
}

// TenantBPMNStats 租户维度 BPMN 统计
type TenantBPMNStats struct {
	TotalDefinitions   int `json:"totalDefinitions"`
	TotalInstances     int `json:"totalInstances"`
	RunningInstances   int `json:"runningInstances"`
	CompletedInstances int `json:"completedInstances"`
	TotalTasks         int `json:"totalTasks"`
	OpenTasks          int `json:"openTasks"`
}

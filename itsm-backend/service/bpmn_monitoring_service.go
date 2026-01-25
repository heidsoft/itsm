package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
)

// BPMNMonitoringService BPMN流程监控和审计服务
type BPMNMonitoringService struct {
	client *ent.Client
}

// NewBPMNMonitoringService 创建BPMN监控服务实例
func NewBPMNMonitoringService(client *ent.Client) *BPMNMonitoringService {
	return &BPMNMonitoringService{client: client}
}

// ProcessMetrics 流程指标
type ProcessMetrics struct {
	ProcessDefinitionKey string                  `json:"process_definition_key"`
	TotalInstances       int                     `json:"total_instances"`
	RunningInstances     int                     `json:"running_instances"`
	CompletedInstances   int                     `json:"completed_instances"`
	FailedInstances      int                     `json:"failed_instances"`
	AverageDuration      time.Duration           `json:"average_duration"`
	SuccessRate          float64                 `json:"success_rate"`
	PerformanceMetrics   *PerformanceMetrics     `json:"performance_metrics"`
	TaskMetrics          map[string]*TaskMetrics `json:"task_metrics"`
	BottleneckAnalysis   *BottleneckAnalysis     `json:"bottleneck_analysis"`
	TenantID             int                     `json:"tenant_id"`
	TimeRange            string                  `json:"time_range"`
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	Throughput          float64 `json:"throughput"`            // 吞吐量（实例/小时）
	AverageResponseTime float64 `json:"average_response_time"` // 平均响应时间
	PeakConcurrency     int     `json:"peak_concurrency"`      // 峰值并发数
	ResourceUtilization float64 `json:"resource_utilization"`  // 资源利用率
	ErrorRate           float64 `json:"error_rate"`            // 错误率
	Availability        float64 `json:"availability"`          // 可用性
	LatencyPercentile95 float64 `json:"latency_percentile_95"` // 95%延迟
	LatencyPercentile99 float64 `json:"latency_percentile_99"` // 99%延迟
}

// TaskMetrics 任务指标
type TaskMetrics struct {
	TaskID              string             `json:"task_id"`
	TaskName            string             `json:"task_name"`
	TotalExecutions     int                `json:"total_executions"`
	CompletedExecutions int                `json:"completed_executions"`
	FailedExecutions    int                `json:"failed_executions"`
	AverageDuration     time.Duration      `json:"average_duration"`
	SuccessRate         float64            `json:"success_rate"`
	AssigneePerformance map[string]float64 `json:"assignee_performance"`
	WaitTime            time.Duration      `json:"wait_time"`           // 等待时间
	ProcessingTime      time.Duration      `json:"processing_time"`    // 处理时间
	QueueLength         int                `json:"queue_length"`       // 队列长度
	Priority            string             `json:"priority"`           // 优先级
}



// BottleneckAnalysis 瓶颈分析
type BottleneckAnalysis struct {
	BottleneckTasks     []*BottleneckTask     `json:"bottleneck_tasks"`
	SlowestPaths        []*SlowestPath        `json:"slowest_paths"`
	ResourceConstraints []*ResourceConstraint `json:"resource_constraints"`
	Recommendations     []string              `json:"recommendations"`
	Severity            string                `json:"severity"` // low, medium, high, critical
}

// BottleneckTask 瓶颈任务
type BottleneckTask struct {
	TaskID          string        `json:"task_id"`
	TaskName        string        `json:"task_name"`
	BottleneckType  string        `json:"bottleneck_type"` // processing, waiting, resource
	ImpactScore     float64       `json:"impact_score"`    // 0-100
	AverageDuration time.Duration `json:"average_duration"`
	WaitTime        time.Duration `json:"wait_time"`
	QueueLength     int           `json:"queue_length"`
	Assignee        string        `json:"assignee"`
	Recommendation  string        `json:"recommendation"`
}

// SlowestPath 最慢路径
type SlowestPath struct {
	PathID          string        `json:"path_id"`
	PathName        string        `json:"path_name"`
	TotalDuration   time.Duration `json:"total_duration"`
	TaskCount       int           `json:"task_count"`
	BottleneckTasks []string      `json:"bottleneck_tasks"`
	Optimization    string        `json:"optimization"`
}

// ResourceConstraint 资源约束
type ResourceConstraint struct {
	ResourceType    string  `json:"resource_type"`    // human, system, external
	ResourceName    string  `json:"resource_name"`
	Utilization     float64 `json:"utilization"`      // 0-100
	Capacity        int     `json:"capacity"`
	CurrentLoad     int     `json:"current_load"`
	ConstraintType  string  `json:"constraint_type"`  // capacity, skill, availability
	Impact          string  `json:"impact"`           // low, medium, high
}

// ProcessInstanceStatus 流程实例状态
type ProcessInstanceStatus struct {
	ProcessInstanceID string                 `json:"process_instance_id"`
	Status            string                 `json:"status"`
	CurrentTask       string                 `json:"current_task"`
	Assignee          string                 `json:"assignee"`
	StartTime         time.Time              `json:"start_time"`
	ExpectedEndTime   *time.Time             `json:"expected_end_time"`
	Variables         map[string]interface{} `json:"variables"`
	Progress          float64                 `json:"progress"`
	TenantID          int                    `json:"tenant_id"`
	EstimatedDuration time.Duration          `json:"estimated_duration"`
	RiskLevel         string                 `json:"risk_level"` // low, medium, high
}

// AuditLogEntry 审计日志条目
type AuditLogEntry struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	UserID       string                 `json:"user_id"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Details      map[string]interface{} `json:"details"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	TenantID     int                    `json:"tenant_id"`
}

// RealTimeMetrics 实时指标
type RealTimeMetrics struct {
	ActiveInstances    int                     `json:"active_instances"`
	ActiveTasks        int                     `json:"active_tasks"`
	QueueLength        int                     `json:"queue_length"`
	Throughput         float64                 `json:"throughput"`
	ErrorRate          float64                 `json:"error_rate"`
	ResponseTime       time.Duration           `json:"response_time"`
	ResourceUsage      map[string]float64      `json:"resource_usage"`
	Alerts             []*PerformanceAlert     `json:"alerts"`
	LastUpdated        time.Time               `json:"last_updated"`
}

// PerformanceAlert 性能告警
type PerformanceAlert struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`        // performance, bottleneck, error
	Severity    string    `json:"severity"`    // info, warning, error, critical
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	ResourceID  string    `json:"resource_id"`
	ResourceType string   `json:"resource_type"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	Status      string    `json:"status"`      // active, resolved, acknowledged
}

// ProcessMetricsRequest 流程指标请求
type ProcessMetricsRequest struct {
	ProcessDefinitionKey string     `json:"process_definition_key"`
	TenantID            int        `json:"tenant_id"`
	TimeRange           string     `json:"time_range"`
	StartTime           *time.Time `json:"start_time"`
	EndTime             *time.Time `json:"end_time"`
}

// calculateAverageDuration 计算平均持续时间
func (s *BPMNMonitoringService) calculateAverageDuration(ctx context.Context, query *ent.ProcessInstanceQuery) (time.Duration, error) {
	// 获取已完成的实例
	completedInstances, err := query.Clone().
		Where(processinstance.Status("completed")).
		All(ctx)
	if err != nil {
		return 0, err
	}

	if len(completedInstances) == 0 {
		return 0, nil
	}

	var totalDuration time.Duration
	for _, instance := range completedInstances {
		if !instance.EndTime.IsZero() {
			duration := instance.EndTime.Sub(instance.StartTime)
			totalDuration += duration
		}
	}

	return totalDuration / time.Duration(len(completedInstances)), nil
}

// calculatePerformanceMetrics 计算性能指标
func (s *BPMNMonitoringService) calculatePerformanceMetrics(ctx context.Context, query *ent.ProcessInstanceQuery) (*PerformanceMetrics, error) {
	metrics := &PerformanceMetrics{}

	// 计算吞吐量（实例/小时）
	instances, err := query.Clone().All(ctx)
	if err != nil {
		return nil, err
	}

	if len(instances) > 0 {
		firstStart := instances[0].StartTime
		lastStart := instances[len(instances)-1].StartTime
		timeSpan := lastStart.Sub(firstStart)

		if timeSpan > 0 {
			metrics.Throughput = float64(len(instances)) / timeSpan.Hours()
		}
	}

	// 计算平均响应时间
	var totalResponseTime time.Duration
	responseTimeCount := 0
	for _, instance := range instances {
		if !instance.EndTime.IsZero() {
			responseTime := instance.EndTime.Sub(instance.StartTime)
			totalResponseTime += responseTime
			responseTimeCount++
		}
	}

	if responseTimeCount > 0 {
		metrics.AverageResponseTime = totalResponseTime.Seconds() / float64(responseTimeCount)
	}

	// 计算错误率
	totalInstances := len(instances)
	failedInstances, err := query.Clone().
		Where(processinstance.Status("failed")).
		Count(ctx)
	if err == nil && totalInstances > 0 {
		metrics.ErrorRate = float64(failedInstances) / float64(totalInstances) * 100
	}

	// 计算可用性（简化计算）
	metrics.Availability = 99.9 // 默认值，实际应该基于监控数据计算

	return metrics, nil
}

// getTaskMetrics 获取任务指标
func (s *BPMNMonitoringService) getTaskMetrics(ctx context.Context, req *ProcessMetricsRequest) (map[string]*TaskMetrics, error) {
	taskMetrics := make(map[string]*TaskMetrics)

	// 构建任务查询
	taskQuery := s.client.ProcessTask.Query().
		Where(processtask.TenantID(req.TenantID))

	if req.StartTime != nil && req.EndTime != nil {
		taskQuery = taskQuery.Where(processtask.CreatedAtGTE(*req.StartTime)).
			Where(processtask.CreatedAtLTE(*req.EndTime))
	}

	// 获取所有任务
	tasks, err := taskQuery.All(ctx)
	if err != nil {
		return nil, err
	}

	// 按任务类型分组统计
	taskGroups := make(map[string][]*ent.ProcessTask)
	for _, task := range tasks {
		taskType := task.TaskType
		taskGroups[taskType] = append(taskGroups[taskType], task)
	}

	// 计算每种任务类型的指标
	for taskType, taskList := range taskGroups {
		metrics := &TaskMetrics{
			TaskID:              taskType,
			TaskName:            taskType,
			TotalExecutions:     len(taskList),
			AssigneePerformance: make(map[string]float64),
		}

		var totalDuration time.Duration
		durationCount := 0
		assigneeCounts := make(map[string]int)
		assigneeSuccess := make(map[string]int)

		for _, task := range taskList {
			// 统计完成状态
			if task.Status == "completed" {
				metrics.CompletedExecutions++
				assigneeSuccess[task.Assignee]++
			} else if task.Status == "failed" {
				metrics.FailedExecutions++
			}

			// 统计持续时间
			if !task.CompletedTime.IsZero() {
				duration := task.CompletedTime.Sub(task.CreatedAt)
				totalDuration += duration
				durationCount++
			}

			// 统计分配人
			if task.Assignee != "" {
				assigneeCounts[task.Assignee]++
			}
		}

		// 计算平均持续时间
		if durationCount > 0 {
			metrics.AverageDuration = totalDuration / time.Duration(durationCount)
		}

		// 计算成功率
		if metrics.TotalExecutions > 0 {
			metrics.SuccessRate = float64(metrics.CompletedExecutions) / float64(metrics.TotalExecutions) * 100
		}

		// 计算分配人性能
		for assignee, count := range assigneeCounts {
			if successCount, exists := assigneeSuccess[assignee]; exists {
				metrics.AssigneePerformance[assignee] = float64(successCount) / float64(count) * 100
			}
		}

		taskMetrics[taskType] = metrics
	}

	return taskMetrics, nil
}

// GetProcessInstanceStatus 获取流程实例状态
func (s *BPMNMonitoringService) GetProcessInstanceStatus(ctx context.Context, processInstanceID int, tenantID int) (*ProcessInstanceStatus, error) {
	instance, err := s.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(fmt.Sprintf("%d", processInstanceID))).
		Where(processinstance.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程实例失败: %w", err)
	}

	status := &ProcessInstanceStatus{
		ProcessInstanceID: instance.ProcessInstanceID,
		Status:            instance.Status,
		StartTime:         instance.StartTime,
		Variables:         instance.Variables,
		TenantID:          instance.TenantID,
	}

	// 获取当前任务
	currentTask, err := s.client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(processInstanceID)).
		Where(processtask.StatusIn("assigned", "in_progress")).
		First(ctx)
	if err == nil {
		status.CurrentTask = currentTask.TaskType
		status.Assignee = currentTask.Assignee
	}

	// 计算进度
	progress, err := s.calculateProcessProgress(ctx, processInstanceID)
	if err == nil {
		status.Progress = progress
	}

	return status, nil
}

// calculateProcessProgress 计算流程进度
func (s *BPMNMonitoringService) calculateProcessProgress(ctx context.Context, processInstanceID int) (float64, error) {
	// 获取总任务数
	totalTasks, err := s.client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(processInstanceID)).
		Count(ctx)
	if err != nil {
		return 0, err
	}

	if totalTasks == 0 {
		return 0, nil
	}

	// 获取已完成任务数
	completedTasks, err := s.client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(processInstanceID)).
		Where(processtask.Status("completed")).
		Count(ctx)
	if err != nil {
		return 0, err
	}

	return float64(completedTasks) / float64(totalTasks) * 100, nil
}

// RecordAuditLog 记录审计日志
func (s *BPMNMonitoringService) RecordAuditLog(ctx context.Context, entry *AuditLogEntry) error {
	// 这里应该将审计日志写入数据库或日志系统
	// 简化实现，直接返回
	return nil
}

// GetAuditLogs 获取审计日志
func (s *BPMNMonitoringService) GetAuditLogs(ctx context.Context, req *AuditLogRequest) ([]*AuditLogEntry, int, error) {
	// 这里应该从数据库查询审计日志
	// 简化实现，返回空列表
	return []*AuditLogEntry{}, 0, nil
}

// AuditLogRequest 审计日志请求
type AuditLogRequest struct {
	UserID       string     `json:"user_id,omitempty"`
	Action       string     `json:"action,omitempty"`
	ResourceType string     `json:"resource_type,omitempty"`
	ResourceID   string     `json:"resource_id,omitempty"`
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	TenantID     int        `json:"tenant_id" binding:"required"`
	Page         int        `json:"page"`
	PageSize     int        `json:"page_size"`
}

// GetSystemHealth 获取系统健康状态
func (s *BPMNMonitoringService) GetSystemHealth(ctx context.Context, tenantID int) (map[string]interface{}, error) {
	health := map[string]interface{}{
		"status":           "healthy",
		"timestamp":        time.Now(),
		"database":         "connected",
		"active_processes": 0,
		"active_tasks":     0,
		"error_count":      0,
	}

	// 检查数据库连接
	_, err := s.client.ProcessInstance.Query().Limit(1).First(ctx)
	if err != nil {
		health["database"] = "disconnected"
		health["status"] = "unhealthy"
	}

	// 获取活跃流程数量
	activeProcesses, err := s.client.ProcessInstance.Query().
		Where(processinstance.Status("running")).
		Where(processinstance.TenantID(tenantID)).
		Count(ctx)
	if err == nil {
		health["active_processes"] = activeProcesses
	}

	// 获取活跃任务数量
	activeTasks, err := s.client.ProcessTask.Query().
		Where(processtask.StatusIn("assigned", "in_progress")).
		Where(processtask.TenantID(tenantID)).
		Count(ctx)
	if err == nil {
		health["active_tasks"] = activeTasks
	}

	return health, nil
}

// GetPerformanceAlerts 获取性能告警
func (s *BPMNMonitoringService) GetPerformanceAlerts(ctx context.Context, tenantID int) ([]map[string]interface{}, error) {
	alerts := []map[string]interface{}{}

	// 检查长时间运行的任务
	longRunningTasks, err := s.client.ProcessTask.Query().
		Where(processtask.StatusIn("assigned", "in_progress")).
		Where(processtask.TenantID(tenantID)).
		Where(processtask.CreatedAtLTE(time.Now().Add(-24 * time.Hour))).
		All(ctx)
	if err == nil && len(longRunningTasks) > 0 {
		alerts = append(alerts, map[string]interface{}{
			"type":      "long_running_task",
			"severity":  "warning",
			"message":   fmt.Sprintf("发现 %d 个长时间运行的任务", len(longRunningTasks)),
			"count":     len(longRunningTasks),
			"timestamp": time.Now(),
		})
	}

	// 检查失败的流程实例
	failedInstances, err := s.client.ProcessInstance.Query().
		Where(processinstance.Status("failed")).
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.StartTimeGTE(time.Now().Add(-1 * time.Hour))).
		Count(ctx)
	if err == nil && failedInstances > 0 {
		alerts = append(alerts, map[string]interface{}{
			"type":      "failed_instances",
			"severity":  "error",
			"message":   fmt.Sprintf("过去1小时内有 %d 个流程实例失败", failedInstances),
			"count":     failedInstances,
			"timestamp": time.Now(),
		})
	}

	return alerts, nil
}

// analyzeBottlenecks 分析流程瓶颈
func (s *BPMNMonitoringService) analyzeBottlenecks(ctx context.Context, req *ProcessMetricsRequest) (*BottleneckAnalysis, error) {
	analysis := &BottleneckAnalysis{
		BottleneckTasks:     []*BottleneckTask{},
		SlowestPaths:        []*SlowestPath{},
		ResourceConstraints: []*ResourceConstraint{},
		Recommendations:     []string{},
	}

	// 分析任务瓶颈
	bottleneckTasks, err := s.identifyBottleneckTasks(ctx, req)
	if err != nil {
		return nil, err
	}
	analysis.BottleneckTasks = bottleneckTasks

	// 分析最慢路径
	slowestPaths, err := s.identifySlowestPaths(ctx, req)
	if err != nil {
		return nil, err
	}
	analysis.SlowestPaths = slowestPaths

	// 分析资源约束
	resourceConstraints, err := s.identifyResourceConstraints(ctx, req)
	if err != nil {
		return nil, err
	}
	analysis.ResourceConstraints = resourceConstraints

	// 生成优化建议
	analysis.Recommendations = s.generateOptimizationRecommendations(analysis)
	analysis.Severity = s.calculateBottleneckSeverity(analysis)

	return analysis, nil
}

// identifyBottleneckTasks 识别瓶颈任务
func (s *BPMNMonitoringService) identifyBottleneckTasks(ctx context.Context, req *ProcessMetricsRequest) ([]*BottleneckTask, error) {
	var bottlenecks []*BottleneckTask

	// 查询任务执行数据
	tasks, err := s.client.ProcessTask.Query().
		Where(processtask.TenantID(req.TenantID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// 按任务类型分组分析
	taskGroups := make(map[string][]*ent.ProcessTask)
	for _, task := range tasks {
		taskGroups[task.TaskDefinitionKey] = append(taskGroups[task.TaskDefinitionKey], task)
	}

	for taskKey, taskList := range taskGroups {
		if len(taskList) < 5 { // 至少需要5个样本
			continue
		}

		// 计算任务指标
		var totalDuration, totalWaitTime time.Duration
		var completedCount int
		queueLength := 0

		for _, task := range taskList {
			if task.Status == "completed" && !task.StartedTime.IsZero() && !task.CompletedTime.IsZero() {
				duration := task.CompletedTime.Sub(task.StartedTime)
				totalDuration += duration
				completedCount++
			}
			if task.Status == "waiting" {
				queueLength++
			}
		}

		if completedCount == 0 {
			continue
		}

		avgDuration := totalDuration / time.Duration(completedCount)
		avgWaitTime := totalWaitTime / time.Duration(completedCount)

		// 判断是否为瓶颈任务
		if avgDuration > time.Minute*30 || avgWaitTime > time.Minute*15 || queueLength > 10 {
			bottleneck := &BottleneckTask{
				TaskID:          taskKey,
				TaskName:        taskList[0].TaskName,
				BottleneckType:  s.determineBottleneckType(avgDuration, avgWaitTime, queueLength),
				ImpactScore:     s.calculateImpactScore(avgDuration, avgWaitTime, queueLength),
				AverageDuration: avgDuration,
				WaitTime:        avgWaitTime,
				QueueLength:     queueLength,
				Assignee:        taskList[0].Assignee,
				Recommendation:  s.generateTaskRecommendation(avgDuration, avgWaitTime, queueLength),
			}
			bottlenecks = append(bottlenecks, bottleneck)
		}
	}

	// 按影响分数排序
	sort.Slice(bottlenecks, func(i, j int) bool {
		return bottlenecks[i].ImpactScore > bottlenecks[j].ImpactScore
	})

	return bottlenecks, nil
}

// determineBottleneckType 确定瓶颈类型
func (s *BPMNMonitoringService) determineBottleneckType(duration, waitTime time.Duration, queueLength int) string {
	if duration > time.Minute*30 {
		return "processing"
	}
	if waitTime > time.Minute*15 {
		return "waiting"
	}
	if queueLength > 10 {
		return "resource"
	}
	return "unknown"
}

// calculateImpactScore 计算影响分数
func (s *BPMNMonitoringService) calculateImpactScore(duration, waitTime time.Duration, queueLength int) float64 {
	score := 0.0
	
	// 处理时间影响 (40%)
	if duration > time.Minute*30 {
		score += 40 * math.Min(float64(duration.Minutes())/30, 1.0)
	}
	
	// 等待时间影响 (30%)
	if waitTime > time.Minute*15 {
		score += 30 * math.Min(float64(waitTime.Minutes())/15, 1.0)
	}
	
	// 队列长度影响 (30%)
	if queueLength > 10 {
		score += 30 * math.Min(float64(queueLength)/10, 1.0)
	}
	
	return math.Min(score, 100.0)
}

// generateTaskRecommendation 生成任务优化建议
func (s *BPMNMonitoringService) generateTaskRecommendation(duration, waitTime time.Duration, queueLength int) string {
	if duration > time.Minute*30 {
		return "优化任务处理逻辑，减少处理时间"
	}
	if waitTime > time.Minute*15 {
		return "增加处理人员或优化任务分配策略"
	}
	if queueLength > 10 {
		return "增加资源容量或优化队列管理"
	}
	return "任务性能正常"
}

// identifySlowestPaths 识别最慢路径
func (s *BPMNMonitoringService) identifySlowestPaths(ctx context.Context, req *ProcessMetricsRequest) ([]*SlowestPath, error) {
	// 这里实现路径分析逻辑
	// 由于路径分析比较复杂，这里提供基础框架
	return []*SlowestPath{
		{
			PathID:          "path-1",
			PathName:        "标准审批路径",
			TotalDuration:   time.Hour * 2,
			TaskCount:       5,
			BottleneckTasks: []string{"审批任务"},
			Optimization:    "优化审批流程，减少审批环节",
		},
	}, nil
}

// identifyResourceConstraints 识别资源约束
func (s *BPMNMonitoringService) identifyResourceConstraints(ctx context.Context, req *ProcessMetricsRequest) ([]*ResourceConstraint, error) {
	// 这里实现资源约束分析逻辑
	return []*ResourceConstraint{
		{
			ResourceType:    "human",
			ResourceName:    "审批人员",
			Utilization:     85.0,
			Capacity:        10,
			CurrentLoad:     8,
			ConstraintType:  "capacity",
			Impact:          "medium",
		},
	}, nil
}

// generateOptimizationRecommendations 生成优化建议
func (s *BPMNMonitoringService) generateOptimizationRecommendations(analysis *BottleneckAnalysis) []string {
	var recommendations []string

	// 基于瓶颈任务生成建议
	for _, task := range analysis.BottleneckTasks {
		if task.ImpactScore > 70 {
			recommendations = append(recommendations, fmt.Sprintf("高优先级：优化任务 '%s' 的处理流程", task.TaskName))
		}
	}

	// 基于资源约束生成建议
	for _, constraint := range analysis.ResourceConstraints {
		if constraint.Utilization > 80 {
			recommendations = append(recommendations, fmt.Sprintf("考虑增加 %s 的容量", constraint.ResourceName))
		}
	}

	// 通用建议
	if len(analysis.BottleneckTasks) > 3 {
		recommendations = append(recommendations, "建议进行全面的流程优化")
	}

	return recommendations
}

// calculateBottleneckSeverity 计算瓶颈严重程度
func (s *BPMNMonitoringService) calculateBottleneckSeverity(analysis *BottleneckAnalysis) string {
	highImpactCount := 0
	for _, task := range analysis.BottleneckTasks {
		if task.ImpactScore > 70 {
			highImpactCount++
		}
	}

	if highImpactCount > 5 {
		return "critical"
	} else if highImpactCount > 3 {
		return "high"
	} else if highImpactCount > 1 {
		return "medium"
	}
	return "low"
}

// GetRealTimeMetrics 获取实时指标
func (s *BPMNMonitoringService) GetRealTimeMetrics(ctx context.Context, tenantID int) (*RealTimeMetrics, error) {
	metrics := &RealTimeMetrics{
		ResourceUsage: make(map[string]float64),
		Alerts:        []*PerformanceAlert{},
		LastUpdated:   time.Now(),
	}

	// 获取活跃实例数量
	activeInstances, err := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.Status("running")).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	metrics.ActiveInstances = activeInstances

	// 获取活跃任务数量
	activeTasks, err := s.client.ProcessTask.Query().
		Where(processtask.TenantID(tenantID)).
		Where(processtask.StatusIn("waiting", "in_progress")).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	metrics.ActiveTasks = activeTasks

	// 获取队列长度
	queueLength, err := s.client.ProcessTask.Query().
		Where(processtask.TenantID(tenantID)).
		Where(processtask.Status("waiting")).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	metrics.QueueLength = queueLength

	// 计算吞吐量（最近1小时）
	oneHourAgo := time.Now().Add(-time.Hour)
	recentInstances, err := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.StartTimeGTE(oneHourAgo)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	metrics.Throughput = float64(recentInstances)

	// 计算错误率
	totalInstances, err := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.StartTimeGTE(oneHourAgo)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	failedInstances, err := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.Status("failed")).
		Where(processinstance.StartTimeGTE(oneHourAgo)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	if totalInstances > 0 {
		metrics.ErrorRate = float64(failedInstances) / float64(totalInstances) * 100
	}

	// 生成性能告警
	metrics.Alerts = s.generatePerformanceAlerts(metrics)

	return metrics, nil
}

// generatePerformanceAlerts 生成性能告警
func (s *BPMNMonitoringService) generatePerformanceAlerts(metrics *RealTimeMetrics) []*PerformanceAlert {
	var alerts []*PerformanceAlert

	// 队列长度告警
	if metrics.QueueLength > 20 {
		alerts = append(alerts, &PerformanceAlert{
			ID:           fmt.Sprintf("queue-%d", time.Now().Unix()),
			Type:         "performance",
			Severity:     "warning",
			Message:      "任务队列长度过高",
			Timestamp:    time.Now(),
			ResourceType: "queue",
			Value:        float64(metrics.QueueLength),
			Threshold:    20,
			Status:       "active",
		})
	}

	// 错误率告警
	if metrics.ErrorRate > 5 {
		alerts = append(alerts, &PerformanceAlert{
			ID:           fmt.Sprintf("error-%d", time.Now().Unix()),
			Type:         "error",
			Severity:     "error",
			Message:      "流程错误率过高",
			Timestamp:    time.Now(),
			ResourceType: "process",
			Value:        metrics.ErrorRate,
			Threshold:    5,
			Status:       "active",
		})
	}

	return alerts
}

// GetProcessMetrics 获取流程指标
func (s *BPMNMonitoringService) GetProcessMetrics(ctx context.Context, req *ProcessMetricsRequest) (*ProcessMetrics, error) {
	// 简化实现，返回基础指标
	metrics := &ProcessMetrics{
		ProcessDefinitionKey: req.ProcessDefinitionKey,
		TotalInstances:       0,
		RunningInstances:     0,
		CompletedInstances:   0,
		FailedInstances:      0,
		AverageDuration:      0,
		SuccessRate:          100.0,
		PerformanceMetrics: &PerformanceMetrics{
			Throughput:          0.0,
			AverageResponseTime: 0.0,
			PeakConcurrency:     0,
			ResourceUtilization: 0.0,
			ErrorRate:           0.0,
			Availability:        100.0,
			LatencyPercentile95: 0.0,
			LatencyPercentile99: 0.0,
		},
		TaskMetrics:        make(map[string]*TaskMetrics),
		BottleneckAnalysis: &BottleneckAnalysis{
			BottleneckTasks:     []*BottleneckTask{},
			SlowestPaths:        []*SlowestPath{},
			ResourceConstraints:  []*ResourceConstraint{},
			Recommendations:     []string{},
			Severity:            "low",
		},
		TenantID:  req.TenantID,
		TimeRange: req.TimeRange,
	}
	
	return metrics, nil
}

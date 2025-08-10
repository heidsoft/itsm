package service

import (
	"context"
	"fmt"
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
	Progress          float64                `json:"progress"`
	TenantID          int                    `json:"tenant_id"`
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

// GetProcessMetrics 获取流程指标
func (s *BPMNMonitoringService) GetProcessMetrics(ctx context.Context, req *ProcessMetricsRequest) (*ProcessMetrics, error) {
	metrics := &ProcessMetrics{
		ProcessDefinitionKey: req.ProcessDefinitionKey,
		TenantID:             req.TenantID,
		TimeRange:            req.TimeRange,
		TaskMetrics:          make(map[string]*TaskMetrics),
	}

	// 构建查询条件
	query := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(req.TenantID))

	if req.ProcessDefinitionKey != "" {
		query = query.Where(processinstance.ProcessDefinitionKey(req.ProcessDefinitionKey))
	}

	// 添加时间范围过滤
	if req.StartTime != nil && req.EndTime != nil {
		query = query.Where(processinstance.StartTimeGTE(*req.StartTime)).
			Where(processinstance.StartTimeLTE(*req.EndTime))
	}

	// 获取实例总数
	totalInstances, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取实例总数失败: %w", err)
	}
	metrics.TotalInstances = totalInstances

	// 获取各状态实例数量
	runningInstances, err := query.Clone().
		Where(processinstance.Status("running")).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取运行中实例数量失败: %w", err)
	}
	metrics.RunningInstances = runningInstances

	completedInstances, err := query.Clone().
		Where(processinstance.Status("completed")).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取已完成实例数量失败: %w", err)
	}
	metrics.CompletedInstances = completedInstances

	failedInstances, err := query.Clone().
		Where(processinstance.Status("failed")).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取失败实例数量失败: %w", err)
	}
	metrics.FailedInstances = failedInstances

	// 计算成功率
	if totalInstances > 0 {
		metrics.SuccessRate = float64(completedInstances) / float64(totalInstances) * 100
	}

	// 计算平均持续时间
	avgDuration, err := s.calculateAverageDuration(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("计算平均持续时间失败: %w", err)
	}
	metrics.AverageDuration = avgDuration

	// 获取性能指标
	performanceMetrics, err := s.calculatePerformanceMetrics(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("计算性能指标失败: %w", err)
	}
	metrics.PerformanceMetrics = performanceMetrics

	// 获取任务指标
	taskMetrics, err := s.getTaskMetrics(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("获取任务指标失败: %w", err)
	}
	metrics.TaskMetrics = taskMetrics

	return metrics, nil
}

// ProcessMetricsRequest 流程指标请求
type ProcessMetricsRequest struct {
	ProcessDefinitionKey string     `json:"process_definition_key,omitempty"`
	StartTime            *time.Time `json:"start_time,omitempty"`
	EndTime              *time.Time `json:"end_time,omitempty"`
	TenantID             int        `json:"tenant_id" binding:"required"`
	TimeRange            string     `json:"time_range,omitempty"`
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
func (s *BPMNMonitoringService) GetProcessInstanceStatus(ctx context.Context, processInstanceID string, tenantID int) (*ProcessInstanceStatus, error) {
	instance, err := s.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(processInstanceID)).
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
func (s *BPMNMonitoringService) calculateProcessProgress(ctx context.Context, processInstanceID string) (float64, error) {
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

package service

import (
	"context"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"

	"go.uber.org/zap"
)

// BPMNMetricsService BPMN指标服务
type BPMNMetricsService struct {
	client     *ent.Client
	logger     *zap.SugaredLogger
	slaService *BPMNSLAService
}

// NewBPMNMetricsService 创建BPMN指标服务
func NewBPMNMetricsService(client *ent.Client, logger *zap.SugaredLogger) *BPMNMetricsService {
	return &BPMNMetricsService{
		client:     client,
		logger:     logger,
		slaService: NewBPMNSLAService(client, logger),
	}
}

// DashboardMetrics 仪表盘指标
type DashboardMetrics struct {
	TotalProcesses    int            `json:"total_processes"`
	ActiveInstances   int            `json:"active_instances"`
	CompletedToday    int            `json:"completed_today"`
	OpenTasks         int            `json:"open_tasks"`
	SLAComplianceRate float64        `json:"sla_compliance_rate"`
	AvgCompletionTime float64        `json:"avg_completion_time_minutes"`
	ProcessHealth     *ProcessHealth `json:"process_health"`
	TopProcesses      []ProcessStat  `json:"top_processes"`
	TaskDistribution  []TaskStat     `json:"task_distribution"`
	TrendData         []TrendPoint   `json:"trend_data"`
}

// ProcessHealth 流程健康度
type ProcessHealth struct {
	Healthy     int     `json:"healthy"`
	Warning     int     `json:"warning"`
	Critical    int     `json:"critical"`
	HealthScore float64 `json:"health_score"` // 0-100
}

// ProcessStat 流程统计
type ProcessStat struct {
	ProcessDefinitionKey string  `json:"process_definition_key"`
	TotalInstances       int     `json:"total_instances"`
	RunningInstances     int     `json:"running_instances"`
	CompletedInstances   int     `json:"completed_instances"`
	AvgDuration          float64 `json:"avg_duration_minutes"`
}

// TaskStat 任务统计
type TaskStat struct {
	Status  string  `json:"status"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// TrendPoint 趋势数据点
type TrendPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// GetDashboardMetrics 获取仪表盘指标
func (s *BPMNMetricsService) GetDashboardMetrics(ctx context.Context, tenantID int, startTime, endTime time.Time) (*DashboardMetrics, error) {
	// 总流程定义数
	totalProcesses, err := s.client.ProcessDefinition.Query().
		Where(processdefinition.TenantID(tenantID)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 活跃流程实例
	activeInstances, err := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.Status("running")).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 今日完成数
	today := time.Now().Format("2006-01-02")
	todayStart, _ := time.Parse("2006-01-02", today)
	completedToday, err := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.Status("completed")).
		Where(processinstance.EndTimeGTE(todayStart)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 待处理任务
	openTasks, err := s.client.ProcessTask.Query().
		Where(processtask.TenantID(tenantID)).
		Where(processtask.StatusNEQ("completed")).
		Where(processtask.StatusNEQ("cancelled")).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// SLA合规率
	slaRate := 100.0
	definitions, err := s.client.ProcessDefinition.Query().
		Where(processdefinition.TenantID(tenantID)).
		All(ctx)
	if err == nil && len(definitions) > 0 {
		var totalRate float64
		var count int
		for _, def := range definitions {
			rate, _, _, err := s.slaService.GetSLAComplianceRate(ctx, def.Key, startTime, endTime, tenantID)
			if err == nil {
				totalRate += rate
				count++
			}
		}
		if count > 0 {
			slaRate = totalRate / float64(count)
		}
	}

	// 平均完成时间
	avgDuration := s.calculateAvgCompletionTime(ctx, tenantID, startTime, endTime)

	// 流程健康度
	health := s.calculateProcessHealth(ctx, tenantID)

	// 热门流程
	topProcesses := s.getTopProcesses(ctx, tenantID, startTime, endTime)

	// 任务分布
	taskDistribution := s.getTaskDistribution(ctx, tenantID)

	// 趋势数据
	trendData := s.getTrendData(ctx, tenantID, startTime, endTime)

	return &DashboardMetrics{
		TotalProcesses:    totalProcesses,
		ActiveInstances:   activeInstances,
		CompletedToday:    completedToday,
		OpenTasks:         openTasks,
		SLAComplianceRate: slaRate,
		AvgCompletionTime: avgDuration,
		ProcessHealth:     health,
		TopProcesses:      topProcesses,
		TaskDistribution:  taskDistribution,
		TrendData:         trendData,
	}, nil
}

// calculateAvgCompletionTime 计算平均完成时间
func (s *BPMNMetricsService) calculateAvgCompletionTime(ctx context.Context, tenantID int, startTime, endTime time.Time) float64 {
	instances, err := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.Status("completed")).
		Where(processinstance.StartTimeGTE(startTime)).
		Where(processinstance.StartTimeLTE(endTime)).
		All(ctx)
	if err != nil || len(instances) == 0 {
		return 0
	}

	var totalDuration float64
	for _, instance := range instances {
		if !instance.EndTime.IsZero() {
			duration := instance.EndTime.Sub(instance.StartTime).Minutes()
			totalDuration += duration
		}
	}

	return totalDuration / float64(len(instances))
}

// calculateProcessHealth 计算流程健康度
func (s *BPMNMetricsService) calculateProcessHealth(ctx context.Context, tenantID int) *ProcessHealth {
	// 获取所有运行中的实例
	instances, err := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.Status("running")).
		All(ctx)
	if err != nil {
		return &ProcessHealth{Healthy: 0, Warning: 0, Critical: 0, HealthScore: 0}
	}

	healthy := 0
	warning := 0
	critical := 0

	for _, instance := range instances {
		info, err := s.slaService.GetProcessInstanceSLAInfo(ctx, instance)
		if err != nil {
			continue
		}

		switch info.Status {
		case SLAStatusOK:
			healthy++
		case SLAStatusWarning:
			warning++
		case SLAStatusBreached:
			critical++
		}
	}

	total := healthy + warning + critical
	healthScore := 0.0
	if total > 0 {
		healthScore = float64(healthy*100+warning*50) / float64(total)
	}

	return &ProcessHealth{
		Healthy:     healthy,
		Warning:     warning,
		Critical:    critical,
		HealthScore: healthScore,
	}
}

// getTopProcesses 获取热门流程
func (s *BPMNMetricsService) getTopProcesses(ctx context.Context, tenantID int, startTime, endTime time.Time) []ProcessStat {
	instances, err := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.StartTimeGTE(startTime)).
		Where(processinstance.StartTimeLTE(endTime)).
		All(ctx)
	if err != nil {
		return []ProcessStat{}
	}

	// 按流程定义分组统计
	processMap := make(map[string]*ProcessStat)
	for _, instance := range instances {
		stat, ok := processMap[instance.ProcessDefinitionKey]
		if !ok {
			stat = &ProcessStat{
				ProcessDefinitionKey: instance.ProcessDefinitionKey,
			}
			processMap[instance.ProcessDefinitionKey] = stat
		}
		stat.TotalInstances++
		if instance.Status == "running" {
			stat.RunningInstances++
		} else if instance.Status == "completed" {
			stat.CompletedInstances++
			if !instance.EndTime.IsZero() {
				duration := instance.EndTime.Sub(instance.StartTime).Minutes()
				stat.AvgDuration = (stat.AvgDuration*float64(stat.CompletedInstances-1) + duration) / float64(stat.CompletedInstances)
			}
		}
	}

	// 转换为切片并排序
	result := make([]ProcessStat, 0, len(processMap))
	for _, stat := range processMap {
		result = append(result, *stat)
	}

	// 按总实例数排序
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].TotalInstances > result[i].TotalInstances {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	// 只返回前10个
	if len(result) > 10 {
		result = result[:10]
	}

	return result
}

// getTaskDistribution 获取任务分布
func (s *BPMNMetricsService) getTaskDistribution(ctx context.Context, tenantID int) []TaskStat {
	tasks, err := s.client.ProcessTask.Query().
		Where(processtask.TenantID(tenantID)).
		Where(processtask.StatusNEQ("cancelled")).
		All(ctx)
	if err != nil {
		return []TaskStat{}
	}

	total := len(tasks)
	distribution := make(map[string]int)

	for _, task := range tasks {
		distribution[task.Status]++
	}

	result := make([]TaskStat, 0, len(distribution))
	for status, count := range distribution {
		percent := 0.0
		if total > 0 {
			percent = float64(count) / float64(total) * 100
		}
		result = append(result, TaskStat{
			Status:  status,
			Count:   count,
			Percent: percent,
		})
	}

	return result
}

// getTrendData 获取趋势数据
func (s *BPMNMetricsService) getTrendData(ctx context.Context, tenantID int, startTime, endTime time.Time) []TrendPoint {
	result := []TrendPoint{}

	// 按天统计
	current := startTime
	for current.Before(endTime) {
		nextDay := current.AddDate(0, 0, 1)

		count, err := s.client.ProcessInstance.Query().
			Where(processinstance.TenantID(tenantID)).
			Where(processinstance.StartTimeGTE(current)).
			Where(processinstance.StartTimeLT(nextDay)).
			Count(ctx)
		if err == nil {
			result = append(result, TrendPoint{
				Date:  current.Format("2006-01-02"),
				Count: count,
			})
		}

		current = nextDay
	}

	return result
}

// GetProcessMetrics 获取流程指标
func (s *BPMNMetricsService) GetProcessMetrics(ctx context.Context, processDefinitionKey string, tenantID int, startTime, endTime time.Time) (*BPMNProcessMetrics, error) {
	// 实例统计
	totalInstances, err := s.client.ProcessInstance.Query().
		Where(processinstance.ProcessDefinitionKey(processDefinitionKey)).
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.StartTimeGTE(startTime)).
		Where(processinstance.StartTimeLTE(endTime)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	runningInstances, err := s.client.ProcessInstance.Query().
		Where(processinstance.ProcessDefinitionKey(processDefinitionKey)).
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.Status("running")).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	completedInstances, err := s.client.ProcessInstance.Query().
		Where(processinstance.ProcessDefinitionKey(processDefinitionKey)).
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.Status("completed")).
		Where(processinstance.StartTimeGTE(startTime)).
		Where(processinstance.StartTimeLTE(endTime)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	terminatedInstances, err := s.client.ProcessInstance.Query().
		Where(processinstance.ProcessDefinitionKey(processDefinitionKey)).
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.Status("terminated")).
		Where(processinstance.StartTimeGTE(startTime)).
		Where(processinstance.StartTimeLTE(endTime)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// SLA合规率
	slaRate, _, _, err := s.slaService.GetSLAComplianceRate(ctx, processDefinitionKey, startTime, endTime, tenantID)
	if err != nil {
		slaRate = 100.0
	}

	// 平均处理时间
	avgDuration := s.calculateAvgCompletionTime(ctx, tenantID, startTime, endTime)

	return &BPMNProcessMetrics{
		TotalInstances:      totalInstances,
		RunningInstances:    runningInstances,
		CompletedInstances:  completedInstances,
		TerminatedInstances: terminatedInstances,
		CompletionRate: func() float64 {
			if totalInstances == 0 {
				return 0
			}
			return float64(completedInstances) / float64(totalInstances) * 100
		}(),
		SLAComplianceRate: slaRate,
		AvgCompletionTime: avgDuration,
	}, nil
}

// BPMNProcessMetrics 流程指标
type BPMNProcessMetrics struct {
	TotalInstances      int     `json:"total_instances"`
	RunningInstances    int     `json:"running_instances"`
	CompletedInstances  int     `json:"completed_instances"`
	TerminatedInstances int     `json:"terminated_instances"`
	CompletionRate      float64 `json:"completion_rate"`
	SLAComplianceRate   float64 `json:"sla_compliance_rate"`
	AvgCompletionTime   float64 `json:"avg_completion_time_minutes"`
}

// GetBottleneckAnalysis 获取瓶颈分析
func (s *BPMNMetricsService) GetBottleneckAnalysis(ctx context.Context, processDefinitionKey string, tenantID int) ([]BottleneckInfo, error) {
	// 获取已完成的任务，分析任务处理时间
	tasks, err := s.client.ProcessTask.Query().
		Where(processtask.ProcessDefinitionKey(processDefinitionKey)).
		Where(processtask.TenantID(tenantID)).
		Where(processtask.Status("completed")).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// 按任务名称分组统计
	taskStats := make(map[string]*TaskStats)
	for _, task := range tasks {
		stats, ok := taskStats[task.TaskName]
		if !ok {
			stats = &TaskStats{
				TaskName: task.TaskName,
			}
			taskStats[task.TaskName] = stats
		}
		stats.Count++
		if !task.CompletedTime.IsZero() && !task.CreatedTime.IsZero() {
			duration := task.CompletedTime.Sub(task.CreatedTime).Minutes()
			stats.TotalDuration += duration
			stats.AvgDuration = stats.TotalDuration / float64(stats.Count)
			if stats.MaxDuration < duration {
				stats.MaxDuration = duration
			}
			if stats.MinDuration == 0 || stats.MinDuration > duration {
				stats.MinDuration = duration
			}
		}
	}

	// 转换为切片
	result := make([]BottleneckInfo, 0, len(taskStats))
	for _, stats := range taskStats {
		result = append(result, BottleneckInfo{
			TaskName:    stats.TaskName,
			TotalCount:  stats.Count,
			AvgDuration: stats.AvgDuration,
			MaxDuration: stats.MaxDuration,
			MinDuration: stats.MinDuration,
		})
	}

	// 按平均处理时间排序，找出瓶颈
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].AvgDuration > result[i].AvgDuration {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

// TaskStats 任务统计
type TaskStats struct {
	TaskName      string
	Count         int
	TotalDuration float64
	AvgDuration   float64
	MaxDuration   float64
	MinDuration   float64
}

// BottleneckInfo 瓶颈信息
type BottleneckInfo struct {
	TaskName    string  `json:"task_name"`
	TotalCount  int     `json:"total_count"`
	AvgDuration float64 `json:"avg_duration_minutes"`
	MaxDuration float64 `json:"max_duration_minutes"`
	MinDuration float64 `json:"min_duration_minutes"`
}

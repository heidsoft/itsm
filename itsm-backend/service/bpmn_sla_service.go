package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"

	"go.uber.org/zap"
)

// BPMNSLAService BPMN SLA服务
type BPMNSLAService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewBPMNSLAService 创建BPMN SLA服务
func NewBPMNSLAService(client *ent.Client, logger *zap.SugaredLogger) *BPMNSLAService {
	return &BPMNSLAService{
		client: client,
		logger: logger,
	}
}

// ProcessSLA 流程SLA定义
type ProcessSLA struct {
	ProcessDefinitionKey string `json:"process_definition_key"`
	TaskDefinitionKey    string `json:"task_definition_key,omitempty"`
	MilestoneName        string `json:"milestone_name"` // 开始->审批, 审批->执行
	DeadlineMinutes      int    `json:"deadline_minutes"`
	WarningMinutes       int    `json:"warning_minutes"` // 预警时间
	BusinessHoursOnly    bool   `json:"business_hours_only"`
	Priority             string `json:"priority,omitempty"` // 对应任务优先级
}

// SLAStatus SLA状态
const (
	SLAStatusOK       = "ok"
	SLAStatusWarning  = "warning"
	SLAStatusBreached = "breached"
	SLAStatusUnknown  = "unknown"
)

// GetProcessSLA 获取流程SLA配置
func (s *BPMNSLAService) GetProcessSLA(ctx context.Context, processDefinitionKey string) (*ProcessSLA, error) {
	// 从流程定义中获取SLA配置
	definition, err := s.client.ProcessDefinition.Query().
		Where(processdefinition.Key(processDefinitionKey)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	// 从流程变量中获取SLA配置
	slaConfig, ok := definition.ProcessVariables["sla"].(map[string]interface{})
	if !ok {
		// 返回默认SLA配置
		return &ProcessSLA{
			ProcessDefinitionKey: processDefinitionKey,
			MilestoneName:        "process_completion",
			DeadlineMinutes:      480, // 8小时
			WarningMinutes:       360, // 6小时
			BusinessHoursOnly:    true,
		}, nil
	}

	deadlineMinutes := 480
	if v, ok := slaConfig["deadline_minutes"].(float64); ok {
		deadlineMinutes = int(v)
	}

	warningMinutes := 360
	if v, ok := slaConfig["warning_minutes"].(float64); ok {
		warningMinutes = int(v)
	}

	return &ProcessSLA{
		ProcessDefinitionKey: processDefinitionKey,
		MilestoneName:        "process_completion",
		DeadlineMinutes:      deadlineMinutes,
		WarningMinutes:       warningMinutes,
		BusinessHoursOnly:    true,
	}, nil
}

// GetTaskSLA 获取任务SLA配置
func (s *BPMNSLAService) GetTaskSLA(ctx context.Context, task *ent.ProcessTask) (*ProcessSLA, error) {
	// 从任务变量中获取SLA配置
	slaConfig, ok := task.TaskVariables["sla"].(map[string]interface{})
	if !ok {
		// 从流程定义中获取默认SLA配置
		sla, err := s.GetProcessSLA(ctx, task.ProcessDefinitionKey)
		if err != nil {
			return nil, err
		}
		sla.TaskDefinitionKey = task.TaskDefinitionKey
		sla.MilestoneName = task.TaskName
		// 任务默认2小时
		sla.DeadlineMinutes = 120
		sla.WarningMinutes = 90
		return sla, nil
	}

	deadlineMinutes := 120
	if v, ok := slaConfig["deadline_minutes"].(float64); ok {
		deadlineMinutes = int(v)
	}

	warningMinutes := 90
	if v, ok := slaConfig["warning_minutes"].(float64); ok {
		warningMinutes = int(v)
	}

	return &ProcessSLA{
		ProcessDefinitionKey: task.ProcessDefinitionKey,
		TaskDefinitionKey:    task.TaskDefinitionKey,
		MilestoneName:        task.TaskName,
		DeadlineMinutes:      deadlineMinutes,
		WarningMinutes:       warningMinutes,
		BusinessHoursOnly:    true,
	}, nil
}

// CalculateSLAStatus 计算SLA状态
func (s *BPMNSLAService) CalculateSLAStatus(ctx context.Context, startTime time.Time, sla *ProcessSLA) (string, time.Time, time.Time, error) {
	var deadline, warning time.Time

	if sla.BusinessHoursOnly {
		// 仅计算工作时间
		deadline = s.calculateBusinessHoursDeadline(startTime, sla.DeadlineMinutes)
		warning = s.calculateBusinessHoursDeadline(startTime, sla.WarningMinutes)
	} else {
		deadline = startTime.Add(time.Duration(sla.DeadlineMinutes) * time.Minute)
		warning = startTime.Add(time.Duration(sla.WarningMinutes) * time.Minute)
	}

	now := time.Now()
	status := SLAStatusOK

	if now.After(deadline) {
		status = SLAStatusBreached
	} else if now.After(warning) {
		status = SLAStatusWarning
	}

	return status, deadline, warning, nil
}

// calculateBusinessHoursDeadline 计算工作时间截止时间
func (s *BPMNSLAService) calculateBusinessHoursDeadline(startTime time.Time, minutes int) time.Time {
	// 默认工作时间为周一至周五 9:00-18:00
	workStartHour := 9
	workEndHour := 18

	remainingMinutes := minutes
	current := startTime

	// 如果在工作时间外，调整到下一个工作日开始
	hour := current.Hour()
	if hour < workStartHour {
		current = time.Date(current.Year(), current.Month(), current.Day(), workStartHour, 0, 0, 0, current.Location())
	} else if hour >= workEndHour {
		// 移到第二天
		current = time.Date(current.Year(), current.Month(), current.Day()+1, workStartHour, 0, 0, 0, current.Location())
	}

	for remainingMinutes > 0 {
		// 检查是否在工作日
		weekday := current.Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			// 移到下一个工作日
			daysToAdd := 1
			if weekday == time.Saturday {
				daysToAdd = 2
			}
			current = current.AddDate(0, 0, daysToAdd)
			current = time.Date(current.Year(), current.Month(), current.Day(), workStartHour, 0, 0, 0, current.Location())
			continue
		}

		// 计算当天剩余工作时间
		hour = current.Hour()
		if hour < workStartHour {
			current = time.Date(current.Year(), current.Month(), current.Day(), workStartHour, 0, 0, 0, current.Location())
			hour = workStartHour
		}

		minutesInDay := (workEndHour - hour) * 60
		if remainingMinutes <= minutesInDay {
			current = current.Add(time.Duration(remainingMinutes) * time.Minute)
			remainingMinutes = 0
		} else {
			remainingMinutes -= minutesInDay
			// 移到下一天
			current = time.Date(current.Year(), current.Month(), current.Day()+1, workStartHour, 0, 0, 0, current.Location())
		}
	}

	return current
}

// GetProcessInstanceSLAInfo 获取流程实例SLA信息
func (s *BPMNSLAService) GetProcessInstanceSLAInfo(ctx context.Context, instance *ent.ProcessInstance) (*SLAInfo, error) {
	sla, err := s.GetProcessSLA(ctx, instance.ProcessDefinitionKey)
	if err != nil {
		return nil, err
	}

	status, deadline, warning, err := s.CalculateSLAStatus(ctx, instance.StartTime, sla)
	if err != nil {
		return nil, err
	}

	elapsedMinutes := int(time.Since(instance.StartTime).Minutes())

	return &SLAInfo{
		Status:         status,
		Deadline:       deadline,
		Warning:        warning,
		ElapsedMinutes: elapsedMinutes,
		TotalMinutes:   sla.DeadlineMinutes,
		RemainingMinutes: func() int {
			if status == SLAStatusBreached {
				return 0
			}
			return int(time.Until(deadline).Minutes())
		}(),
	}, nil
}

// GetTaskSLAInfo 获取任务SLA信息
func (s *BPMNSLAService) GetTaskSLAInfo(ctx context.Context, task *ent.ProcessTask) (*SLAInfo, error) {
	sla, err := s.GetTaskSLA(ctx, task)
	if err != nil {
		return nil, err
	}

	// 使用任务的创建时间或分配时间作为起点
	startTime := task.CreatedAt
	if !task.AssignedTime.IsZero() {
		startTime = task.AssignedTime
	}

	status, deadline, warning, err := s.CalculateSLAStatus(ctx, startTime, sla)
	if err != nil {
		return nil, err
	}

	elapsedMinutes := int(time.Since(startTime).Minutes())

	return &SLAInfo{
		Status:         status,
		Deadline:       deadline,
		Warning:        warning,
		ElapsedMinutes: elapsedMinutes,
		TotalMinutes:   sla.DeadlineMinutes,
		RemainingMinutes: func() int {
			if status == SLAStatusBreached {
				return 0
			}
			return int(time.Until(deadline).Minutes())
		}(),
	}, nil
}

// CheckSLAViolations 检查SLA违规
func (s *BPMNSLAService) CheckSLAViolations(ctx context.Context, tenantID int) ([]*SLAViolation, error) {
	violations := []*SLAViolation{}

	// 1. 检查流程实例SLA
	instances, err := s.client.ProcessInstance.Query().
		Where(processinstance.Status("running")).
		Where(processinstance.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询流程实例失败: %w", err)
	}

	for _, instance := range instances {
		info, err := s.GetProcessInstanceSLAInfo(ctx, instance)
		if err != nil {
			continue
		}

		if info.Status == SLAStatusBreached || info.Status == SLAStatusWarning {
			violations = append(violations, &SLAViolation{
				ResourceType:   "process_instance",
				ResourceID:     instance.ID,
				ResourceKey:    instance.ProcessInstanceID,
				SLAStatus:      info.Status,
				StartTime:      instance.StartTime,
				Deadline:       info.Deadline,
				ElapsedMinutes: info.ElapsedMinutes,
				TenantID:       instance.TenantID,
			})
		}
	}

	// 2. 检查任务SLA
	tasks, err := s.client.ProcessTask.Query().
		Where(processtask.StatusNEQ("completed")).
		Where(processtask.StatusNEQ("cancelled")).
		Where(processtask.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}

	for _, task := range tasks {
		info, err := s.GetTaskSLAInfo(ctx, task)
		if err != nil {
			continue
		}

		if info.Status == SLAStatusBreached || info.Status == SLAStatusWarning {
			violations = append(violations, &SLAViolation{
				ResourceType:   "task",
				ResourceID:     task.ID,
				ResourceKey:    task.TaskID,
				SLAStatus:      info.Status,
				StartTime:      task.CreatedAt,
				Deadline:       info.Deadline,
				ElapsedMinutes: info.ElapsedMinutes,
				TenantID:       task.TenantID,
			})
		}
	}

	return violations, nil
}

// RecordSLAAlert 记录SLA告警
// 注意: 当前使用日志记录，未来可以扩展为独立的BPMN SLA告警表
func (s *BPMNSLAService) RecordSLAAlert(ctx context.Context, violation *SLAViolation) error {
	// 记录到日志
	s.logger.Infow("SLA Alert",
		"resource_type", violation.ResourceType,
		"resource_key", violation.ResourceKey,
		"status", violation.SLAStatus,
		"elapsed_minutes", violation.ElapsedMinutes,
	)
	return nil
}

// GetSLAComplianceRate 获取SLA合规率
func (s *BPMNSLAService) GetSLAComplianceRate(ctx context.Context, processDefinitionKey string, startTime, endTime time.Time, tenantID int) (float64, int, int, error) {
	// 统计已完成和终止的流程实例
	completedInstances, err := s.client.ProcessInstance.Query().
		Where(processinstance.ProcessDefinitionKey(processDefinitionKey)).
		Where(processinstance.ProcessDefinitionKey(processDefinitionKey)).
		Where(processinstance.TenantID(tenantID)).
		Where(processinstance.StartTimeGTE(startTime)).
		Where(processinstance.StartTimeLTE(endTime)).
		Where(processinstance.StatusIn("completed", "terminated")).
		All(ctx)
	if err != nil {
		return 0, 0, 0, err
	}

	total := len(completedInstances)
	compliant := 0

	for _, instance := range completedInstances {
		info, err := s.GetProcessInstanceSLAInfo(ctx, instance)
		if err != nil {
			continue
		}
		if info.Status == SLAStatusOK {
			compliant++
		}
	}

	if total == 0 {
		return 100.0, 0, 0, nil
	}

	rate := float64(compliant) / float64(total) * 100
	return rate, compliant, total, nil
}

// SLAInfo SLA信息
type SLAInfo struct {
	Status           string    `json:"status"`
	Deadline         time.Time `json:"deadline"`
	Warning          time.Time `json:"warning"`
	ElapsedMinutes   int       `json:"elapsed_minutes"`
	TotalMinutes     int       `json:"total_minutes"`
	RemainingMinutes int       `json:"remaining_minutes"`
}

// SLAViolation SLA违规
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

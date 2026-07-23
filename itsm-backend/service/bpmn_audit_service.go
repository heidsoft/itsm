package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processauditlog"

	"go.uber.org/zap"
)

// BPMNAuditService BPMN审计服务
type BPMNAuditService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewBPMNAuditService 创建BPMN审计服务
func NewBPMNAuditService(client *ent.Client, logger *zap.SugaredLogger) *BPMNAuditService {
	return &BPMNAuditService{
		client: client,
		logger: logger,
	}
}

// AuditAction 审计操作类型
const (
	AuditActionProcessStarted    = "started"
	AuditActionProcessCompleted  = "completed"
	AuditActionProcessSuspended  = "suspended"
	AuditActionProcessResumed    = "resumed"
	AuditActionProcessTerminated = "terminated"
	AuditActionTaskAssigned      = "assigned"
	AuditActionTaskUnassigned    = "unassigned"
	AuditActionTaskClaimed       = "claimed"
	AuditActionTaskCompleted     = "completed"
	AuditActionTaskCancelled     = "cancelled"
	AuditActionTaskEscalated     = "escalated"
	AuditActionTaskReassigned    = "reassigned"
	AuditActionVariableChanged   = "variable_changed"
	AuditActionActivityStarted   = "activity_started"
	AuditActionActivityCompleted = "activity_completed"
)

// ActivityType 活动类型
const (
	ActivityTypeStartEvent        = "startEvent"
	ActivityTypeEndEvent          = "endEvent"
	ActivityTypeUserTask          = "userTask"
	ActivityTypeServiceTask       = "serviceTask"
	ActivityTypeScriptTask        = "scriptTask"
	ActivityTypeManualTask        = "manualTask"
	ActivityTypeGateway           = "gateway"
	ActivityTypeSubProcess        = "subProcess"
	ActivityTypeIntermediateEvent = "intermediateEvent"
)

// AuditContext 审计上下文
type AuditContext struct {
	ProcessInstanceID    int    // ProcessInstance 表的数据库整数主键（instance.ID）
	ProcessInstanceKey   string // BPMN 流程实例业务键（instance.ProcessInstanceID，例如 PI-change-123）
	ProcessDefinitionKey string
	ProcessDefinitionID  int
	ActivityID           string
	ActivityName         string
	ActivityType         string
	Action               string
	UserID               int
	UserName             string
	AssigneeID           int
	AssigneeName         string
	VariablesBefore      map[string]interface{}
	VariablesAfter       map[string]interface{}
	Comment              string
	IPAddress            string
	UserAgent            string
	TenantID             int
	DurationMs           int
	Metadata             map[string]interface{}
}

// RecordAudit 记录审计日志
func (s *BPMNAuditService) RecordAudit(ctx context.Context, auditCtx *AuditContext) error {
	startTime := time.Now()

	// 构建审计日志
	create := s.client.ProcessAuditLog.Create().
		SetProcessInstanceID(auditCtx.ProcessInstanceID).
		SetProcessInstanceKey(auditCtx.ProcessInstanceKey).
		SetProcessDefinitionKey(auditCtx.ProcessDefinitionKey).
		SetProcessDefinitionID(auditCtx.ProcessDefinitionID).
		SetActivityID(auditCtx.ActivityID).
		SetActivityName(auditCtx.ActivityName).
		SetActivityType(auditCtx.ActivityType).
		SetAction(auditCtx.Action).
		SetUserID(auditCtx.UserID).
		SetUserName(auditCtx.UserName).
		SetAssigneeID(auditCtx.AssigneeID).
		SetAssigneeName(auditCtx.AssigneeName).
		SetVariablesBefore(auditCtx.VariablesBefore).
		SetVariablesAfter(auditCtx.VariablesAfter).
		SetComment(auditCtx.Comment).
		SetIPAddress(auditCtx.IPAddress).
		SetUserAgent(auditCtx.UserAgent).
		SetTenantID(auditCtx.TenantID).
		SetTimestamp(time.Now()).
		SetMetadata(auditCtx.Metadata)

	// 如果没有设置durationMs，在保存后计算
	if auditCtx.DurationMs > 0 {
		create.SetDurationMs(auditCtx.DurationMs)
	} else {
		create.SetDurationMs(int(time.Since(startTime).Milliseconds()))
	}

	_, err := create.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to record audit log", "error", err, "action", auditCtx.Action)
		return fmt.Errorf("记录审计日志失败: %w", err)
	}

	s.logger.Debugw("Audit log recorded", "action", auditCtx.Action, "processInstanceKey", auditCtx.ProcessInstanceKey)
	return nil
}

// RecordProcessStarted 记录流程启动
func (s *BPMNAuditService) RecordProcessStarted(ctx context.Context, instance *ent.ProcessInstance, userID int, userName string, variables map[string]interface{}) error {
	return s.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    instance.ID,
		ProcessInstanceKey:   instance.ProcessInstanceID,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessDefinitionID:  instance.ProcessDefinitionID,
		ActivityID:           "start",
		ActivityName:         "流程开始",
		ActivityType:         ActivityTypeStartEvent,
		Action:               AuditActionProcessStarted,
		UserID:               userID,
		UserName:             userName,
		VariablesBefore:      nil,
		VariablesAfter:       variables,
		TenantID:             instance.TenantID,
	})
}

// RecordProcessCompleted 记录流程完成
func (s *BPMNAuditService) RecordProcessCompleted(ctx context.Context, instance *ent.ProcessInstance, userID int, userName string, variables map[string]interface{}) error {
	return s.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    instance.ID,
		ProcessInstanceKey:   instance.ProcessInstanceID,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessDefinitionID:  instance.ProcessDefinitionID,
		ActivityID:           "end",
		ActivityName:         "流程结束",
		ActivityType:         ActivityTypeEndEvent,
		Action:               AuditActionProcessCompleted,
		UserID:               userID,
		UserName:             userName,
		VariablesBefore:      instance.Variables,
		VariablesAfter:       variables,
		TenantID:             instance.TenantID,
	})
}

// RecordTaskAssigned 记录任务分配
func (s *BPMNAuditService) RecordTaskAssigned(ctx context.Context, task *ent.ProcessTask, assignee *ent.User, assignerID int, assignerName string) error {
	assigneeName := ""
	assigneeID := 0
	if assignee != nil {
		assigneeName = assignee.Name
		assigneeID = assignee.ID
	}

	// Get process instance to get the process instance key
	instance, err := s.client.ProcessInstance.Get(ctx, task.ProcessInstanceID)
	processInstanceKey := ""
	processDefinitionID := 0
	tenantID := 0
	if err == nil {
		processInstanceKey = instance.ProcessInstanceID
		processDefinitionID = instance.ProcessDefinitionID
		tenantID = instance.TenantID
	}

	return s.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    task.ProcessInstanceID,
		ProcessInstanceKey:   processInstanceKey,
		ProcessDefinitionKey: task.ProcessDefinitionKey,
		ProcessDefinitionID:  processDefinitionID,
		ActivityID:           task.TaskDefinitionKey,
		ActivityName:         task.TaskName,
		ActivityType:         task.TaskType,
		Action:               AuditActionTaskAssigned,
		UserID:               assignerID,
		UserName:             assignerName,
		AssigneeID:           assigneeID,
		AssigneeName:         assigneeName,
		TenantID:             tenantID,
	})
}

// RecordTaskClaimed 记录任务签收
func (s *BPMNAuditService) RecordTaskClaimed(ctx context.Context, task *ent.ProcessTask, userID int, userName string) error {
	// Get process instance to get the process instance key
	instance, err := s.client.ProcessInstance.Get(ctx, task.ProcessInstanceID)
	processInstanceKey := ""
	processDefinitionID := 0
	tenantID := 0
	if err == nil {
		processInstanceKey = instance.ProcessInstanceID
		processDefinitionID = instance.ProcessDefinitionID
		tenantID = instance.TenantID
	}

	return s.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    task.ProcessInstanceID,
		ProcessInstanceKey:   processInstanceKey,
		ProcessDefinitionKey: task.ProcessDefinitionKey,
		ProcessDefinitionID:  processDefinitionID,
		ActivityID:           task.TaskDefinitionKey,
		ActivityName:         task.TaskName,
		ActivityType:         task.TaskType,
		Action:               AuditActionTaskClaimed,
		UserID:               userID,
		UserName:             userName,
		AssigneeID:           userID,
		AssigneeName:         userName,
		TenantID:             tenantID,
	})
}

// RecordTaskCompleted 记录任务完成
func (s *BPMNAuditService) RecordTaskCompleted(ctx context.Context, task *ent.ProcessTask, userID int, userName string, variablesBefore, variablesAfter map[string]interface{}) error {
	// Get process instance to get the process instance key
	instance, err := s.client.ProcessInstance.Get(ctx, task.ProcessInstanceID)
	processInstanceKey := ""
	processDefinitionID := 0
	tenantID := 0
	if err == nil {
		processInstanceKey = instance.ProcessInstanceID
		processDefinitionID = instance.ProcessDefinitionID
		tenantID = instance.TenantID
	}

	return s.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    task.ProcessInstanceID,
		ProcessInstanceKey:   processInstanceKey,
		ProcessDefinitionKey: task.ProcessDefinitionKey,
		ProcessDefinitionID:  processDefinitionID,
		ActivityID:           task.TaskDefinitionKey,
		ActivityName:         task.TaskName,
		ActivityType:         task.TaskType,
		Action:               AuditActionTaskCompleted,
		UserID:               userID,
		UserName:             userName,
		VariablesBefore:      variablesBefore,
		VariablesAfter:       variablesAfter,
		TenantID:             tenantID,
	})
}

// RecordVariableChanged 记录变量变更
func (s *BPMNAuditService) RecordVariableChanged(ctx context.Context, instance *ent.ProcessInstance, userID int, userName string, varName string, oldValue, newValue interface{}) error {
	variablesBefore := map[string]interface{}{varName: oldValue}
	variablesAfter := map[string]interface{}{varName: newValue}

	return s.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    instance.ID,
		ProcessInstanceKey:   instance.ProcessInstanceID,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessDefinitionID:  instance.ProcessDefinitionID,
		ActivityID:           "variable",
		ActivityName:         "变量变更: " + varName,
		ActivityType:         ActivityTypeServiceTask,
		Action:               AuditActionVariableChanged,
		UserID:               userID,
		UserName:             userName,
		VariablesBefore:      variablesBefore,
		VariablesAfter:       variablesAfter,
		TenantID:             instance.TenantID,
	})
}

// QueryAuditLogs 查询审计日志
func (s *BPMNAuditService) QueryAuditLogs(ctx context.Context, req *QueryAuditLogsRequest) ([]*ent.ProcessAuditLog, int, error) {
	query := s.client.ProcessAuditLog.Query()

	// 构建查询条件
	if req.ProcessInstanceID > 0 {
		query = query.Where(processauditlog.ProcessInstanceID(req.ProcessInstanceID))
	}
	if req.ProcessInstanceKey != "" {
		query = query.Where(processauditlog.ProcessInstanceKey(req.ProcessInstanceKey))
	}
	if req.ProcessDefinitionKey != "" {
		query = query.Where(processauditlog.ProcessDefinitionKey(req.ProcessDefinitionKey))
	}
	if req.ActivityID != "" {
		query = query.Where(processauditlog.ActivityID(req.ActivityID))
	}
	if req.ActivityType != "" {
		query = query.Where(processauditlog.ActivityType(req.ActivityType))
	}
	if req.Action != "" {
		query = query.Where(processauditlog.Action(req.Action))
	}
	if req.UserID > 0 {
		query = query.Where(processauditlog.UserID(req.UserID))
	}
	if req.AssigneeID > 0 {
		query = query.Where(processauditlog.AssigneeID(req.AssigneeID))
	}
	if req.TenantID > 0 {
		query = query.Where(processauditlog.TenantID(req.TenantID))
	}
	if !req.StartTime.IsZero() {
		query = query.Where(processauditlog.TimestampGTE(req.StartTime))
	}
	if !req.EndTime.IsZero() {
		query = query.Where(processauditlog.TimestampLTE(req.EndTime))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询审计日志总数失败: %w", err)
	}

	// 分页查询
	if req.Page > 0 && req.PageSize > 0 {
		query = query.Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize)
	}

	// 排序
	if req.SortBy != "" {
		switch req.SortBy {
		case "timestamp":
			if req.SortOrder == "asc" {
				query = query.Order(ent.Asc(processauditlog.FieldTimestamp))
			} else {
				query = query.Order(ent.Desc(processauditlog.FieldTimestamp))
			}
		default:
			query = query.Order(ent.Desc(processauditlog.FieldTimestamp))
		}
	} else {
		query = query.Order(ent.Desc(processauditlog.FieldTimestamp))
	}

	logs, err := query.All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询审计日志失败: %w", err)
	}

	return logs, total, nil
}

// GetProcessTimeline 获取流程时间线
func (s *BPMNAuditService) GetProcessTimeline(ctx context.Context, processInstanceKey string, tenantID int) ([]*ent.ProcessAuditLog, error) {
	query := s.client.ProcessAuditLog.Query().
		Where(processauditlog.ProcessInstanceKey(processInstanceKey))

	if tenantID > 0 {
		query = query.Where(processauditlog.TenantID(tenantID))
	}

	logs, err := query.
		Order(ent.Asc(processauditlog.FieldTimestamp)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程时间线失败: %w", err)
	}
	return logs, nil
}

// GetUserActivity 获取用户活动
func (s *BPMNAuditService) GetUserActivity(ctx context.Context, userID int, tenantID int, startTime, endTime time.Time) ([]*ent.ProcessAuditLog, error) {
	query := s.client.ProcessAuditLog.Query().
		Where(processauditlog.UserID(userID)).
		Where(processauditlog.TimestampGTE(startTime)).
		Where(processauditlog.TimestampLTE(endTime))

	if tenantID > 0 {
		query = query.Where(processauditlog.TenantID(tenantID))
	}

	return query.Order(ent.Desc(processauditlog.FieldTimestamp)).All(ctx)
}

// GetActivityStatistics 获取活动统计
func (s *BPMNAuditService) GetActivityStatistics(ctx context.Context, processDefinitionKey string, tenantID int, startTime, endTime time.Time) (map[string]int, error) {
	stats := make(map[string]int)

	query := s.client.ProcessAuditLog.Query().
		Where(processauditlog.ProcessDefinitionKey(processDefinitionKey)).
		Where(processauditlog.TimestampGTE(startTime)).
		Where(processauditlog.TimestampLTE(endTime))

	if tenantID > 0 {
		query = query.Where(processauditlog.TenantID(tenantID))
	}

	logs, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取活动统计失败: %w", err)
	}

	for _, log := range logs {
		action := log.Action
		stats[action]++
	}

	return stats, nil
}

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

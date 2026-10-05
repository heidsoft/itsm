package dto

import (
	"time"

	"itsm-backend/ent"
)

// ProcessAuditLogResponse BPMN 流程审计日志的 API DTO。
//
// 历史缺陷（台账 E4-48）：/bpmn/dashboard/audit-logs 与 /audit-logs/user/:userId
// 直接序列化 Ent 模型，响应键为 snake_case（process_instance_key 等），且把
// tenant_id 发给前端；唯一生产消费方 /workflow/audit 页面按 camelCase 类型读取，
// 四列恒空。此处是该用例的唯一响应形态：camelCase、不含 TenantID
// （租户由认证上下文隐式收敛，无需回显给客户端）。
type ProcessAuditLogResponse struct {
	ID                   int                    `json:"id"`
	ProcessInstanceID    int                    `json:"processInstanceId"`
	ProcessInstanceKey   string                 `json:"processInstanceKey"`
	ProcessDefinitionKey string                 `json:"processDefinitionKey"`
	ProcessDefinitionID  int                    `json:"processDefinitionId"`
	ActivityID           string                 `json:"activityId"`
	ActivityName         string                 `json:"activityName"`
	ActivityType         string                 `json:"activityType"`
	Action               string                 `json:"action"`
	UserID               int                    `json:"userId"`
	UserName             string                 `json:"userName"`
	AssigneeID           int                    `json:"assigneeId"`
	AssigneeName         string                 `json:"assigneeName"`
	VariablesBefore      map[string]interface{} `json:"variablesBefore"`
	VariablesAfter       map[string]interface{} `json:"variablesAfter"`
	Comment              string                 `json:"comment"`
	IPAddress            string                 `json:"ipAddress"`
	UserAgent            string                 `json:"userAgent"`
	Timestamp            time.Time              `json:"timestamp"`
	DurationMs           int                    `json:"durationMs"`
	Metadata             map[string]interface{} `json:"metadata"`
}

// ToProcessAuditLogResponse 转换单条审计日志；nil 返回 nil。
func ToProcessAuditLogResponse(log *ent.ProcessAuditLog) *ProcessAuditLogResponse {
	if log == nil {
		return nil
	}
	return &ProcessAuditLogResponse{
		ID:                   log.ID,
		ProcessInstanceID:    log.ProcessInstanceID,
		ProcessInstanceKey:   log.ProcessInstanceKey,
		ProcessDefinitionKey: log.ProcessDefinitionKey,
		ProcessDefinitionID:  log.ProcessDefinitionID,
		ActivityID:           log.ActivityID,
		ActivityName:         log.ActivityName,
		ActivityType:         log.ActivityType,
		Action:               log.Action,
		UserID:               log.UserID,
		UserName:             log.UserName,
		AssigneeID:           log.AssigneeID,
		AssigneeName:         log.AssigneeName,
		VariablesBefore:      log.VariablesBefore,
		VariablesAfter:       log.VariablesAfter,
		Comment:              log.Comment,
		IPAddress:            log.IPAddress,
		UserAgent:            log.UserAgent,
		Timestamp:            log.Timestamp,
		DurationMs:           log.DurationMs,
		Metadata:             log.Metadata,
	}
}

// ToProcessAuditLogResponseList 批量转换，返回非 nil 切片（空结果序列化为 []）。
func ToProcessAuditLogResponseList(logs []*ent.ProcessAuditLog) []*ProcessAuditLogResponse {
	result := make([]*ProcessAuditLogResponse, 0, len(logs))
	for _, log := range logs {
		result = append(result, ToProcessAuditLogResponse(log))
	}
	return result
}

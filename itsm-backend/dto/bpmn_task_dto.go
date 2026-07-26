package dto

import (
	"strconv"
	"strings"
	"time"

	"itsm-backend/ent"
)

// BPMNTaskResponse 「我的待办」任务视图：任务字段 + 所属流程实例的业务上下文（camelCase）
type BPMNTaskResponse struct {
	ID                   int                    `json:"id"`
	TaskID               string                 `json:"taskId"`
	TaskDefinitionKey    string                 `json:"taskDefinitionKey"`
	TaskName             string                 `json:"taskName"`
	TaskType             string                 `json:"taskType"`
	Status               string                 `json:"status"`
	Priority             string                 `json:"priority"`
	Assignee             string                 `json:"assignee"`
	CandidateUsers       string                 `json:"candidateUsers"`
	CandidateGroups      string                 `json:"candidateGroups"`
	ProcessInstanceID    int                    `json:"processInstanceId"`
	ProcessInstanceKey   string                 `json:"processInstanceKey"`
	ProcessDefinitionKey string                 `json:"processDefinitionKey"`
	BusinessKey          string                 `json:"businessKey"`
	BusinessType         string                 `json:"businessType"`
	BusinessID           int                    `json:"businessId"`
	TaskPurpose          string                 `json:"taskPurpose"`
	FormKey              string                 `json:"formKey,omitempty"`
	TaskVariables        map[string]interface{} `json:"taskVariables,omitempty"`
	DueDate              *time.Time             `json:"dueDate,omitempty"`
	CreatedTime          time.Time              `json:"createdTime"`
}

// parseBusinessKey 解析 "ticket:123" 形式的 businessKey
func parseBusinessKey(businessKey string) (businessType string, businessID int) {
	parts := strings.SplitN(businessKey, ":", 2)
	if len(parts) != 2 {
		return "", 0
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil {
		return parts[0], 0
	}
	return parts[0], id
}

// ToBPMNTaskResponse 转换任务实体；instance 允许为 nil（历史数据缺实例时业务上下文留空）
func ToBPMNTaskResponse(task *ent.ProcessTask, instance *ent.ProcessInstance) *BPMNTaskResponse {
	resp := &BPMNTaskResponse{
		ID:                   task.ID,
		TaskID:               task.TaskID,
		TaskDefinitionKey:    task.TaskDefinitionKey,
		TaskName:             task.TaskName,
		TaskType:             task.TaskType,
		Status:               task.Status,
		Priority:             task.Priority,
		Assignee:             task.Assignee,
		CandidateUsers:       task.CandidateUsers,
		CandidateGroups:      task.CandidateGroups,
		ProcessInstanceID:    task.ProcessInstanceID,
		ProcessDefinitionKey: task.ProcessDefinitionKey,
		FormKey:              task.FormKey,
		TaskVariables:        task.TaskVariables,
		CreatedTime:          task.CreatedTime,
	}
	if !task.DueDate.IsZero() {
		due := task.DueDate
		resp.DueDate = &due
	}
	if purpose, ok := task.TaskVariables["taskPurpose"].(string); ok {
		resp.TaskPurpose = purpose
	}
	if instance != nil {
		resp.ProcessInstanceKey = instance.ProcessInstanceID
		resp.BusinessKey = instance.BusinessKey
		resp.BusinessType, resp.BusinessID = parseBusinessKey(instance.BusinessKey)
	}
	return resp
}

// ToBPMNTaskResponseList 批量转换，instances 以实例数据库 ID 为键
func ToBPMNTaskResponseList(tasks []*ent.ProcessTask, instances map[int]*ent.ProcessInstance) []*BPMNTaskResponse {
	result := make([]*BPMNTaskResponse, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, ToBPMNTaskResponse(task, instances[task.ProcessInstanceID]))
	}
	return result
}

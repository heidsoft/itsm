package dto

import (
	"time"

	"itsm-backend/ent"
)

// BPMNProcessInstanceResponse BPMN流程实例响应（camelCase JSON，供前端使用）
type BPMNProcessInstanceResponse struct {
	ID                    string                 `json:"id"`
	InstanceID            string                 `json:"instanceId"`
	BusinessKey           string                 `json:"businessKey"`
	ProcessDefinitionKey  string                 `json:"processDefinitionKey"`
	ProcessDefinitionID   int                    `json:"processDefinitionId"`
	Status                string                 `json:"status"`
	CurrentActivityID     string                 `json:"currentActivityId"`
	CurrentActivityName   string                 `json:"currentActivityName"`
	Variables             map[string]interface{} `json:"variables"`
	StartTime             time.Time              `json:"startTime"`
	EndTime               time.Time              `json:"endTime,omitempty"`
	SuspendedTime         time.Time              `json:"suspendedTime,omitempty"`
	SuspendedReason       string                 `json:"suspendedReason"`
	TenantID              int                    `json:"tenantId"`
	Version               int                    `json:"version"`
	Initiator             string                 `json:"initiator"`
	ParentProcessInstanceID string               `json:"parentProcessInstanceId"`
	RootProcessInstanceID   string               `json:"rootProcessInstanceId"`
	CreatedAt             time.Time              `json:"createdAt"`
	UpdatedAt             time.Time              `json:"updatedAt"`
}

// BPMNProcessInstanceListResponse 流程实例列表响应
type BPMNProcessInstanceListResponse struct {
	Items      []*BPMNProcessInstanceResponse `json:"items"`
	Total      int64                          `json:"total"`
	Page       int                            `json:"page"`
	PageSize   int                            `json:"pageSize"`
	TotalPages int                            `json:"totalPages"`
}

// ToBPMNProcessInstanceResponse 将 ent.ProcessInstance 转换为 BPMNProcessInstanceResponse
func ToBPMNProcessInstanceResponse(p *ent.ProcessInstance) *BPMNProcessInstanceResponse {
	if p == nil {
		return nil
	}
	return &BPMNProcessInstanceResponse{
		ID:                    p.ProcessInstanceID,
		InstanceID:            p.ProcessInstanceID,
		BusinessKey:           p.BusinessKey,
		ProcessDefinitionKey:  p.ProcessDefinitionKey,
		ProcessDefinitionID:    p.ProcessDefinitionID,
		Status:                p.Status,
		CurrentActivityID:     p.CurrentActivityID,
		CurrentActivityName:   p.CurrentActivityName,
		Variables:             p.Variables,
		StartTime:             p.StartTime,
		EndTime:               p.EndTime,
		SuspendedTime:         p.SuspendedTime,
		SuspendedReason:       p.SuspendedReason,
		TenantID:              p.TenantID,
		Version:               p.Version,
		Initiator:             p.Initiator,
		ParentProcessInstanceID: p.ParentProcessInstanceID,
		RootProcessInstanceID:   p.RootProcessInstanceID,
		CreatedAt:             p.CreatedAt,
		UpdatedAt:             p.UpdatedAt,
	}
}

// ToBPMNProcessInstanceListResponse 批量转换
func ToBPMNProcessInstanceListResponse(instances []*ent.ProcessInstance) []*BPMNProcessInstanceResponse {
	if instances == nil {
		return nil
	}
	result := make([]*BPMNProcessInstanceResponse, 0, len(instances))
	for _, p := range instances {
		result = append(result, ToBPMNProcessInstanceResponse(p))
	}
	return result
}

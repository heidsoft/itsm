package dto

import (
	"fmt"
	"time"

	"itsm-backend/ent"
)

// BPMNProcessInstanceResponse BPMN流程实例响应（camelCase JSON，供前端使用）
//
// 字段说明:
//   - ID: 数据库主键 UUID
//   - InstanceID: 流程实例唯一标识（用于外部引用）
//   - BusinessKey: 业务键（如工单号），用于关联业务实体
//   - ProcessDefinitionKey: 流程定义键（如 incident_workflow）
//   - ProcessDefinitionID: 流程定义版本ID
//   - Status: 实例状态（pending/running/completed/terminated/suspended）
//   - CurrentActivityID: 当前活动节点ID
//   - CurrentActivityName: 当前活动节点名称
//   - Variables: 流程变量（JSON对象）
//   - StartTime: 开始时间
//   - EndTime: 结束时间（完成后填充）
//   - SuspendedTime: 挂起时间（暂停时填充）
//   - SuspendedReason: 挂起原因
//   - TenantID: 租户ID（多租户隔离）
//   - Version: 流程版本号
//   - Initiator: 发起人用户名
//   - ParentProcessInstanceID: 父流程实例ID（用于子流程）
//   - RootProcessInstanceID: 根流程实例ID（用于嵌套子流程追溯）
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
		ID:                    fmt.Sprintf("%d", p.ID),
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

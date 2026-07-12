package dto

import (
	"time"

	"itsm-backend/ent"
)

type ProcessApprovalDecisionResponse struct {
	ID                   int                    `json:"id"`
	ProcessInstanceID    int                    `json:"processInstanceId"`
	ProcessInstanceKey   string                 `json:"processInstanceKey"`
	ProcessTaskID        int                    `json:"processTaskId"`
	TaskID               string                 `json:"taskId"`
	ProcessDefinitionKey string                 `json:"processDefinitionKey"`
	NodeKey              string                 `json:"nodeKey"`
	BusinessType         string                 `json:"businessType,omitempty"`
	BusinessID           string                 `json:"businessId,omitempty"`
	ActorID              int                    `json:"actorId"`
	ActorName            string                 `json:"actorName,omitempty"`
	Action               string                 `json:"action"`
	Decision             string                 `json:"decision"`
	Comment              string                 `json:"comment,omitempty"`
	VariablesSnapshot    map[string]interface{} `json:"variablesSnapshot,omitempty"`
	CreatedAt            time.Time              `json:"createdAt"`
}

func ToProcessApprovalDecisionResponse(v *ent.ProcessApprovalDecision) *ProcessApprovalDecisionResponse {
	return &ProcessApprovalDecisionResponse{
		ID: v.ID, ProcessInstanceID: v.ProcessInstanceID, ProcessInstanceKey: v.ProcessInstanceKey,
		ProcessTaskID: v.ProcessTaskID, TaskID: v.TaskID, ProcessDefinitionKey: v.ProcessDefinitionKey,
		NodeKey: v.NodeKey, BusinessType: v.BusinessType, BusinessID: v.BusinessID,
		ActorID: v.ActorID, ActorName: v.ActorName, Action: v.Action, Decision: v.Decision,
		Comment: v.Comment, VariablesSnapshot: v.VariablesSnapshot, CreatedAt: v.CreatedAt,
	}
}

func ToProcessApprovalDecisionResponseList(values []*ent.ProcessApprovalDecision) []*ProcessApprovalDecisionResponse {
	result := make([]*ProcessApprovalDecisionResponse, 0, len(values))
	for _, value := range values {
		result = append(result, ToProcessApprovalDecisionResponse(value))
	}
	return result
}

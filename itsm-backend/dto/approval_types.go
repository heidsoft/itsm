package dto

import "fmt"

// === 审批链强类型枚举（新增，不与已有类型冲突） ===

// ApprovalNodeType 审批人类型枚举
type ApprovalNodeType string

const (
	ApprovalNodeTypeUser           ApprovalNodeType = "user"
	ApprovalNodeTypeRole           ApprovalNodeType = "role"
	ApprovalNodeTypeDepartment     ApprovalNodeType = "department"
	ApprovalNodeTypeDynamic        ApprovalNodeType = "dynamic"
	ApprovalNodeTypeDeptManager    ApprovalNodeType = "dept_manager"
	ApprovalNodeTypeTeamLeader     ApprovalNodeType = "team_leader"
	ApprovalNodeTypeProjectManager ApprovalNodeType = "project_manager"
	ApprovalNodeTypeTempTeamLeader ApprovalNodeType = "temp_team_leader"
	ApprovalNodeTypeAmountBased    ApprovalNodeType = "amount_based"
)

// ApprovalMode 审批模式枚举
type ApprovalMode string

const (
	ApprovalModeSequential ApprovalMode = "sequential"
	ApprovalModeParallel   ApprovalMode = "parallel"
	ApprovalModeAny        ApprovalMode = "any"
	ApprovalModeAll        ApprovalMode = "all"
)

// ApprovalRejectAction 拒绝动作枚举
type ApprovalRejectAction string

const (
	ApprovalRejectActionEnd    ApprovalRejectAction = "end"
	ApprovalRejectActionReturn ApprovalRejectAction = "return"
	ApprovalRejectActionCustom ApprovalRejectAction = "custom"
)

// ApprovalAction 审批操作枚举
type ApprovalAction string

const (
	ApprovalActionApprove  ApprovalAction = "approve"
	ApprovalActionReject   ApprovalAction = "reject"
	ApprovalActionDelegate ApprovalAction = "delegate"
)

// ApprovalConditionOperator 条件操作符枚举
type ApprovalConditionOperator string

const (
	ConditionOpGreaterThan     ApprovalConditionOperator = "greater_than"
	ConditionOpLessThan        ApprovalConditionOperator = "less_than"
	ConditionOpEqual           ApprovalConditionOperator = "equal"
	ConditionOpNotEqual        ApprovalConditionOperator = "not_equal"
	ConditionOpGreaterThanOrEq ApprovalConditionOperator = "greater_than_or_equal"
	ConditionOpLessThanOrEq    ApprovalConditionOperator = "less_than_or_equal"
	ConditionOpContains        ApprovalConditionOperator = "contains"
)

// === 强类型节点配置（取代 map[string]interface{}） ===

// ApprovalNodeConfig 审批节点强类型配置
// 这是存储到数据库的节点结构，取代 map[string]interface{}
// 注意：Conditions 复用已在 ticket_approval_dto.go 中定义的 ApprovalConditionConfig
type ApprovalNodeConfig struct {
	Level            int                       `json:"level"`
	Name             string                    `json:"name"`
	ApproverType     ApprovalNodeType          `json:"approver_type"`
	ApproverIDs      []int                     `json:"approver_ids,omitempty"`
	AssigneeType     string                    `json:"assignee_type,omitempty"`
	AssigneeValue    string                    `json:"assignee_value,omitempty"`
	ApprovalMode     ApprovalMode              `json:"approval_mode"`
	MinimumApprovals *int                      `json:"minimum_approvals,omitempty"`
	TimeoutHours     *int                      `json:"timeout_hours,omitempty"`
	Conditions       []ApprovalConditionConfig `json:"conditions,omitempty"`
	AllowReject      bool                      `json:"allow_reject"`
	AllowDelegate    bool                      `json:"allow_delegate"`
	RejectAction     ApprovalRejectAction      `json:"reject_action"`
	ReturnToLevel    *int                      `json:"return_to_level,omitempty"`
}

// WorkflowListFilter 审批工作流列表过滤条件（强类型，取代 map[string]interface{}）
type WorkflowListFilter struct {
	TicketType string
	Priority   string
	IsActive   *bool
}

// === 转换辅助函数 ===

// NodeRequestToConfig 从 ApprovalNodeRequest 转换为 ApprovalNodeConfig
func NodeRequestToConfig(node ApprovalNodeRequest) ApprovalNodeConfig {
	return ApprovalNodeConfig{
		Level:            node.Level,
		Name:             node.Name,
		ApproverType:     ApprovalNodeType(node.ApproverType),
		ApproverIDs:      node.ApproverIDs,
		AssigneeType:     node.AssigneeType,
		AssigneeValue:    node.AssigneeValue,
		ApprovalMode:     ApprovalMode(node.ApprovalMode),
		MinimumApprovals: node.MinimumApprovals,
		TimeoutHours:     node.TimeoutHours,
		Conditions:       node.Conditions,
		AllowReject:      node.AllowReject,
		AllowDelegate:    node.AllowDelegate,
		RejectAction:     ApprovalRejectAction(node.RejectAction),
		ReturnToLevel:    node.ReturnToLevel,
	}
}

// NodesToConfigs 批量转换 []ApprovalNodeRequest -> []ApprovalNodeConfig
func NodesToConfigs(nodes []ApprovalNodeRequest) []ApprovalNodeConfig {
	result := make([]ApprovalNodeConfig, len(nodes))
	for i, n := range nodes {
		result[i] = NodeRequestToConfig(n)
	}
	return result
}

// ConfigsToResponses 批量转换 []ApprovalNodeConfig -> []ApprovalNodeResponse
func ConfigsToResponses(configs []ApprovalNodeConfig) []ApprovalNodeResponse {
	responses := make([]ApprovalNodeResponse, len(configs))
	for i, c := range configs {
		responses[i] = ApprovalNodeResponse{
			ID:               fmt.Sprintf("node%d", i+1),
			Level:            c.Level,
			Name:             c.Name,
			ApproverType:     string(c.ApproverType),
			ApproverIDs:      c.ApproverIDs,
			AssigneeType:     c.AssigneeType,
			AssigneeValue:    c.AssigneeValue,
			ApprovalMode:     string(c.ApprovalMode),
			MinimumApprovals: c.MinimumApprovals,
			TimeoutHours:     c.TimeoutHours,
			Conditions:       c.Conditions,
			AllowReject:      c.AllowReject,
			AllowDelegate:    c.AllowDelegate,
			RejectAction:     string(c.RejectAction),
			ReturnToLevel:    c.ReturnToLevel,
		}
	}
	return responses
}

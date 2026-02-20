package dto

import "time"

// BusinessType 业务类型枚举
type BusinessType string

const (
	BusinessTypeTicket       BusinessType = "ticket"        // 工单
	BusinessTypeChange      BusinessType = "change"        // 变更
	BusinessTypeIncident    BusinessType = "incident"      // 事件
	BusinessTypeServiceRequest BusinessType = "service_request" // 服务请求
	BusinessTypeProblem     BusinessType = "problem"       // 问题
)

// ProcessStatus 流程状态
type ProcessStatus string

const (
	ProcessStatusPending   ProcessStatus = "pending"    // 待处理
	ProcessStatusRunning   ProcessStatus = "running"    // 运行中
	ProcessStatusCompleted ProcessStatus = "completed" // 已完成
	ProcessStatusSuspended ProcessStatus = "suspended"  // 已暂停
	ProcessStatusTerminated ProcessStatus = "terminated" // 已终止
)

// ProcessTriggerRequest 流程触发请求
type ProcessTriggerRequest struct {
	// 业务标识
	BusinessType BusinessType `json:"business_type" binding:"required"`
	BusinessID   int          `json:"business_id" binding:"required"`

	// 流程定义（可选，如果不传则根据业务类型自动匹配）
	ProcessDefinitionKey string                 `json:"process_definition_key,omitempty"`
	ProcessVersion       int                    `json:"process_version,omitempty"`

	// 流程变量
	Variables map[string]interface{} `json:"variables,omitempty"`

	// 触发者信息
	TriggeredBy    string    `json:"triggered_by"`
	TriggeredAt    time.Time `json:"triggered_at"`
	TenantID       int      `json:"tenant_id"`
}

// ProcessTriggerResponse 流程触发响应
type ProcessTriggerResponse struct {
	// 流程实例信息
	ProcessInstanceID   int          `json:"process_instance_id"`
	ProcessDefinitionKey string      `json:"process_definition_key"`
	ProcessDefinitionName string     `json:"process_definition_name"`
	BusinessKey         string      `json:"business_key"`

	// 状态信息
	Status              ProcessStatus `json:"status"`
	CurrentActivityID   string        `json:"current_activity_id,omitempty"`
	CurrentActivityName string        `json:"current_activity_name,omitempty"`

	// 任务信息
	StartTime          time.Time     `json:"start_time"`
	EndTime            *time.Time    `json:"end_time,omitempty"`

	// 消息
	Message            string        `json:"message,omitempty"`
}

// ProcessBinding 流程绑定配置（用于配置表）
type ProcessBinding struct {
	ID                   int          `json:"id"`
	BusinessType         BusinessType `json:"business_type" binding:"required"`
	BusinessSubType      string       `json:"business_sub_type,omitempty"`      // 子类型（如：ticket的incident/change/problem）
	ProcessDefinitionKey string       `json:"process_definition_key" binding:"required"`
	ProcessVersion       int          `json:"process_version,omitempty"`
	IsDefault            bool         `json:"is_default"`                      // 是否默认流程
	Priority             int          `json:"priority"`                         // 优先级（多个匹配时使用）
	IsActive             bool         `json:"is_active"`
	TenantID             int          `json:"tenant_id"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

// ProcessBindingQueryRequest 查询流程绑定配置请求
type ProcessBindingQueryRequest struct {
	BusinessType    BusinessType `json:"business_type,omitempty"`
	BusinessSubType string       `json:"business_sub_type,omitempty"`
	IsActive        *bool        `json:"is_active,omitempty"`
	TenantID       int          `json:"-"` // 从上下文获取，不从请求参数绑定
}

// BatchProcessBindingRequest 批量绑定请求
type BatchProcessBindingRequest struct {
	Bindings []ProcessBinding `json:"bindings" binding:"required,dive"`
	TenantID int              `json:"tenant_id"`
}

// StandardVariables 标准流程变量
type StandardVariables struct {
	// 业务标识（必填）
	BusinessType     string `json:"business_type"`               // 业务类型
	BusinessID      int    `json:"business_id"`                 // 业务记录ID
	BusinessKey     string `json:"business_key"`                // 业务唯一键（可选）

	// 业务基本信息
	Title           string `json:"title,omitempty"`             // 标题
	Description     string `json:"description,omitempty"`        // 描述
	Priority        string `json:"priority,omitempty"`            // 优先级
	Severity        string `json:"severity,omitempty"`          // 严重程度

	// 申请人/发起人信息
	ApplicantID     string `json:"applicant_id,omitempty"`       // 申请人ID
	ApplicantName   string `json:"applicant_name,omitempty"`    // 申请人名称
	ApplicantDept   string `json:"applicant_dept,omitempty"`    // 申请人部门

	// 处理人信息
	HandlerID       string `json:"handler_id,omitempty"`        // 当前处理人ID
	HandlerName     string `json:"handler_name,omitempty"`     // 当前处理人名称
	AssigneeID      string `json:"assignee_id,omitempty"`      // 受理人ID
	AssigneeName    string `json:"assignee_name,omitempty"`    // 受理人名称

	// 审批信息
	ApproverID      string `json:"approver_id,omitempty"`       // 审批人ID
	ApproverName    string `json:"approver_name,omitempty"`    // 审批人名称
	ApprovalResult  string `json:"approval_result,omitempty"`   // 审批结果

	// 时间相关
	DueDate         string `json:"due_date,omitempty"`         // 截止日期
	EstimatedHours  int    `json:"estimated_hours,omitempty"`  // 预估工时

	// 分类信息
	Category1       string `json:"category_1,omitempty"`       // 一级分类
	Category2       string `json:"category_2,omitempty"`       // 二级分类

	// SLA相关
	SLAID           int    `json:"sla_id,omitempty"`           // SLA ID
	SLAPriority     string `json:"sla_priority,omitempty"`     // SLA优先级
	ResponseDueTime string `json:"response_due_time,omitempty"`// 响应截止时间
	ResolveDueTime  string `json:"resolve_due_time,omitempty"` // 解决截止时间

	// 扩展字段
	CustomFields    map[string]interface{} `json:"custom_fields,omitempty"`
}

// ConvertToMap 将标准变量转换为Map
func (sv *StandardVariables) ConvertToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// 业务标识
	result["business_type"] = sv.BusinessType
	result["business_id"] = sv.BusinessID
	if sv.BusinessKey != "" {
		result["business_key"] = sv.BusinessKey
	}

	// 业务基本信息
	if sv.Title != "" {
		result["title"] = sv.Title
	}
	if sv.Description != "" {
		result["description"] = sv.Description
	}
	if sv.Priority != "" {
		result["priority"] = sv.Priority
	}
	if sv.Severity != "" {
		result["severity"] = sv.Severity
	}

	// 申请人信息
	if sv.ApplicantID != "" {
		result["applicant_id"] = sv.ApplicantID
	}
	if sv.ApplicantName != "" {
		result["applicant_name"] = sv.ApplicantName
	}
	if sv.ApplicantDept != "" {
		result["applicant_dept"] = sv.ApplicantDept
	}

	// 处理人信息
	if sv.HandlerID != "" {
		result["handler_id"] = sv.HandlerID
	}
	if sv.HandlerName != "" {
		result["handler_name"] = sv.HandlerName
	}
	if sv.AssigneeID != "" {
		result["assignee_id"] = sv.AssigneeID
	}
	if sv.AssigneeName != "" {
		result["assignee_name"] = sv.AssigneeName
	}

	// 审批信息
	if sv.ApproverID != "" {
		result["approver_id"] = sv.ApproverID
	}
	if sv.ApproverName != "" {
		result["approver_name"] = sv.ApproverName
	}
	if sv.ApprovalResult != "" {
		result["approval_result"] = sv.ApprovalResult
	}

	// 时间相关
	if sv.DueDate != "" {
		result["due_date"] = sv.DueDate
	}
	if sv.EstimatedHours > 0 {
		result["estimated_hours"] = sv.EstimatedHours
	}

	// 分类信息
	if sv.Category1 != "" {
		result["category_1"] = sv.Category1
	}
	if sv.Category2 != "" {
		result["category_2"] = sv.Category2
	}

	// SLA相关
	if sv.SLAID > 0 {
		result["sla_id"] = sv.SLAID
	}
	if sv.SLAPriority != "" {
		result["sla_priority"] = sv.SLAPriority
	}
	if sv.ResponseDueTime != "" {
		result["response_due_time"] = sv.ResponseDueTime
	}
	if sv.ResolveDueTime != "" {
		result["resolve_due_time"] = sv.ResolveDueTime
	}

	// 扩展字段
	if sv.CustomFields != nil {
		for k, v := range sv.CustomFields {
			result[k] = v
		}
	}

	return result
}

// ServiceTaskResult 服务任务执行结果
type ServiceTaskResult struct {
	Success      bool                   `json:"success"`
	Message      string                 `json:"message,omitempty"`
	OutputVars   map[string]interface{} `json:"output_vars,omitempty"`
	UpdatedData  map[string]interface{} `json:"updated_data,omitempty"`
}

// CallbackRequest 流程回调请求
type CallbackRequest struct {
	ProcessInstanceID int                    `json:"process_instance_id" binding:"required"`
	ActivityID       string                 `json:"activity_id" binding:"required"`
	ActivityType     string                 `json:"activity_type" binding:"required"` // service_task/script_task
	Result           ServiceTaskResult      `json:"result"`
	ExecutedBy       string                 `json:"executed_by"`
	ExecutedAt       time.Time              `json:"executed_at"`
}

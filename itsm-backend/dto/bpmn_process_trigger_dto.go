package dto

import "time"

// BusinessType 业务类型枚举
type BusinessType string

const (
	BusinessTypeTicket         BusinessType = "ticket"          // 工单
	BusinessTypeChange         BusinessType = "change"          // 变更
	BusinessTypeIncident       BusinessType = "incident"        // 事件
	BusinessTypeServiceRequest BusinessType = "service_request" // 服务请求
	BusinessTypeProblem        BusinessType = "problem"         // 问题
	BusinessTypeRelease        BusinessType = "release"         // 发布
)

// ProcessStatus 流程状态
type ProcessStatus string

const (
	ProcessStatusPending    ProcessStatus = "pending"    // 待处理
	ProcessStatusRunning    ProcessStatus = "running"    // 运行中
	ProcessStatusCompleted  ProcessStatus = "completed"  // 已完成
	ProcessStatusSuspended  ProcessStatus = "suspended"  // 已暂停
	ProcessStatusTerminated ProcessStatus = "terminated" // 已终止
)

// ProcessTriggerRequest 流程触发请求
type ProcessTriggerRequest struct {
	// 业务标识
	BusinessType    BusinessType `json:"businessType" binding:"required"`
	BusinessSubType string       `json:"businessSubType,omitempty"`
	BusinessID      int          `json:"businessId" binding:"required"`

	// 多维流程路由上下文
	DepartmentID int    `json:"departmentId,omitempty"`
	TeamID       int    `json:"teamId,omitempty"`
	ProjectID    int    `json:"projectId,omitempty"`
	Scenario     string `json:"scenario,omitempty"`
	Category     string `json:"category,omitempty"`

	// 流程定义（可选，如果不传则根据业务类型自动匹配）
	ProcessDefinitionKey string `json:"processDefinitionKey,omitempty"`
	ProcessVersion       int    `json:"processVersion,omitempty"`

	// 流程变量
	Variables map[string]interface{} `json:"variables,omitempty"`

	// 触发者信息
	TriggeredBy string    `json:"triggeredBy"`
	TriggeredAt time.Time `json:"triggeredAt"`
	TenantID    int       `json:"tenantId"`
}

// ProcessTriggerResponse 流程触发响应
type ProcessTriggerResponse struct {
	// 流程实例信息
	ProcessInstanceID     int    `json:"processInstanceId"`
	ProcessDefinitionKey  string `json:"processDefinitionKey"`
	ProcessDefinitionName string `json:"processDefinitionName"`
	BusinessKey           string `json:"businessKey"`

	// 状态信息
	Status              ProcessStatus `json:"status"`
	CurrentActivityID   string        `json:"currentActivityId,omitempty"`
	CurrentActivityName string        `json:"currentActivityName,omitempty"`

	// 任务信息
	StartTime time.Time  `json:"startTime"`
	EndTime   *time.Time `json:"endTime,omitempty"`

	// 消息
	Message string `json:"message,omitempty"`
}

// ProcessBinding 流程绑定配置（用于配置表）
type ProcessBinding struct {
	ID                   int                    `json:"id"`
	BusinessType         BusinessType           `json:"businessType" binding:"required"`
	BusinessSubType      string                 `json:"businessSubType,omitempty"` // 子类型（如：ticket的incident/change/problem）
	ProcessDefinitionKey string                 `json:"processDefinitionKey" binding:"required"`
	ProcessVersion       int                    `json:"processVersion,omitempty"`
	IsDefault            bool                   `json:"isDefault"` // 是否默认流程
	Priority             int                    `json:"priority"`  // 优先级（多个匹配时使用）
	IsActive             bool                   `json:"isActive"`
	DepartmentID         int                    `json:"departmentId"`
	TeamID               int                    `json:"teamId"`
	Scenario             string                 `json:"scenario,omitempty"`
	Category             string                 `json:"category,omitempty"`
	Conditions           map[string]interface{} `json:"conditions,omitempty"`
	ApprovalChainID      string                 `json:"approvalChainId,omitempty"`
	SLAPolicyID          string                 `json:"slaPolicyId,omitempty"`
	Overrides            map[string]interface{} `json:"overrides,omitempty"`
	TenantID             int                    `json:"tenantId"`
	CreatedAt            time.Time              `json:"createdAt"`
	UpdatedAt            time.Time              `json:"updatedAt"`
}

// ProcessBindingQueryRequest 查询流程绑定配置请求
type ProcessBindingQueryRequest struct {
	BusinessType    BusinessType `form:"business_type" json:"businessType,omitempty"`
	BusinessSubType string       `form:"business_sub_type" json:"businessSubType,omitempty"`
	DepartmentID    int          `form:"department_id" json:"departmentId,omitempty"`
	TeamID          int          `form:"team_id" json:"teamId,omitempty"`
	Scenario        string       `form:"scenario" json:"scenario,omitempty"`
	Category        string       `form:"category" json:"category,omitempty"`
	IsActive        *bool        `form:"is_active" json:"isActive,omitempty"`
	TenantID        int          `json:"-"` // 从上下文获取，不从请求参数绑定
}

// BatchProcessBindingRequest 批量绑定请求
type BatchProcessBindingRequest struct {
	Bindings []ProcessBinding `json:"bindings" binding:"required,dive"`
	TenantID int              `json:"tenantId"`
}

// StandardVariables 标准流程变量
type StandardVariables struct {
	// 业务标识（必填）
	BusinessType string `json:"businessType"` // 业务类型
	BusinessID   int    `json:"businessId"`   // 业务记录ID
	BusinessKey  string `json:"businessKey"`  // 业务唯一键（可选）

	// 业务基本信息
	Title       string `json:"title,omitempty"`       // 标题
	Description string `json:"description,omitempty"` // 描述
	Priority    string `json:"priority,omitempty"`    // 优先级
	Severity    string `json:"severity,omitempty"`    // 严重程度

	// 申请人/发起人信息
	ApplicantID   string `json:"applicantId,omitempty"`   // 申请人ID
	ApplicantName string `json:"applicantName,omitempty"` // 申请人名称
	ApplicantDept string `json:"applicantDept,omitempty"` // 申请人部门

	// 处理人信息
	HandlerID    string `json:"handlerId,omitempty"`    // 当前处理人ID
	HandlerName  string `json:"handlerName,omitempty"`  // 当前处理人名称
	AssigneeID   string `json:"assigneeId,omitempty"`   // 受理人ID
	AssigneeName string `json:"assigneeName,omitempty"` // 受理人名称

	// 审批信息
	ApproverID     string `json:"approverId,omitempty"`     // 审批人ID
	ApproverName   string `json:"approverName,omitempty"`   // 审批人名称
	ApprovalResult string `json:"approvalResult,omitempty"` // 审批结果

	// 时间相关
	DueDate        string `json:"dueDate,omitempty"`        // 截止日期
	EstimatedHours int    `json:"estimatedHours,omitempty"` // 预估工时

	// 分类信息
	Category1 string `json:"category1,omitempty"` // 一级分类
	Category2 string `json:"category2,omitempty"` // 二级分类

	// SLA相关
	SLAID           int    `json:"slaId,omitempty"`           // SLA ID
	SLAPriority     string `json:"slaPriority,omitempty"`     // SLA优先级
	ResponseDueTime string `json:"responseDueTime,omitempty"` // 响应截止时间
	ResolveDueTime  string `json:"resolveDueTime,omitempty"`  // 解决截止时间

	// 扩展字段
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
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
	Success     bool                   `json:"success"`
	Message     string                 `json:"message,omitempty"`
	OutputVars  map[string]interface{} `json:"outputVars,omitempty"`
	UpdatedData map[string]interface{} `json:"updatedData,omitempty"`
}

// CallbackRequest 流程回调请求
type CallbackRequest struct {
	ProcessInstanceID int               `json:"processInstanceId" binding:"required"`
	ActivityID        string            `json:"activityId" binding:"required"`
	ActivityType      string            `json:"activityType" binding:"required"` // service_task/script_task
	Result            ServiceTaskResult `json:"result"`
	ExecutedBy        string            `json:"executedBy"`
	ExecutedAt        time.Time         `json:"executedAt"`
}

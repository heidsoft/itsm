package dto

import "time"

// TicketTypeStatus 工单类型状态
type TicketTypeStatus string

const (
	TicketTypeStatusActive   TicketTypeStatus = "active"
	TicketTypeStatusInactive TicketTypeStatus = "inactive"
	TicketTypeStatusDraft    TicketTypeStatus = "draft"
)

// CustomFieldType 自定义字段类型
type CustomFieldType string

const (
	CustomFieldTypeText             CustomFieldType = "text"
	CustomFieldTypeTextarea         CustomFieldType = "textarea"
	CustomFieldTypeNumber           CustomFieldType = "number"
	CustomFieldTypeDate             CustomFieldType = "date"
	CustomFieldTypeDatetime         CustomFieldType = "datetime"
	CustomFieldTypeSelect           CustomFieldType = "select"
	CustomFieldTypeMultiSelect      CustomFieldType = "multi_select"
	CustomFieldTypeCheckbox         CustomFieldType = "checkbox"
	CustomFieldTypeRadio            CustomFieldType = "radio"
	CustomFieldTypeFile             CustomFieldType = "file"
	CustomFieldTypeUserPicker       CustomFieldType = "user_picker"
	CustomFieldTypeDepartmentPicker CustomFieldType = "department_picker"
)

// CustomFieldOption 字段选项
type CustomFieldOption struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// CustomFieldValidation 字段验证规则
type CustomFieldValidation struct {
	Min     *int   `json:"min,omitempty"`
	Max     *int   `json:"max,omitempty"`
	Pattern string `json:"pattern,omitempty"`
	Message string `json:"message,omitempty"`
}

// CustomFieldConditionalDisplay 条件显示
type CustomFieldConditionalDisplay struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // equals, not_equals, contains
	Value    interface{} `json:"value"`
}

// CustomFieldDefinition 自定义字段定义
type CustomFieldDefinition struct {
	ID                 string                         `json:"id"`
	Name               string                         `json:"name"`
	Label              string                         `json:"label"`
	Type               CustomFieldType                `json:"type"`
	Required           bool                           `json:"required"`
	Description        string                         `json:"description,omitempty"`
	Placeholder        string                         `json:"placeholder,omitempty"`
	DefaultValue       interface{}                    `json:"defaultValue,omitempty"`
	Options            []CustomFieldOption            `json:"options,omitempty"`
	Validation         *CustomFieldValidation         `json:"validation,omitempty"`
	ConditionalDisplay *CustomFieldConditionalDisplay `json:"conditionalDisplay,omitempty"`
	Order              int                            `json:"order"`
}

// ApproverInfo 审批人信息
type ApproverInfo struct {
	Type  string      `json:"type"`  // user, role, department, dynamic
	Value interface{} `json:"value"` // 用户ID、角色名、部门ID等
	Name  string      `json:"name"`
}

// ApprovalCondition 审批条件
type ApprovalCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // equals, not_equals, greater_than, less_than
	Value    interface{} `json:"value"`
}

// ApprovalChainDefinition 审批链定义
type ApprovalChainDefinition struct {
	ID               string              `json:"id"`
	Level            int                 `json:"level"`
	Name             string              `json:"name"`
	Approvers        []ApproverInfo      `json:"approvers"`
	ApprovalType     string              `json:"approvalType"` // any, all, majority
	MinimumApprovals *int                `json:"minimumApprovals,omitempty"`
	AllowReject      bool                `json:"allowReject"`
	AllowDelegate    bool                `json:"allowDelegate"`
	RejectAction     string              `json:"rejectAction"` // end, return, custom
	ReturnToLevel    *int                `json:"returnToLevel,omitempty"`
	Conditions       []ApprovalCondition `json:"conditions,omitempty"`
	Timeout          *int                `json:"timeout,omitempty"`        // 超时时间（小时）
	TimeoutAction    string              `json:"timeoutAction,omitempty"`  // auto_approve, auto_reject, escalate
}

// AssignToConfig 分配目标配置
type AssignToConfig struct {
	Type  string      `json:"type"`  // user, role, department, round_robin, load_balance
	Value interface{} `json:"value"` // 用户ID、角色名、部门ID等
}

// AssignmentRuleCondition 分配规则条件
type AssignmentRuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // equals, not_equals, contains, greater_than, less_than
	Value    interface{} `json:"value"`
}

// AssignmentRule 分配规则
type AssignmentRule struct {
	ID         string                    `json:"id"`
	Name       string                    `json:"name"`
	Priority   int                       `json:"priority"`
	Conditions []AssignmentRuleCondition `json:"conditions"`
	AssignTo   AssignToConfig            `json:"assignTo"`
	Enabled    bool                      `json:"enabled"`
}

// NotificationRecipients 通知接收人配置
type NotificationRecipients struct {
	Enabled    bool     `json:"enabled"`
	Recipients []string `json:"recipients"` // requester, assignee, approvers, watchers等
	Template   string   `json:"template,omitempty"`
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	OnCreate   NotificationRecipients `json:"onCreate"`
	OnUpdate   NotificationRecipients `json:"onUpdate"`
	OnApproval NotificationRecipients `json:"onApproval"`
	OnReject   NotificationRecipients `json:"onReject"`
	OnComplete NotificationRecipients `json:"onComplete"`
}

// PermissionConfig 权限配置
type PermissionConfig struct {
	CanCreate  []string `json:"canCreate"`
	CanView    []string `json:"canView"`
	CanEdit    []string `json:"canEdit"`
	CanDelete  []string `json:"canDelete"`
	CanApprove []string `json:"canApprove"`
}

// TicketTypeDefinition 工单类型定义
type TicketTypeDefinition struct {
	ID                 int                       `json:"id"`
	Code               string                    `json:"code"`
	Name               string                    `json:"name"`
	Description        string                    `json:"description,omitempty"`
	Icon               string                    `json:"icon,omitempty"`
	Color              string                    `json:"color,omitempty"`
	Status             TicketTypeStatus          `json:"status"`
	CustomFields       []CustomFieldDefinition   `json:"customFields"`
	ApprovalEnabled    bool                      `json:"approvalEnabled"`
	ApprovalWorkflowID *string                   `json:"approvalWorkflowId,omitempty"`
	ApprovalChain      []ApprovalChainDefinition `json:"approvalChain,omitempty"`
	SLAEnabled         bool                      `json:"slaEnabled"`
	DefaultSLAID       *int                      `json:"defaultSlaId,omitempty"`
	AutoAssignEnabled  bool                      `json:"autoAssignEnabled"`
	AssignmentRules    []AssignmentRule          `json:"assignmentRules,omitempty"`
	NotificationConfig *NotificationConfig       `json:"notificationConfig,omitempty"`
	PermissionConfig   *PermissionConfig         `json:"permissionConfig,omitempty"`
	CreatedBy          int                       `json:"createdBy"`
	CreatedByName      string                    `json:"createdByName"`
	UpdatedBy          *int                      `json:"updatedBy,omitempty"`
	UpdatedByName      *string                   `json:"updatedByName,omitempty"`
	CreatedAt          time.Time                 `json:"createdAt"`
	UpdatedAt          time.Time                 `json:"updatedAt"`
	TenantID           int                       `json:"tenantId"`
	UsageCount         int                       `json:"usageCount,omitempty"`
}

// CreateTicketTypeRequest 创建工单类型请求
type CreateTicketTypeRequest struct {
	Code               string                    `json:"code" binding:"required"`
	Name               string                    `json:"name" binding:"required"`
	Description        string                    `json:"description"`
	Icon               string                    `json:"icon"`
	Color              string                    `json:"color"`
	CustomFields       []CustomFieldDefinition   `json:"custom_fields"`
	ApprovalEnabled    bool                      `json:"approval_enabled"`
	ApprovalChain      []ApprovalChainDefinition `json:"approval_chain,omitempty"`
	SLAEnabled         bool                      `json:"sla_enabled"`
	DefaultSLAID       *int                      `json:"default_sla_id,omitempty"`
	AutoAssignEnabled  bool                      `json:"auto_assign_enabled"`
	AssignmentRules    []AssignmentRule          `json:"assignment_rules,omitempty"`
	NotificationConfig *NotificationConfig       `json:"notification_config,omitempty"`
	PermissionConfig   *PermissionConfig         `json:"permission_config,omitempty"`
}

// UpdateTicketTypeRequest 更新工单类型请求
type UpdateTicketTypeRequest struct {
	Name               *string                    `json:"name"`
	Description        *string                    `json:"description"`
	Icon               *string                    `json:"icon"`
	Color              *string                    `json:"color"`
	Status             *TicketTypeStatus          `json:"status"`
	CustomFields       *[]CustomFieldDefinition   `json:"custom_fields"`
	ApprovalEnabled    *bool                      `json:"approval_enabled"`
	ApprovalChain      *[]ApprovalChainDefinition `json:"approval_chain"`
	SLAEnabled         *bool                      `json:"sla_enabled"`
	DefaultSLAID       *int                       `json:"default_sla_id"`
	AutoAssignEnabled  *bool                      `json:"auto_assign_enabled"`
	AssignmentRules    *[]AssignmentRule          `json:"assignment_rules"`
	NotificationConfig *NotificationConfig        `json:"notification_config"`
	PermissionConfig   *PermissionConfig          `json:"permission_config"`
}

// ListTicketTypesRequest 查询工单类型列表请求
type ListTicketTypesRequest struct {
	Status   *TicketTypeStatus `form:"status"`
	Keyword  string            `form:"keyword"`
	Page     int               `form:"page" binding:"min=1"`
	PageSize int               `form:"page_size" binding:"min=1,max=100"`
}

// TicketTypeListResponse 工单类型列表响应
type TicketTypeListResponse struct {
	Types      []TicketTypeDefinition `json:"types"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"pageSize"`
	TotalPages int                    `json:"totalPages"`
}

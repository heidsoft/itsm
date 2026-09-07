package dto

// GenerateBPMNRequest AI生成BPMN流程请求
type GenerateBPMNRequest struct {
	// 业务需求描述
	Requirement string `json:"requirement" binding:"required,min=10,max=2000"`
	// 流程类型：incident/change/problem/service_request/custom
	ProcessType string `json:"processType" binding:"required,oneof=incident change problem service_request leave expense hr procurement it custom"`
	// 企业类型：cn_enterprise/international/startup/government
	EnterpriseType string `json:"enterpriseType" binding:"required,oneof=cn_enterprise international startup government"`
	// 是否包含SLA配置
	IncludeSLA bool `json:"includeSla"`
	// 是否包含通知配置
	IncludeNotifications bool `json:"includeNotifications"`
	// 是否包含审批节点
	IncludeApprovals bool `json:"includeApprovals"`
	// 租户ID
	// TenantID is populated from the authenticated tenant context by the handler.
	TenantID int `json:"tenantId,omitempty"`
}

// GenerateBPMNResponse AI生成BPMN流程响应
type GenerateBPMNResponse struct {
	// 生成的BPMN XML内容
	BPMNXML string `json:"bpmnXml"`
	// 流程ID
	ProcessID string `json:"processId"`
	// 流程名称
	ProcessName string `json:"processName"`
	// 流程描述
	ProcessDescription string `json:"processDescription"`
	// 版本号
	Version string `json:"version"`
	// 节点数量
	NodeCount int `json:"nodeCount"`
	// 预估复杂度：low/medium/high
	Complexity string `json:"complexity"`
	// 生成说明
	Explanation string `json:"explanation"`
	// 部署后的流程定义ID（如果选择自动部署）
	DeploymentID string `json:"deploymentId,omitempty"`
	// 流程定义ID
	ProcessDefinitionID int `json:"processDefinitionId,omitempty"`
	// 生成后自动 Lint 的结果（结构/连通性/网关语义检查；HasErrors 时不应部署）
	LintResult          *BPMNLintResult           `json:"lintResult,omitempty"`
	CandidateDefinition *BusinessProcessCandidate `json:"candidateDefinition,omitempty"`
}

// BusinessProcessCandidate is the structured contract edited in the existing
// workflow designer before a BPMN definition can be published.
type BusinessProcessCandidate struct {
	Domain               string                 `json:"domain"`
	FormSchema           map[string]interface{} `json:"formSchema"`
	ApprovalPolicy       map[string]interface{} `json:"approvalPolicy"`
	OntologyBindings     map[string]interface{} `json:"ontologyBindings"`
	SLAConfig            map[string]interface{} `json:"slaConfig,omitempty"`
	RequiresConfirmation bool                   `json:"requiresConfirmation"`
}

// PreviewBPMNRequest 预览生成的BPMN流程请求
type PreviewBPMNRequest struct {
	// 业务需求描述
	Requirement string `json:"requirement" binding:"required,min=10,max=2000"`
	// 流程类型
	ProcessType string `json:"processType" binding:"required,oneof=incident change problem service_request leave expense hr procurement it custom"`
	// 企业类型
	EnterpriseType string `json:"enterpriseType" binding:"required,oneof=cn_enterprise international startup government"`
}

// PreviewBPMNResponse 预览生成的BPMN流程响应
type PreviewBPMNResponse struct {
	// 流程结构描述
	StructureDescription string `json:"structureDescription"`
	// 节点列表
	Nodes []BPMNNodePreview `json:"nodes"`
	// 预估复杂度
	Complexity string `json:"complexity"`
	// 预估节点数量
	EstimatedNodeCount int `json:"estimatedNodeCount"`
	// 适用场景说明
	UseCases string `json:"useCases"`
	// 优化建议
	Suggestions []string `json:"suggestions"`
}

// BPMNNodePreview BPMN节点预览
type BPMNNodePreview struct {
	// 节点ID
	ID string `json:"id"`
	// 节点名称
	Name string `json:"name"`
	// 节点类型：startEvent/endEvent/userTask/serviceTask/exclusiveGateway/parallelGateway
	Type string `json:"type"`
	// 节点描述
	Description string `json:"description"`
	// 处理人角色
	AssigneeRole string `json:"assigneeRole,omitempty"`
	// SLA时间（分钟）
	SLAMinutes int `json:"slaMinutes,omitempty"`
}

// BPMNTemplateSuggestion is the stable contract returned by template search.
type BPMNTemplateSuggestion struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	ProcessType string  `json:"processType"`
	Score       float64 `json:"score"`
}

type WorkflowTemplate struct {
	ID               int                    `json:"id"`
	Key              string                 `json:"key"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Domain           string                 `json:"domain"`
	FormSchema       map[string]interface{} `json:"formSchema"`
	ApprovalPolicy   map[string]interface{} `json:"approvalPolicy"`
	OntologyBindings map[string]interface{} `json:"ontologyBindings"`
	SLAConfig        map[string]interface{} `json:"slaConfig"`
	BPMNXML          string                 `json:"bpmnXml,omitempty"`
	Version          string                 `json:"version"`
	Status           string                 `json:"status"`
	IsPublic         bool                   `json:"isPublic"`
	CreatedBy        int                    `json:"createdBy"`
	CreatedAt        string                 `json:"createdAt"`
	UpdatedAt        string                 `json:"updatedAt"`
}

type WorkflowTemplateListResponse struct {
	Items      []*WorkflowTemplate `json:"items"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"pageSize"`
	TotalPages int                 `json:"totalPages"`
}

type CreateWorkflowTemplateRequest struct {
	Key              string                 `json:"key" binding:"required,min=2,max=120"`
	Name             string                 `json:"name" binding:"required,max=200"`
	Description      string                 `json:"description"`
	Domain           string                 `json:"domain" binding:"required,max=40"`
	FormSchema       map[string]interface{} `json:"formSchema"`
	ApprovalPolicy   map[string]interface{} `json:"approvalPolicy"`
	OntologyBindings map[string]interface{} `json:"ontologyBindings"`
	SLAConfig        map[string]interface{} `json:"slaConfig"`
	BPMNXML          string                 `json:"bpmnXml" binding:"required"`
	IsPublic         bool                   `json:"isPublic"`
}

type UpdateWorkflowTemplateRequest struct {
	Name             *string                 `json:"name"`
	Description      *string                 `json:"description"`
	Domain           *string                 `json:"domain"`
	FormSchema       *map[string]interface{} `json:"formSchema"`
	ApprovalPolicy   *map[string]interface{} `json:"approvalPolicy"`
	OntologyBindings *map[string]interface{} `json:"ontologyBindings"`
	SLAConfig        *map[string]interface{} `json:"slaConfig"`
	BPMNXML          *string                 `json:"bpmnXml"`
	IsPublic         *bool                   `json:"isPublic"`
}

// ReloadWorkflowTemplateResponse 重新部署已发布模板为 BPMN 流程定义的响应。
// PreviousVersion 为空表示该 key 在 process_definitions 中尚无历史部署记录。
// Source 标识重载源（当前固定为 workflow_templates，便于后续扩展内置模板）。
type ReloadWorkflowTemplateResponse struct {
	Key                string `json:"key"`
	Name               string `json:"name"`
	PreviousVersion    string `json:"previousVersion,omitempty"`
	NewVersion         string `json:"newVersion"`
	DeploymentID       string `json:"deploymentId"`
	ProcessDefinitionID int   `json:"processDefinitionId"`
	Source             string `json:"source"`
	ReloadedAt         string `json:"reloadedAt"`
}

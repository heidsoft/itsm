package dto

// GenerateBPMNRequest AI生成BPMN流程请求
type GenerateBPMNRequest struct {
	// 业务需求描述
	Requirement string `json:"requirement" binding:"required,min=10,max=2000"`
	// 流程类型：incident/change/problem/service_request/custom
	ProcessType string `json:"processType" binding:"required,oneof=incident change problem service_request custom"`
	// 企业类型：cn_enterprise/international/startup/government
	EnterpriseType string `json:"enterpriseType" binding:"required,oneof=cn_enterprise international startup government"`
	// 是否包含SLA配置
	IncludeSLA bool `json:"includeSla"`
	// 是否包含通知配置
	IncludeNotifications bool `json:"includeNotifications"`
	// 是否包含审批节点
	IncludeApprovals bool `json:"includeApprovals"`
	// 租户ID
	TenantID int `json:"tenantId" binding:"required"`
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
}

// PreviewBPMNRequest 预览生成的BPMN流程请求
type PreviewBPMNRequest struct {
	// 业务需求描述
	Requirement string `json:"requirement" binding:"required,min=10,max=2000"`
	// 流程类型
	ProcessType string `json:"processType" binding:"required,oneof=incident change problem service_request custom"`
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

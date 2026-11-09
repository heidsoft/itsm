package dto

// LegacyApprovalMigrationResult 是旧审批工作流迁移到 BPMN 的结果视图。
// 该类型是纯响应 DTO，放在 dto 包以便 HTTP 域声明窄契约时无需回指共享层；
// service 包保留同名别名以维持既有引用与 API 兼容。
type LegacyApprovalMigrationResult struct {
	WorkflowID           int    `json:"workflowId"`
	ProcessDefinitionKey string `json:"processDefinitionKey"`
	BPMNXML              string `json:"bpmnXml,omitempty"`
	Skipped              bool   `json:"skipped"`
	BindingReplaced      bool   `json:"bindingReplaced"`
}

package dto

// BPMNLintResult BPMN Lint 校验结果
type BPMNLintResult struct {
	// 是否存在 error 级问题（true 时不应部署）
	HasErrors bool `json:"hasErrors"`
	// error 级问题数
	ErrorCount int `json:"errorCount"`
	// warning 级问题数
	WarningCount int `json:"warningCount"`
	// 问题列表（按出现顺序）
	Issues []*BPMNLintIssue `json:"issues"`
}

// BPMNLintIssue 单条 Lint 问题
type BPMNLintIssue struct {
	// 严重级别：error / warning
	Severity string `json:"severity"`
	// 规则类别：structure / events / tasks / connectivity / gateways / flows
	Category string `json:"category"`
	// 关联节点 ID（可空）
	ElementID string `json:"elementId,omitempty"`
	// 关联节点名称（可空）
	ElementName string `json:"elementName,omitempty"`
	// 问题描述（中文，可直接展示给用户）
	Message string `json:"message"`
}

// BPMNLintRequest BPMN Lint 请求
type BPMNLintRequest struct {
	// BPMN XML 内容
	BPMNXML string `json:"bpmnXml" binding:"required"`
}

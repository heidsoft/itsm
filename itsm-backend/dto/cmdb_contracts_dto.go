package dto

// CMDB 生产服务窄契约签名闭包里的 2 个值类型与 1 个 sentinel。
// 2026-11-10 由 service/{configuration_item_service,tool_registry,ci_relationship_service}.go
// 下沉；service 侧保留同名别名，字段/JSON tag/errors.Is 身份不变。

import "errors"

// ErrInvalidRelationshipType 关系类型不在受控词表内（词表单一源见 ent/schema/ci_relationship.go）。
// handler 通过 errors.Is 将其映射为 400 参数错误，而不是 500。
var ErrInvalidRelationshipType = errors.New("invalid CI relationship type")

type LifecycleChangeParams struct {
	Status           string
	Remark           string
	RetirementReason string
	// ExpectedVersion > 0 时启用乐观锁，不匹配返回 VersionConflictError；
	// 批量路径传 0，跳过版本校验。
	ExpectedVersion int
}

type ToolDefinition struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	ReadOnly     bool                   `json:"readOnly"`
	Resource     string                 `json:"resource"`
	Action       string                 `json:"action"`
	ArgsSchema   map[string]interface{} `json:"argsSchema"`
	ResultSchema map[string]interface{} `json:"resultSchema"`
}

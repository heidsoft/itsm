package dto

// CMDB 生产服务窄契约签名闭包里的 2 个值类型与 1 个 sentinel。
// 2026-11-10 由 service/{configuration_item_service,tool_registry,ci_relationship_service}.go
// 下沉；service 侧保留同名别名，字段/JSON tag/errors.Is 身份不变。

import "errors"

// ErrInvalidRelationshipType 关系类型不在受控词表内（词表单一源见 ent/schema/ci_relationship.go）。
// handler 通过 errors.Is 将其映射为 400 参数错误，而不是 500。
var ErrInvalidRelationshipType = errors.New("invalid CI relationship type")

// ErrInvalidRelationshipGrade 关系的 strength / impact_level 不在四级词表内。
//
// 这两个字段是 ent Enum（critical/high/medium/low，见 ent/schema/ci_relationship.go:152,157），
// 非法值若直落 DB 会撞 check constraint，报出的是 500「服务端故障」——调用方（含 AI Agent
// 写工具 create_ci_relationship）无法区分「我传错了」与「后端挂了」，也白白做完了
// 存在性查询与环检测。与关系类型同理，在 service 层做 fail-fast 校验，映射 400。
var ErrInvalidRelationshipGrade = errors.New("invalid CI relationship strength/impact level")

// IsValidRelationshipStrength 判断关系强度是否在四级词表内（单一源：enum Values）。
func IsValidRelationshipStrength(v RelationshipStrength) bool {
	switch v {
	case StrengthCritical, StrengthHigh, StrengthMedium, StrengthLow:
		return true
	}
	return false
}

// IsValidImpactLevel 判断影响程度是否在四级词表内（单一源：enum Values）。
func IsValidImpactLevel(v ImpactLevel) bool {
	switch v {
	case ImpactCritical, ImpactHigh, ImpactMedium, ImpactLow:
		return true
	}
	return false
}

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

package dto

// CMDB 本体自描述（ontology introspection）DTO。
// 用途：LLM Agent 通过 GET /api/v1/cmdb/ontology 一次性拿到
// 「CI 类型 × 属性定义 × 关系词表 × 枚举值域 × 可用 AI 工具」，
// 形成 introspect → 查询 → 操作 → 验证 的闭环（2026-09-04 CMDB AI-Native 评审 P0-1）。

// CMDBOntologyResponse 本体自描述响应
type CMDBOntologyResponse struct {
	Version           string                          `json:"version"`
	CITypes           []CMDBOntologyCIType            `json:"ciTypes"`
	RelationshipTypes []CMDBOntologyRelationshipType  `json:"relationshipTypes"`
	Enums             CMDBOntologyEnums               `json:"enums"`
	AITools           []CMDBOntologyTool              `json:"aiTools"`
}

// CMDBOntologyCIType 单个 CI 类型的本体描述
type CMDBOntologyCIType struct {
	ID                   int                             `json:"id"`
	Name                 string                          `json:"name"`
	Description          string                          `json:"description"`
	Icon                 string                          `json:"icon"`
	Color                string                          `json:"color"`
	ParentTypeID         *int                            `json:"parentTypeId,omitempty"`
	// AttributeSchema 为 CIType 表中的 schema 文本：合法 JSON 时输出为 JSON 对象，
	// 否则原样输出字符串（保持透明，便于诊断脏数据）。
	AttributeSchema      interface{}                      `json:"attributeSchema,omitempty"`
	AttributeDefinitions []*CIAttributeDefinitionResponse `json:"attributeDefinitions"`
}

// CMDBOntologyRelationshipType 关系词表条目（从 ent/schema 单一受控源派生）
type CMDBOntologyRelationshipType struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Direction   string `json:"direction"` // uni-directional / bi-directional
	Icon        string `json:"icon"`
	ReverseType string `json:"reverseType"` // 语义反向类型
}

// CMDBOntologyEnums 受控枚举值域
type CMDBOntologyEnums struct {
	CIStatus        []string `json:"ciStatus"`
	Environment     []string `json:"environment"`
	Criticality     []string `json:"criticality"`
	LifecycleStatus []string `json:"lifecycleStatus"`
	OwnershipMode   []string `json:"ownershipMode"`
	Strength        []string `json:"relationshipStrength"`
	ImpactLevel     []string `json:"impactLevel"`
}

// CMDBOntologyTool AI 工具描述（与 service.ToolDefinition 对齐；dto 不能反向依赖 service，故独立定义）
type CMDBOntologyTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	ReadOnly    bool                   `json:"readOnly"`
	Resource    string                 `json:"resource"`
	Action      string                 `json:"action"`
	ArgsSchema  map[string]interface{} `json:"argsSchema"`
}

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ApprovalChain holds the schema definition for the ApprovalChain entity.
type ApprovalChain struct {
	ent.Schema
}

// Fields of the ApprovalChain.
func (ApprovalChain) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.String("description").Optional(),
		field.String("entity_type").Default("ticket"), // ticket, change, service_request
		field.JSON("chain", []ApprovalChainStep{}),
		field.String("status").Default("active"), // active, inactive
		field.Int("created_by").Optional(),
		field.Int("tenant_id").Optional(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// ApprovalChainStep 审批链步骤
type ApprovalChainStep struct {
	Level        int    `json:"level"`
	ApproverID   int    `json:"approver_id,omitempty"`
	Role         string `json:"role"`                    // manager, it_admin, security_admin, 或 group:xxx / dept_manager:ID / amount_based:...
	Name         string `json:"name"`                    // 步骤名称
	IsRequired   bool   `json:"is_required"`             // 是否必需
	ApprovalType string `json:"approval_type,omitempty"` // serial: 顺序审批, parallel/all: 会签(AND), or: 或签(OR,默认阈值1)
	Threshold    int    `json:"threshold,omitempty"`     // 会签投票阈值（默认等于审批人数）
	// 兜底策略：当本层解析不到任何审批人且 IsRequired=true 时触发。
	// block(默认,失败关闭): 阻塞待人工介入; auto_approve: 自动通过本层;
	// escalate: 升级到 FallbackApproverID / FallbackRole; auto_reject: 自动拒绝。
	FallbackAction     string `json:"fallback_action,omitempty"`
	FallbackApproverID int    `json:"fallback_approver_id,omitempty"`
	FallbackRole       string `json:"fallback_role,omitempty"`
}

// Edges of the ApprovalChain.
func (ApprovalChain) Edges() []ent.Edge {
	return nil
}

// Indexes of the ApprovalChain.
func (ApprovalChain) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "entity_type"),
		index.Fields("status"),
	}
}

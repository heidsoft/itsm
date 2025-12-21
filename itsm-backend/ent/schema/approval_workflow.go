package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ApprovalWorkflow holds the schema definition for the ApprovalWorkflow entity.
type ApprovalWorkflow struct {
	ent.Schema
}

// Fields of the ApprovalWorkflow.
func (ApprovalWorkflow) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("工作流名称").
			NotEmpty(),
		field.Text("description").
			Comment("工作流描述").
			Optional(),
		field.String("ticket_type").
			Comment("工单类型").
			Optional(),
		field.String("priority").
			Comment("优先级").
			Optional(),
		field.JSON("nodes", []map[string]interface{}{}).
			Comment("审批节点配置").
			Default([]map[string]interface{}{}),
		field.Bool("is_active").
			Comment("是否启用").
			Default(true),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the ApprovalWorkflow.
func (ApprovalWorkflow) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("approval_records", ApprovalRecord.Type).
			Comment("审批记录"),
	}
}

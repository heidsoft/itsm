package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// TicketTemplate holds the schema definition for the TicketTemplate entity.
type TicketTemplate struct {
	ent.Schema
}

// Fields of the TicketTemplate.
func (TicketTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("模板名称").
			NotEmpty(),
		field.Text("description").
			Comment("模板描述").
			Optional(),
		field.String("category").
			Comment("工单分类").
			NotEmpty(),
		field.String("priority").
			Comment("默认优先级").
			Default("medium"),
		field.JSON("form_fields", []byte{}).
			Comment("表单字段定义").
			Optional(),
		field.JSON("workflow_steps", []byte{}).
			Comment("工作流步骤定义").
			Optional(),
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

// Edges of the TicketTemplate.
func (TicketTemplate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("tickets", Ticket.Type).
			Comment("使用此模板的工单"),
	}
}

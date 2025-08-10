package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/edge"
)

// TicketTag holds the schema definition for the TicketTag entity.
type TicketTag struct {
	ent.Schema
}

// Fields of the TicketTag.
func (TicketTag) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("标签名称").
			NotEmpty(),
		field.String("color").
			Comment("标签颜色").
			Default("#1890ff"),
		field.Text("description").
			Comment("标签描述").
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

// Edges of the TicketTag.
func (TicketTag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("tickets", Ticket.Type).
			Comment("使用此标签的工单"),
	}
}

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// TicketCC holds the schema definition for the TicketCC entity.
type TicketCC struct {
	ent.Schema
}

// Fields of the TicketCC.
func (TicketCC) Fields() []ent.Field {
	return []ent.Field{
		field.Int("ticket_id").
			Comment("工单ID"),
		field.Int("user_id").
			Comment("抄送用户ID"),
		field.Int("added_by").
			Comment("添加人ID"),
		field.Int("tenant_id").
			Comment("租户ID"),
		field.Time("added_at").
			Default(time.Now).
			Comment("添加时间"),
		field.Bool("is_active").
			Default(true).
			Comment("是否有效"),
	}
}

// Edges of the TicketCC.
func (TicketCC) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("cc_users").
			Field("ticket_id").
			Unique().
			Required().
			Comment("所属工单"),
	}
}

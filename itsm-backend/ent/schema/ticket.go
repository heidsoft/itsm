package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Ticket holds the schema definition for the Ticket entity.
type Ticket struct {
	ent.Schema
}

// Fields of the Ticket.
func (Ticket) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("工单标题").
			NotEmpty(),
		field.Text("description").
			Comment("工单描述").
			Optional(),
		field.String("status").
			Comment("状态").
			Default("open"),
		field.String("priority").
			Comment("优先级").
			Default("medium"),
		field.String("ticket_number").
			Comment("工单编号").
			Unique().
			NotEmpty(),
		field.Int("requester_id").
			Comment("申请人ID").
			Positive(),
		field.Int("assignee_id").
			Comment("处理人ID").
			Optional(),
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

// Edges of the Ticket.
func (Ticket) Edges() []ent.Edge {
	return nil
}

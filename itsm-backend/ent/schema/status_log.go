package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// StatusLog holds the schema definition for the StatusLog entity.
type StatusLog struct {
	ent.Schema
}

// Fields of the StatusLog.
func (StatusLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("ticket_id").
			Positive().
			Comment("工单ID"),
		field.String("from_status").
			MaxLen(50).
			Comment("原状态"),
		field.String("to_status").
			MaxLen(50).
			Comment("目标状态"),
		field.Int("user_id").
			Positive().
			Comment("操作用户ID"),
		field.Text("reason").
			Optional().
			Comment("变更原因"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
	}
}

// Edges of the StatusLog.
func (StatusLog) Edges() []ent.Edge {
	return []ent.Edge{
		// 关联工单
		edge.From("ticket", Ticket.Type).
			Ref("status_logs").
			Field("ticket_id").
			Required().
			Unique(),
		// 关联操作用户
		edge.From("user", User.Type).
			Ref("status_logs").
			Field("user_id").
			Required().
			Unique(),
	}
}

// Indexes of the StatusLog.
func (StatusLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ticket_id"),
		index.Fields("user_id"),
		index.Fields("created_at"),
		index.Fields("ticket_id", "created_at"),
	}
}
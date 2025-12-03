package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// TicketNotification holds the schema definition for the TicketNotification entity.
type TicketNotification struct {
	ent.Schema
}

// Fields of the TicketNotification.
func (TicketNotification) Fields() []ent.Field {
	return []ent.Field{
		field.Int("ticket_id").
			Comment("工单ID").
			Positive(),
		field.Int("user_id").
			Comment("接收人ID").
			Positive(),
		field.String("type").
			Comment("通知类型: created, assigned, status_changed, commented, sla_warning, resolved, closed").
			NotEmpty(),
		field.String("channel").
			Comment("通知渠道: email, in_app, sms").
			Default("in_app"),
		field.Text("content").
			Comment("通知内容").
			NotEmpty(),
		field.Time("sent_at").
			Comment("发送时间").
			Optional(),
		field.Time("read_at").
			Comment("阅读时间").
			Optional(),
		field.String("status").
			Comment("状态: pending, sent, read").
			Default("pending"),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
	}
}

// Edges of the TicketNotification.
func (TicketNotification) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("notifications").
			Field("ticket_id").
			Required().
			Unique().
			Comment("所属工单"),
		edge.From("user", User.Type).
			Ref("ticket_notifications").
			Field("user_id").
			Required().
			Unique().
			Comment("接收人"),
	}
}


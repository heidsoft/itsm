package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// TicketComment holds the schema definition for the TicketComment entity.
type TicketComment struct {
	ent.Schema
}

// Fields of the TicketComment.
func (TicketComment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("ticket_id").
			Comment("工单ID").
			Positive(),
		field.Int("user_id").
			Comment("评论人ID").
			Positive(),
		field.Text("content").
			Comment("评论内容").
			NotEmpty(),
		field.Bool("is_internal").
			Comment("是否内部备注").
			Default(false),
		field.JSON("mentions", []int{}).
			Comment("@的用户ID列表").
			Optional(),
		field.JSON("attachments", []int{}).
			Comment("附件ID列表").
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

// Edges of the TicketComment.
func (TicketComment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("comments").
			Field("ticket_id").
			Required().
			Unique().
			Comment("所属工单"),
		edge.From("user", User.Type).
			Ref("ticket_comments").
			Field("user_id").
			Required().
			Unique().
			Comment("评论人"),
	}
}


package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type EmailOutboundMessage struct{ ent.Schema }

func (EmailOutboundMessage) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.Int("conversation_id").Positive(),
		field.String("mailbox_instance_key").NotEmpty().MaxLen(160),
		field.String("reply_type").NotEmpty().MaxLen(40),
		field.Int("revision").Positive(),
		field.String("to_address").NotEmpty().MaxLen(320),
		field.String("subject").NotEmpty().MaxLen(998),
		field.Text("body_text"),
		field.String("in_reply_to").Optional().MaxLen(512),
		field.JSON("references", []string{}).Optional(),
		field.String("status").Default("PENDING").MaxLen(40),
		field.Int("attempts").Default(0).NonNegative(),
		field.String("last_error").Optional().MaxLen(2000),
		field.Time("sent_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (EmailOutboundMessage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("conversation", EmailConversation.Type).Ref("outbound_messages").Field("conversation_id").Unique().Required(),
	}
}

func (EmailOutboundMessage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "conversation_id", "reply_type", "revision").Unique(),
		index.Fields("tenant_id", "status", "created_at"),
	}
}

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type InboundEmailMessage struct{ ent.Schema }

func (InboundEmailMessage) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.Int("conversation_id").Positive(),
		field.String("provider").Default("imap").MaxLen(40),
		field.String("mailbox_instance_key").NotEmpty().MaxLen(160),
		field.Uint64("uid_validity"),
		field.Uint64("uid"),
		field.String("external_message_id").Optional().MaxLen(512),
		field.String("in_reply_to").Optional().MaxLen(512),
		field.JSON("references", []string{}).Optional(),
		field.String("from_address").NotEmpty().MaxLen(320),
		field.JSON("to_addresses", []string{}).Optional(),
		field.String("reply_to_address").Optional().MaxLen(320),
		field.String("subject").Optional().MaxLen(998),
		field.Text("plain_text").Optional(),
		field.Text("sanitized_html").Optional(),
		field.Bytes("raw_mime").Optional(),
		field.String("raw_sha256").NotEmpty().MaxLen(64),
		field.JSON("attachment_metadata", []map[string]interface{}{}).Optional(),
		field.String("processing_status").Default("RECEIVED").MaxLen(40),
		field.String("last_error").Optional().MaxLen(2000),
		field.Time("received_at"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (InboundEmailMessage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("conversation", EmailConversation.Type).Ref("messages").Field("conversation_id").Unique().Required(),
		edge.To("analyses", EmailIntakeAnalysis.Type),
	}
}

func (InboundEmailMessage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "mailbox_instance_key", "uid_validity", "uid").Unique(),
		index.Fields("tenant_id", "external_message_id"),
		index.Fields("tenant_id", "processing_status", "received_at"),
	}
}

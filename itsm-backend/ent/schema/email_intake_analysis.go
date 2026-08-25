package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type EmailIntakeAnalysis struct{ ent.Schema }

func (EmailIntakeAnalysis) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.Int("conversation_id").Positive(),
		field.Int("message_id").Positive(),
		field.String("provider").Optional().MaxLen(80),
		field.String("model").Optional().MaxLen(160),
		field.String("prompt_version").NotEmpty().MaxLen(80),
		field.Text("raw_result").Optional(),
		field.JSON("result", map[string]interface{}{}).Optional(),
		field.Float("confidence").Default(0),
		field.Int64("latency_ms").Default(0).NonNegative(),
		field.String("status").Default("pending").MaxLen(40),
		field.String("validation_error").Optional().MaxLen(2000),
		field.Int("reviewed_by").Optional().Nillable(),
		field.JSON("corrections", map[string]interface{}{}).Optional(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (EmailIntakeAnalysis) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("conversation", EmailConversation.Type).Ref("analyses").Field("conversation_id").Unique().Required(),
		edge.From("message", InboundEmailMessage.Type).Ref("analyses").Field("message_id").Unique().Required(),
	}
}

func (EmailIntakeAnalysis) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "message_id"),
		index.Fields("tenant_id", "status", "created_at"),
	}
}

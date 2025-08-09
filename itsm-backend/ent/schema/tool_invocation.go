package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ToolInvocation stores each tool call during a conversation.
type ToolInvocation struct{ ent.Schema }

func (ToolInvocation) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Default(time.Now),
		field.Int("conversation_id"),
		field.String("tool_name"),
		field.Text("arguments").Default(""),
		field.Text("result").Optional().Nillable(),
		field.String("status").Default("success"),
		field.String("request_id").Optional(),
		field.Bool("needs_approval").Default(false),
		field.String("approval_state").Default("none"), // none|pending|approved|rejected
		field.String("approval_reason").Default(""),
		field.Int("approved_by").Optional(),
		field.Time("approved_at").Optional(),
		field.Bool("dry_run").Default(false),
		field.Text("error").Optional().Nillable(),
	}
}

func (ToolInvocation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("conversation", Conversation.Type).Ref("tool_invocations").Unique().Field("conversation_id").Required(),
	}
}

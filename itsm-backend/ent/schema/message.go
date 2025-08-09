package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Message holds the schema definition for the Message entity.
type Message struct{ ent.Schema }

func (Message) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Default(time.Now),
		field.Int("conversation_id"),
		field.String("role"), // user/assistant/system/tool
		field.Text("content").Default(""),
		field.String("request_id").Optional(),
	}
}

func (Message) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("conversation", Conversation.Type).Ref("messages").Unique().Field("conversation_id").Required(),
	}
}

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// PromptTemplate defines prompt templates for AI orchestration.
type PromptTemplate struct{ ent.Schema }

func (PromptTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.String("name").Unique(),
		field.String("version").Default("v1"),
		field.Text("template"),
		field.String("description").Default(""),
		field.JSON("metadata", map[string]any{}).Optional(),
	}
}

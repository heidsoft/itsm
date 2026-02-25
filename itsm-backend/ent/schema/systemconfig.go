package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// SystemConfig holds the schema definition for the SystemConfig entity.
type SystemConfig struct {
	ent.Schema
}

// Fields of the SystemConfig.
func (SystemConfig) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").NotEmpty().Unique(),
		field.String("value").Optional(),
		field.String("value_type").Default("string"), // string, number, boolean, json
		field.String("category").Default("general"),
		field.String("description").Optional(),
		field.String("created_by").Optional(),
		field.Int("tenant_id").Optional(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the SystemConfig.
func (SystemConfig) Edges() []ent.Edge {
	return nil
}

// Indexes of the SystemConfig.
func (SystemConfig) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "category"),
	}
}

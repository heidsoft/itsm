package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Alert stores normalized alerts received from external monitoring sources.
type Alert struct{ ent.Schema }

func (Alert) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.String("source").NotEmpty().MaxLen(100),
		field.String("external_alert_id").NotEmpty().MaxLen(255),
		field.String("source_raw").Default("").MaxLen(100),
		field.String("name").NotEmpty().MaxLen(500),
		field.Text("description").Default(""),
		field.String("severity").NotEmpty().MaxLen(20),
		field.String("status").NotEmpty().MaxLen(40),
		field.JSON("labels", map[string]string{}).Optional(),
		field.JSON("annotations", map[string]string{}).Optional(),
		field.String("source_ip").Default("").MaxLen(255),
		field.String("service").Default("").MaxLen(255),
		field.JSON("tags", []string{}).Optional(),
		field.Time("fired_at"),
		field.Time("acknowledged_at").Optional().Nillable(),
		field.Time("resolved_at").Optional().Nillable(),
		field.JSON("raw_payload", map[string]interface{}{}).Optional(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Alert) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "source", "external_alert_id").Unique(),
	}
}

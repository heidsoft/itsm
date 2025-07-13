package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// CIChangeRecord CI变更记录
type CIChangeRecord struct {
	ent.Schema
}

func (CIChangeRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int("ci_id"),
		field.String("change_type").NotEmpty(), // create, update, delete, relationship_add, relationship_remove
		field.JSON("old_values", map[string]interface{}{}).Optional(),
		field.JSON("new_values", map[string]interface{}{}).Optional(),
		field.String("changed_by").NotEmpty(),
		field.String("change_source").Default("manual"), // manual, discovery, api, import
		field.Text("reason").Optional(),
		field.String("version_before").Optional(),
		field.String("version_after").Optional(),
		field.Int("tenant_id"),
		field.Time("created_at").Default(time.Now),
	}
}

func (CIChangeRecord) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("tenant", Tenant.Type).Ref("ci_change_records").Field("tenant_id").Unique().Required(),
		edge.From("configuration_item", ConfigurationItem.Type).Ref("change_records").Field("ci_id").Unique().Required(),
	}
}

func (CIChangeRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("ci_id"),
		index.Fields("change_type"),
		index.Fields("changed_by"),
		index.Fields("created_at"),
	}
}

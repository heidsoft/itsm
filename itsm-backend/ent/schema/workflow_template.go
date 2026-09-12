package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WorkflowTemplate is the tenant-visible business workflow catalog entry.
// JSON sections are versioned with the template so designer drafts remain
// reproducible and can be linted before publication.
type WorkflowTemplate struct{ ent.Schema }

func (WorkflowTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").NotEmpty(), field.String("name").NotEmpty(),
		field.Text("description").Optional(), field.String("domain").Default("it"),
		field.JSON("form_schema", map[string]interface{}{}),
		field.JSON("approval_policy", map[string]interface{}{}),
		field.JSON("ontology_bindings", map[string]interface{}{}),
		field.JSON("sla_config", map[string]interface{}{}),
		field.JSON("bpmn_xml", []byte{}).Optional(),
		field.String("version").Default("1.0.0"), field.String("status").Default("draft"),
		field.Bool("is_public").Default(false), field.Int("tenant_id").Positive(),
		field.Int("created_by").Positive(), field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (WorkflowTemplate) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id", "key", "version").Unique(), index.Fields("tenant_id", "domain", "status"), index.Fields("tenant_id", "is_public")}
}

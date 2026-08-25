package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type SourceOrganization struct{ ent.Schema }

func (SourceOrganization) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.String("name").NotEmpty().MaxLen(255),
		field.String("normalized_name").NotEmpty().MaxLen(255),
		field.JSON("email_addresses", []string{}).Optional(),
		field.JSON("email_domains", []string{}).Optional(),
		field.String("status").Default("active").MaxLen(32),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (SourceOrganization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("external_contract_references", ExternalContractReference.Type),
		edge.To("conversations", EmailConversation.Type),
	}
}

func (SourceOrganization) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "normalized_name").Unique(),
		index.Fields("tenant_id", "status"),
	}
}

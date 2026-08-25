package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ExternalContractReference struct{ ent.Schema }

func (ExternalContractReference) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.Int("source_organization_id").Positive(),
		field.Int("support_contract_id").Positive(),
		field.Int("customer_id").Positive(),
		field.Int("branch_id").Optional().Nillable(),
		field.String("external_contract_number").NotEmpty().MaxLen(160),
		field.String("normalized_external_contract_number").NotEmpty().MaxLen(160),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ExternalContractReference) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("source_organization", SourceOrganization.Type).Ref("external_contract_references").Field("source_organization_id").Unique().Required(),
		edge.From("support_contract", SupportContract.Type).Ref("external_references").Field("support_contract_id").Unique().Required(),
	}
}

func (ExternalContractReference) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "source_organization_id", "normalized_external_contract_number").Unique(),
		index.Fields("tenant_id", "support_contract_id"),
	}
}

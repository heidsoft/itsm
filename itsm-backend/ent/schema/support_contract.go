package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type SupportContract struct{ ent.Schema }

func (SupportContract) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.Int("customer_id").Positive(),
		field.Int("branch_id").Optional().Nillable(),
		field.String("contract_number").NotEmpty().MaxLen(160),
		field.String("normalized_contract_number").NotEmpty().MaxLen(160),
		field.String("status").Default("active").MaxLen(32),
		field.Time("start_at").Optional().Nillable(),
		field.Time("end_at").Optional().Nillable(),
		field.Int("source_document_id").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (SupportContract) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("customer", ServiceCustomer.Type).Ref("contracts").Field("customer_id").Unique().Required(),
		edge.From("branch", CustomerBranch.Type).Ref("contracts").Field("branch_id").Unique(),
		edge.To("external_references", ExternalContractReference.Type),
		edge.To("conversations", EmailConversation.Type),
	}
}

func (SupportContract) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "normalized_contract_number").Unique(),
		index.Fields("tenant_id", "customer_id", "status"),
	}
}

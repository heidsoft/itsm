package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ServiceCustomer is a tenant-owned customer served by a NOC/MSP.
type ServiceCustomer struct{ ent.Schema }

func (ServiceCustomer) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.String("name").NotEmpty().MaxLen(255),
		field.String("normalized_name").NotEmpty().MaxLen(255),
		field.String("short_name").Optional().MaxLen(120),
		field.JSON("aliases", []string{}).Optional(),
		field.JSON("historical_names", []string{}).Optional(),
		field.String("status").Default("active").MaxLen(32),
		field.Int("linked_customer_tenant_id").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ServiceCustomer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("branches", CustomerBranch.Type),
		edge.To("contracts", SupportContract.Type),
		edge.To("conversations", EmailConversation.Type),
	}
}

func (ServiceCustomer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "normalized_name").Unique(),
		index.Fields("tenant_id", "status"),
	}
}

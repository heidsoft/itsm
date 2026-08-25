package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type CustomerBranch struct{ ent.Schema }

func (CustomerBranch) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.Int("customer_id").Positive(),
		field.String("name").NotEmpty().MaxLen(255),
		field.String("normalized_name").NotEmpty().MaxLen(255),
		field.JSON("aliases", []string{}).Optional(),
		field.String("status").Default("active").MaxLen(32),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (CustomerBranch) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("customer", ServiceCustomer.Type).Ref("branches").Field("customer_id").Unique().Required(),
		edge.To("contracts", SupportContract.Type),
		edge.To("conversations", EmailConversation.Type),
	}
}

func (CustomerBranch) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "customer_id", "normalized_name").Unique(),
		index.Fields("tenant_id", "customer_id", "status"),
	}
}

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type EmailConversation struct{ ent.Schema }

func (EmailConversation) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.String("external_thread_id").Optional().MaxLen(512),
		field.String("conversation_token").NotEmpty().MaxLen(120),
		field.Int("source_organization_id").Optional().Nillable(),
		field.Int("customer_id").Optional().Nillable(),
		field.Int("branch_id").Optional().Nillable(),
		field.Int("support_contract_id").Optional().Nillable(),
		field.String("status").Default("PROCESSING").MaxLen(40),
		field.JSON("canonical_data", map[string]interface{}{}).Optional(),
		field.JSON("field_sources", map[string]interface{}{}).Optional(),
		field.JSON("missing_fields", []string{}).Optional(),
		field.Float("confidence").Default(0),
		field.Int("version").Default(1).Positive(),
		field.Time("last_message_at").Default(time.Now),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (EmailConversation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("source_organization", SourceOrganization.Type).Ref("conversations").Field("source_organization_id").Unique(),
		edge.From("customer", ServiceCustomer.Type).Ref("conversations").Field("customer_id").Unique(),
		edge.From("branch", CustomerBranch.Type).Ref("conversations").Field("branch_id").Unique(),
		edge.From("support_contract", SupportContract.Type).Ref("conversations").Field("support_contract_id").Unique(),
		edge.To("messages", InboundEmailMessage.Type),
		edge.To("analyses", EmailIntakeAnalysis.Type),
		edge.To("outbound_messages", EmailOutboundMessage.Type),
		edge.To("incidents", Incident.Type),
	}
}

func (EmailConversation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "conversation_token").Unique(),
		index.Fields("tenant_id", "external_thread_id"),
		index.Fields("tenant_id", "status", "last_message_at"),
	}
}

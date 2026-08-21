package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TicketType holds the schema definition for the TicketType entity.
type TicketType struct {
	ent.Schema
}

// Fields of the TicketType.
func (TicketType) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").NotEmpty().MaxLen(50),
		field.String("name").NotEmpty().MaxLen(100),
		field.Text("description"),
		field.String("icon").MaxLen(50),
		field.String("color").MaxLen(20),
		field.String("status").Default("active"),
		field.Int("category_id").Optional(),
		field.String("default_priority").Default("medium").MaxLen(20),
		field.Int("sort_order").Default(0),
		field.String("workflow_definition_key").Optional().MaxLen(128),
		field.Int("assignment_rule_id").Optional(),
		field.Time("archived_at").Optional().Nillable(),
		field.Int64("archived_by").Optional(),
		field.JSON("custom_fields", map[string]interface{}{}),
		field.Bool("approval_enabled").Default(false),
		field.Int64("approval_workflow_id").Optional(),
		field.JSON("approval_chain", []interface{}{}),
		field.Bool("sla_enabled").Default(false),
		field.Int64("default_sla_id").Optional(),
		field.Bool("auto_assign_enabled").Default(false),
		field.JSON("assignment_rules", []interface{}{}),
		field.JSON("notification_config", map[string]interface{}{}),
		field.JSON("permission_config", map[string]interface{}{}),
		field.Int64("tenant_id"),
		field.Int64("created_by"),
		field.Time("created_at"),
		field.Time("updated_at"),
		field.Int64("updated_by").Optional(),
		field.Int("usage_count").Default(0),
	}
}

// Edges of the TicketType.
func (TicketType) Edges() []ent.Edge {
	return []ent.Edge{edge.To("tickets", Ticket.Type)}
}

// Indexes of the TicketType.
func (TicketType) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code", "tenant_id").Unique(),
		index.Fields("tenant_id"),
		index.Fields("status"),
		index.Fields("tenant_id", "sort_order"),
	}
}

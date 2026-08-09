package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// OperationalCommand is the durable outbox for cross-domain side effects.
type OperationalCommand struct{ ent.Schema }

func (OperationalCommand) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.String("command_type").NotEmpty().MaxLen(100),
		field.String("aggregate_type").NotEmpty().MaxLen(50),
		field.Int("aggregate_id").Positive(),
		field.String("idempotency_key").NotEmpty().MaxLen(200),
		field.JSON("payload", map[string]interface{}{}).Optional(),
		field.String("status").Default("pending").MaxLen(32),
		field.Int("attempt").Default(0).NonNegative(),
		field.Int("max_attempts").Default(8).Positive(),
		field.Time("available_at").Default(time.Now),
		field.String("lease_owner").Optional().MaxLen(200),
		field.Time("lease_expires_at").Optional().Nillable(),
		field.Int64("fencing_token").Default(0).NonNegative(),
		field.String("last_error").Optional().MaxLen(2000),
		field.Time("completed_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (OperationalCommand) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "command_type", "idempotency_key").Unique(),
		index.Fields("status", "available_at"),
		index.Fields("status", "lease_expires_at"),
		index.Fields("tenant_id", "aggregate_type", "aggregate_id"),
	}
}
